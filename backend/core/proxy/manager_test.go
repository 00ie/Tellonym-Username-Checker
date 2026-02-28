package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tellonym-checker/backend/utils/logger"
)

func TestRemoveDeadProxiesPersistsFile(t *testing.T) {
	tempDir := t.TempDir()
	proxyFile := filepath.Join(tempDir, "proxies.txt")

	cfg := &Config{
		Enabled:             true,
		Types:               []string{"*"},
		MaxConsecutiveFails: 3,
		HealthCheckInterval: time.Hour,
		ValidationURL:       "https://httpbin.org/ip",
		ValidationTimeout:   time.Second,
	}

	log := logger.NewLogger(logger.Config{Level: "error", Encoding: "console"})
	defer log.Sync()

	m := NewManager(cfg, log)
	m.sourceFile = proxyFile

	proxyAURL, err := ParseProxy("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("parse proxy A failed: %v", err)
	}
	proxyBURL, err := ParseProxy("http://127.0.0.2:8080")
	if err != nil {
		t.Fatalf("parse proxy B failed: %v", err)
	}

	proxyA := &Proxy{URL: proxyAURL, Type: "http", IsAlive: false}
	proxyB := &Proxy{URL: proxyBURL, Type: "http", IsAlive: true}
	atomic.StoreInt32(&proxyA.FailCount, 5)

	m.proxies = []*Proxy{proxyA, proxyB}
	m.rebuildHealthyListLocked()

	removed := m.RemoveDeadProxies()
	if removed != 1 {
		t.Fatalf("expected 1 removed proxy, got %d", removed)
	}

	data, err := os.ReadFile(proxyFile)
	if err != nil {
		t.Fatalf("read persisted file failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 proxy line, got %d", len(lines))
	}
	if lines[0] != "http://127.0.0.2:8080" {
		t.Fatalf("unexpected proxy persisted: %s", lines[0])
	}
}
