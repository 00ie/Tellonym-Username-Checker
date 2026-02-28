package backend

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type runtimeLayout struct {
	RootDir    string
	ConfigDir  string
	DataDir    string
	LogsDir    string
	ExportsDir string
	ConfigFile string
	ProxyFile  string
	FoundFile  string
	LogFile    string
}

func prepareRuntimeLayout() (runtimeLayout, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}

	rootDir := filepath.Join(workingDir, "tellonym checker")
	layout := runtimeLayout{
		RootDir:    rootDir,
		ConfigDir:  filepath.Join(rootDir, "config"),
		DataDir:    filepath.Join(rootDir, "data"),
		LogsDir:    filepath.Join(rootDir, "logs"),
		ExportsDir: filepath.Join(rootDir, "exports"),
		ConfigFile: filepath.Join(rootDir, "config", "config.yaml"),
		ProxyFile:  filepath.Join(rootDir, "proxies.txt"),
		FoundFile:  filepath.Join(rootDir, "data", "found_usernames.txt"),
		LogFile:    filepath.Join(rootDir, "logs", "app.log"),
	}

	dirs := []string{
		layout.RootDir,
		layout.ConfigDir,
		layout.DataDir,
		layout.LogsDir,
		layout.ExportsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return runtimeLayout{}, err
		}
	}

	if err := ensureFile(layout.ProxyFile); err != nil {
		return runtimeLayout{}, err
	}
	if err := ensureFile(layout.FoundFile); err != nil {
		return runtimeLayout{}, err
	}

	if err := seedFileIfTargetEmpty(layout.ProxyFile, filepath.Join(workingDir, "proxies.txt")); err != nil {
		return runtimeLayout{}, err
	}
	if err := seedFileIfMissing(layout.ConfigFile, filepath.Join(workingDir, "configs", "config.yaml")); err != nil {
		return runtimeLayout{}, err
	}

	return layout, nil
}

func ensureFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func seedFileIfMissing(targetPath, sourcePath string) error {
	_, err := os.Stat(targetPath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return copyFile(sourcePath, targetPath)
}

func seedFileIfTargetEmpty(targetPath, sourcePath string) error {
	info, err := os.Stat(targetPath)
	if err == nil && info.Size() > 0 {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return copyFile(sourcePath, targetPath)
}

func copyFile(sourcePath, targetPath string) error {
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ensureFile(targetPath)
		}
		return err
	}
	if strings.TrimSpace(string(sourceData)) == "" {
		return ensureFile(targetPath)
	}
	return os.WriteFile(targetPath, sourceData, 0o644)
}

func resolveRuntimeFile(baseDir, configuredPath, fallbackName string) string {
	trimmed := strings.TrimSpace(configuredPath)
	if trimmed == "" {
		return filepath.Join(baseDir, fallbackName)
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(baseDir, filepath.Base(trimmed))
}
