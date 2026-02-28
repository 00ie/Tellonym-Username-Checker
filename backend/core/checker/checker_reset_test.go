package checker

import (
	"sync/atomic"
	"testing"
	"time"

	"tellonym-checker/backend/utils/logger"
)

func TestResetStatsRequiresStoppedChecker(t *testing.T) {
	c := NewChecker(&Config{}, logger.NewLogger(logger.Config{Level: "error"}), nil)

	c.mu.Lock()
	c.isRunning = true
	c.mu.Unlock()

	if err := c.ResetStats(); err == nil {
		t.Fatalf("expected error when checker is running")
	}
}

func TestResetStatsClearsCounters(t *testing.T) {
	c := NewChecker(&Config{}, logger.NewLogger(logger.Config{Level: "error"}), nil)

	atomic.StoreUint64(&c.attempts, 10)
	atomic.StoreUint64(&c.found, 2)
	atomic.StoreUint64(&c.errors, 3)
	atomic.StoreUint64(&c.rateLimited, 4)
	atomic.StoreUint64(&c.responseTotal, uint64((10 * time.Millisecond).Nanoseconds()))
	atomic.StoreUint64(&c.responseCount, 5)
	c.stats = Stats{
		Rate:        15.0,
		StartTime:   time.Now().Add(-time.Minute),
		ElapsedTime: 15 * time.Second,
		IsRunning:   false,
		IsPaused:    false,
	}

	if err := c.ResetStats(); err != nil {
		t.Fatalf("reset stats failed: %v", err)
	}

	if got := atomic.LoadUint64(&c.attempts); got != 0 {
		t.Fatalf("expected attempts=0, got %d", got)
	}
	if got := atomic.LoadUint64(&c.found); got != 0 {
		t.Fatalf("expected found=0, got %d", got)
	}
	if got := atomic.LoadUint64(&c.errors); got != 0 {
		t.Fatalf("expected errors=0, got %d", got)
	}
	if got := atomic.LoadUint64(&c.rateLimited); got != 0 {
		t.Fatalf("expected rateLimited=0, got %d", got)
	}
	if got := atomic.LoadUint64(&c.responseTotal); got != 0 {
		t.Fatalf("expected responseTotal=0, got %d", got)
	}
	if got := atomic.LoadUint64(&c.responseCount); got != 0 {
		t.Fatalf("expected responseCount=0, got %d", got)
	}
	if c.stats.Rate != 0 || !c.stats.StartTime.IsZero() || c.stats.ElapsedTime != 0 {
		t.Fatalf("expected checker stats struct to be zeroed")
	}
}
