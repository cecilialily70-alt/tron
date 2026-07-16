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
	"golang.org/x/crypto/sha3"
)

const (
	defaultToken = "8611216521:AAGXFb_Popymx2FAi3T7VCXKOX64LRmFxHY"
	defaultChat  = "8500753537"
	recordSize   = 96 // 关键！适配 GPU 发来的 96 字节
	readChunk    = 96 * 1024
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
	if err != nil { log.Fatalf("pipe: %v", err) }
	if err := cmd.Start(); err != nil { log.Fatalf("start GPU: %v", err) }
	log.Printf("[GO] 架构 v10 | CPU Cores: %d | Batch: %d", numW, *batchSize)
	sendStartup(tg, numW, *batchSize)

	// Log every verification failure (GPU output didn't match trusted derivation)
	checker.VerifyFailHook = func(addr string) {
		log.Printf("[WARN] 验签未通过! 地址 %s 的私钥不匹配，已丢弃", addr)
	}

	var wg sync.WaitGroup
	pipeData := make(chan []byte, 128)

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
			if err != nil { return }
		}
	}()

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k := sha3.NewLegacyKeccak256() // 彻底抛弃 CPU 的 secp256k1，只做极速哈希
			for buf := range pipeData {
				n := len(buf) / recordSize
				for j := 0; j < n; j++ {
					record := buf[j*recordSize : (j+1)*recordSize]
					privKey := record[:32]
					pubKeyXY := record[32:96]

					k.Reset()
					k.Write(pubKeyXY)
					hash20 := k.Sum(nil)[12:32]

					if match := checker.CheckFull(privKey, hash20); match != nil {
						st.AddMatch()
						typeLabel := map[checker.MatchType]string{checker.Suffix3: "后3位相同", checker.Prefix3: "前3位相同"}
						log.Printf("[GO] VERIFIED + 私钥已校验! %s (%s '%c')", match.Address, typeLabel[match.Type], match.Pattern)
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
			case <-ctx.Done(): return
			case m := <-matchCh:
				typeLabel := map[checker.MatchType]string{checker.Suffix3: "后3位相同", checker.Prefix3: "前3位相同"}
				msg := fmt.Sprintf("🎯 TRON 靓号 (3位验证版)!\n\n✅ 地址: `%s`\n🔑 私钥: `%s`\n📌 模式: %s '%c'\n🔒 私钥已校验，地址匹配", m.Address, m.PrivateKey, typeLabel[m.Type], m.Pattern)
				tg.SendMessage(msg)
			case <-statTicker.C:
				totalKeys, totalMatch, rate, _ := st.Snapshot()
				log.Printf("[STATS] 已处理: %d 个密钥 | 命中: %d | 速率: %s", totalKeys, totalMatch, stats.FormatRate(rate))
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
	msg := fmt.Sprintf("🚀 TRON 3位靓号生成器 v10\n\n🎯 目标: 前3位/后3位相同 (3位数靓号)\n🖥  Workers: %d | GPU Batch: %d\n🔒 私钥校验: 强制验证(secp256k1)\n⚡ 计算模式: GPU推导 + CPU验签", workers, batch)
	tg.SendMessage(msg)
}