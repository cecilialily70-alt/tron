package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"tron-address-generator/checker"
	"tron-address-generator/stats"
	"tron-address-generator/telegram"
)

const (
	defaultToken = "8611216521:AAGXFb_Popymx2FAi3T7VCXKOX64LRmFxHY"
	defaultChat  = "8500753537"
	recordSize   = 32
	readChunk    = 32 * 1024
)

func main() {
	botToken := flag.String("token", defaultToken, "Telegram Bot Token")
	chatID := flag.String("chat", defaultChat, "Telegram Chat ID")
	gpuBinary := flag.String("gpu", "./gpu/vanity_worker", "CUDA binary path")
	batchSize := flag.Int("batch", 67108864, "GPU batch size (64M for RTX 5090)")
	flag.Parse()

	numW := runtime.NumCPU()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	tg := telegram.NewClient(telegram.Config{BotToken: *botToken, ChatID: *chatID})
	st := stats.NewTracker()
	matchCh := make(chan *checker.Match, 64)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.CommandContext(ctx, *gpuBinary, "--batch", fmt.Sprintf("%d", *batchSize))
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("start GPU: %v", err)
	}
	log.Printf("[GO] v15 final | CPU: %d cores | Batch: %d", numW, *batchSize)
	sendStartup(tg, numW, *batchSize)

	var wg sync.WaitGroup
	pipeData := make(chan []byte, 256)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(pipeData)
		br := bufio.NewReaderSize(stdout, 8<<20)
		for {
			buf := make([]byte, readChunk)
			n, err := io.ReadFull(br, buf)
			if n > 0 {
				pipeData <- buf[:n]
				st.AddKeys(uint64(n / recordSize))
			}
			if err != nil {
				return
			}
		}
	}()

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for buf := range pipeData {
				n := len(buf) / recordSize
				for j := 0; j < n; j++ {
					privKey := buf[j*recordSize : (j+1)*recordSize]
					if match := checker.Check(privKey); match != nil {
						st.AddMatch()
						typeLabel := map[checker.MatchType]string{checker.Suffix7: "后7位相同", checker.Prefix7: "前7位相同", checker.SixSixes: "连续6个6", checker.SixEights: "连续6个8"}
						log.Printf("[MATCH] %s (%s '%c')", match.Address, typeLabel[match.Type], match.Pattern)
						matchCh <- match
					}
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		statTicker := time.NewTicker(10 * time.Second)
		reportTicker := time.NewTicker(30 * time.Minute)
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-matchCh:
				typeLabel := map[checker.MatchType]string{checker.Suffix7: "后7位相同", checker.Prefix7: "前7位相同", checker.SixSixes: "连续6个6", checker.SixEights: "连续6个8"}
				msg := fmt.Sprintf("%s\n%s\n\n🎯 TRON 靓号 (%s)", m.Address, m.PrivateKey, typeLabel[m.Type])
				tg.SendMessage(msg)
			case <-statTicker.C:
				totalKeys, totalMatch, rate, _ := st.Snapshot()
				log.Printf("[STATS] 已处理: %d | 命中: %d | 速率: %s", totalKeys, totalMatch, stats.FormatRate(rate))
			case <-reportTicker.C:
				tg.SendMessage(st.ReportMessage())
			}
		}
	}()

	<-sigCh
	cancel()
	cmd.Process.Kill()
	wg.Wait()
}

func sendStartup(tg *telegram.Client, workers, batch int) {
	msg := fmt.Sprintf("🚀 TRON 靓号生成器 v15\n\n🎯 目标: 7位相同 / 6个6 / 6个8\n🖥  Workers: %d | GPU Batch: %d\n🔒 加密: libsecp256k1 (Bitcoin C库)", workers, batch)
	tg.SendMessage(msg)
}
