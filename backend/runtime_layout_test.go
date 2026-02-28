package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareRuntimeLayout(t *testing.T) {
	root := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	if err := os.WriteFile(filepath.Join(root, "proxies.txt"), []byte("http://127.0.0.1:8080\n"), 0o644); err != nil {
		t.Fatalf("write source proxies failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir source config dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("app:\n  name: Test\n"), 0o644); err != nil {
		t.Fatalf("write source config failed: %v", err)
	}

	layout, err := prepareRuntimeLayout()
	if err != nil {
		t.Fatalf("prepare runtime layout failed: %v", err)
	}

	requiredDirs := []string{
		layout.RootDir,
		layout.ConfigDir,
		layout.DataDir,
		layout.LogsDir,
		layout.ExportsDir,
	}
	for _, dir := range requiredDirs {
		stat, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("missing dir %s: %v", dir, err)
		}
		if !stat.IsDir() {
			t.Fatalf("path is not dir: %s", dir)
		}
	}

	requiredFiles := []string{
		layout.ProxyFile,
		layout.ConfigFile,
		layout.FoundFile,
	}
	for _, filePath := range requiredFiles {
		stat, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("missing file %s: %v", filePath, err)
		}
		if stat.IsDir() {
			t.Fatalf("path is not file: %s", filePath)
		}
	}

	proxyData, err := os.ReadFile(layout.ProxyFile)
	if err != nil {
		t.Fatalf("read proxy file failed: %v", err)
	}
	if string(proxyData) != "http://127.0.0.1:8080\n" {
		t.Fatalf("unexpected proxy content: %q", string(proxyData))
	}
}
