package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tellonym-checker/backend/utils/logger"
)

func TestRemoveProxyPersistsFile(t *testing.T) {
	tempDir := t.TempDir()
	proxyFile := filepath.Join(tempDir, "proxies.txt")

	log := logger.NewLogger(logger.Config{Level: "error", Encoding: "console"})
	defer log.Sync()

	cfg := &Config{
		Enabled: true,
		Types:   []string{"*"},
	}
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

	m.proxies = []*Proxy{
		{URL: proxyAURL, Type: "http", IsAlive: true},
		{URL: proxyBURL, Type: "http", IsAlive: true},
	}
	m.rebuildHealthyListLocked()

	m.RemoveProxy("http://127.0.0.1:8080")

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
