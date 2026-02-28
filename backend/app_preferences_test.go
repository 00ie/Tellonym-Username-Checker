package backend

import (
	"path/filepath"
	"testing"
)

func TestLoadAppSettingsDefaultsWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	settings, err := loadAppSettings(path)
	if err != nil {
		t.Fatalf("load app settings failed: %v", err)
	}
	if !settings.OpenLinksOnClose {
		t.Fatalf("expected open links on close default to true")
	}
}

func TestSaveAndLoadWindowStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window_state.json")
	input := WindowState{
		Width:  1280,
		Height: 800,
		X:      100,
		Y:      120,
		State:  "normal",
	}

	if err := saveWindowState(path, input); err != nil {
		t.Fatalf("save window state failed: %v", err)
	}

	output, err := loadWindowState(path)
	if err != nil {
		t.Fatalf("load window state failed: %v", err)
	}
	if output != input {
		t.Fatalf("expected %#v, got %#v", input, output)
	}
}
