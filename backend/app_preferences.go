package backend

import (
	"encoding/json"
	"os"
)

type AppSettings struct {
	OpenLinksOnClose bool `json:"openLinksOnClose"`
}

type WindowState struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	State  string `json:"state"`
}

func defaultAppSettings() AppSettings {
	return AppSettings{
		OpenLinksOnClose: true,
	}
}

func loadAppSettings(path string) (AppSettings, error) {
	settings := defaultAppSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return settings, nil
		}
		return settings, err
	}
	if len(data) == 0 {
		return settings, nil
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultAppSettings(), err
	}
	return settings, nil
}

func saveAppSettings(path string, settings AppSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadWindowState(path string) (WindowState, error) {
	var state WindowState
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return WindowState{}, err
	}
	if state.State == "" {
		state.State = "normal"
	}
	return state, nil
}

func saveWindowState(path string, state WindowState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s WindowState) IsValid() bool {
	if s.State != "normal" && s.State != "maximised" && s.State != "minimised" && s.State != "fullscreen" {
		return false
	}
	if s.State != "normal" {
		return true
	}
	return s.Width >= 700 && s.Height >= 500
}
