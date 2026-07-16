// Package stats tracks key generation statistics and sends periodic reports.
package stats

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Tracker holds atomic counters for key generation progress.
type Tracker struct {
	TotalKeys   uint64 // total keys processed
	TotalMatch  uint64 // total vanity addresses found
	LastRate    uint64 // keys/second in the last interval
	startTime   time.Time
	lastTime    time.Time
	lastKeys    uint64
}

// NewTracker creates a stats tracker.
func NewTracker() *Tracker {
	now := time.Now()
	return &Tracker{
		startTime: now,
		lastTime:  now,
	}
}

// AddKeys atomically adds to the key counter.
func (t *Tracker) AddKeys(n uint64) {
	atomic.AddUint64(&t.TotalKeys, n)
}

// AddMatch atomically adds to the match counter.
func (t *Tracker) AddMatch() {
	atomic.AddUint64(&t.TotalMatch, 1)
}

// Snapshot returns a point-in-time reading of all stats.
func (t *Tracker) Snapshot() (totalKeys, totalMatch, keysPerSec uint64, elapsed time.Duration) {
	now := time.Now()
	totalKeys = atomic.LoadUint64(&t.TotalKeys)
	totalMatch = atomic.LoadUint64(&t.TotalMatch)
	elapsed = now.Sub(t.startTime)

	// Compute instantaneous rate
	deltaKeys := totalKeys - t.lastKeys
	deltaTime := now.Sub(t.lastTime)
	if deltaTime > 0 {
		keysPerSec = uint64(float64(deltaKeys) / deltaTime.Seconds())
	}
	t.lastKeys = totalKeys
	t.lastTime = now

	return
}

// FormatRate formats a keys/sec value into a human-readable string.
func FormatRate(rate uint64) string {
	switch {
	case rate >= 1_000_000:
		return fmt.Sprintf("%.2f M/s", float64(rate)/1_000_000)
	case rate >= 1_000:
		return fmt.Sprintf("%.1f K/s", float64(rate)/1_000)
	default:
		return fmt.Sprintf("%d/s", rate)
	}
}

// FormatDuration formats a duration into a compact string.
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ReportMessage builds a human-readable stats message for Telegram.
func (t *Tracker) ReportMessage() string {
	totalKeys, totalMatch, rate, elapsed := t.Snapshot()

	return fmt.Sprintf(
		"📊 TRON Vanity Generator 状态报告\n\n"+
			"⏱  运行时间: %s\n"+
			"🔑 已生成密钥: %d\n"+
			"✅ 发现靓号: %d\n"+
			"⚡ 当前速率: %s\n",
		FormatDuration(elapsed),
		totalKeys,
		totalMatch,
		FormatRate(rate),
	)
}
