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
	recordSize   = 32 // GPU only outputs 32-byte private keys (no secp256k1)
	readChunk    = 32 * 1024
)

func main() {
	botToken := flag.String("token", defaultToken, "Telegram Bot Token")
	chatID := flag.String("chat", defaultChat, "Telegram Chat ID")
	gpuBinary := flag.String("gpu", "./gpu/vanity_worker", "CUDA binary path")
	batchSize := flag.Int("batch", 1048576, "GPU batch size")
	flag.Parse()

	numW := runtime.NumCPU()
	if numW > 48 {
		numW = 48
	}
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
	log.Printf("[GO] v11 全受信架构 | CPU: %d cores | Batch: %d | GPU只出随机私钥", numW, *batchSize)
	sendStartup(tg, numW, *batchSize)

	var wg sync.WaitGroup
	pipeData := make(chan []byte, 128)

	// GPU reader — feeds raw 32-byte private key chunks to workers
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(pipeData)
		br := bufio.NewReaderSize(stdout, 4<<20)
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

	// Worker pool: each goroutine does full derivation (secp256k1 + Keccak256 + base58)
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
						typeLabel := map[checker.MatchType]string{checker.Suffix3: "后3位相同", checker.Prefix3: "前3位相同"}
						log.Printf("[MATCH] %s (%s '%c')", match.Address, typeLabel[match.Type], match.Pattern)
						matchCh <- match
					}
				}
			}
		}()
	}

	// Reporter: stats every 10s, matches instantly to Telegram, status every 30min
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
				typeLabel := map[checker.MatchType]string{checker.Suffix3: "后3位相同", checker.Prefix3: "前3位相同"}
				msg := fmt.Sprintf("🎯 TRON 靓号!\n\n✅ 地址: `%s`\n🔑 私钥: `%s`\n📌 模式: %s '%c'\n🔒 全受信Go加密推导", m.Address, m.PrivateKey, typeLabel[m.Type], m.Pattern)
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
	msg := fmt.Sprintf("🚀 TRON 靓号生成器 v11 (全受信架构)\n\n🎯 目标: 前3位/后3位相同\n🖥  Workers: %d | GPU Batch: %d\n🔒 加密: Go secp256k1 (100%可信)\n⚡ GPU: 只产生随机私钥", workers, batch)
	tg.SendMessage(msg)
}
