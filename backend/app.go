package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"tellonym-checker/backend/core/checker"
	"tellonym-checker/backend/core/config"
	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/core/storage"
	"tellonym-checker/backend/services"
	"tellonym-checker/backend/utils/logger"
)

type App struct {
	ctx            context.Context
	config         *config.Config
	logger         *logger.Logger
	layout         runtimeLayout
	settings       AppSettings
	settingsMu     sync.RWMutex
	checkerService *services.CheckerService
	proxyService   *services.ProxyService
	statsService   *services.StatsService
	webhookService *services.WebhookService
	storage        *storage.Storage
}

const (
	serverInviteURL = "https://discord.gg/2asv4rEhGh"
	githubProfile   = "https://github.com/00ie"
)

type ConfigExportSnapshot struct {
	Version         string                   `json:"version"`
	ExportedAt      string                   `json:"exportedAt"`
	AppSettings     AppSettings              `json:"appSettings"`
	CheckerSettings services.CheckerSettings `json:"checkerSettings"`
	WebhookSettings services.WebhookSettings `json:"webhookSettings"`
}

type ConfigImportSnapshot struct {
	AppSettings     *AppSettings              `json:"appSettings"`
	CheckerSettings *services.CheckerSettings `json:"checkerSettings"`
	WebhookSettings *services.WebhookSettings `json:"webhookSettings"`
}

func NewApp() *App {
	return &App{
		settings: defaultAppSettings(),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	a.logger = logger.NewLogger(logger.Config{
		Level:    "info",
		Encoding: "console",
	})

	layout, err := prepareRuntimeLayout()
	if err != nil {
		a.logger.Error("failed to prepare runtime layout", "error", err)
		runtime.LogError(ctx, fmt.Sprintf("runtime layout error: %v", err))
		return
	}
	a.layout = layout
	a.settings = defaultAppSettings()

	settings, err := loadAppSettings(layout.AppSettingsFile)
	if err != nil {
		a.logger.Warn("failed to load app settings", "error", err)
	} else {
		a.settings = settings
	}

	if err := saveAppSettings(layout.AppSettingsFile, a.settings); err != nil {
		a.logger.Warn("failed to persist app settings", "error", err)
	}

	if os.Getenv("TELLONYM_CONFIG") == "" {
		if err := os.Setenv("TELLONYM_CONFIG", layout.ConfigFile); err != nil {
			a.logger.Warn("failed to set config path", "error", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		a.logger.Error("failed to load config", "error", err)
		runtime.LogError(ctx, fmt.Sprintf("config error: %v", err))
		return
	}

	cfg.Logging.OutputPath = resolveRuntimeFile(layout.LogsDir, cfg.Logging.OutputPath, "app.log")
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Encoding == "" {
		cfg.Logging.Encoding = "console"
	}

	a.logger.Sync()
	a.logger = logger.NewLogger(logger.Config{
		Level:      cfg.Logging.Level,
		OutputPath: cfg.Logging.OutputPath,
		Encoding:   cfg.Logging.Encoding,
	})

	if cfg.Storage.Type == "" || cfg.Storage.Type == "file" {
		cfg.Storage.Type = "file"
		cfg.Storage.FilePath = resolveRuntimeFile(layout.DataDir, cfg.Storage.FilePath, "found_usernames.txt")
	}

	a.config = cfg

	store, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		a.logger.Error("failed to initialize storage", "error", err)
		runtime.LogError(ctx, fmt.Sprintf("storage error: %v", err))
		return
	}
	a.storage = store

	proxyManager := proxy.NewManager(&cfg.Proxy, a.logger)
	if err := proxyManager.LoadFromFile(layout.ProxyFile); err != nil {
		a.logger.Warn("no proxies loaded", "error", err)
	}

	tellonymChecker := checker.NewChecker(&cfg.Checker, a.logger, proxyManager)
	a.webhookService = services.NewWebhookService(&cfg.Webhook, a.logger)
	tellonymChecker.SetOnFound(func(username string) {
		if a.storage != nil {
			_ = a.storage.AddFound(username)
		}
		if a.webhookService != nil {
			if err := a.webhookService.SendUsernameAvailable(username); err != nil {
				a.logger.Warn("failed to enqueue webhook", "error", err)
			}
		}
	})

	a.checkerService = services.NewCheckerService(tellonymChecker, a.logger)
	a.proxyService = services.NewProxyService(proxyManager, a.logger)
	a.statsService = services.NewStatsService(tellonymChecker, proxyManager, a.logger, layout.ExportsDir)

	go a.statsService.StartStreaming(ctx)

	a.logger.Info("application started", "runtime_root", layout.RootDir)
	runtime.LogInfo(ctx, "Tellonym Username Checker started")
}

func (a *App) Shutdown(ctx context.Context) {
	if a.logger != nil {
		a.logger.Info("shutting down application")
	}

	if a.checkerService != nil {
		a.checkerService.Stop()
	}

	if a.statsService != nil {
		a.statsService.Stop()
	}

	if a.proxyService != nil {
		a.proxyService.Stop()
	}

	if a.webhookService != nil {
		a.webhookService.Stop()
	}

	if a.storage != nil {
		a.storage.Close()
	}

	if a.logger != nil {
		a.logger.Sync()
	}

	if a.shouldOpenLinksOnClose() {
		runtime.BrowserOpenURL(ctx, serverInviteURL)
		runtime.BrowserOpenURL(ctx, githubProfile)
	}

	runtime.LogInfo(ctx, "application shutdown complete")
}

func (a *App) DomReady(ctx context.Context) {
	if a.layout.WindowStateFile == "" {
		return
	}

	windowState, err := loadWindowState(a.layout.WindowStateFile)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("failed to load window state", "error", err)
		}
		return
	}
	if !windowState.IsValid() {
		return
	}

	if windowState.Width >= 700 && windowState.Height >= 500 {
		runtime.WindowSetSize(ctx, windowState.Width, windowState.Height)
		runtime.WindowSetPosition(ctx, windowState.X, windowState.Y)
	}

	switch windowState.State {
	case "maximised":
		runtime.WindowMaximise(ctx)
	case "minimised":
		runtime.WindowMinimise(ctx)
	case "fullscreen":
		runtime.WindowFullscreen(ctx)
	}
}

func (a *App) BeforeClose(ctx context.Context) bool {
	a.saveCurrentWindowState(ctx)
	return false
}

func (a *App) saveCurrentWindowState(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil && a.logger != nil {
			a.logger.Warn("recovered while saving window state", "error", recovered)
		}
	}()

	if a.layout.WindowStateFile == "" {
		return
	}

	existing, _ := loadWindowState(a.layout.WindowStateFile)
	state := existing
	state.State = "normal"

	if runtime.WindowIsFullscreen(ctx) {
		state.State = "fullscreen"
	} else if runtime.WindowIsMaximised(ctx) {
		state.State = "maximised"
	} else if runtime.WindowIsMinimised(ctx) {
		state.State = "minimised"
	}

	if runtime.WindowIsNormal(ctx) {
		width, height := runtime.WindowGetSize(ctx)
		x, y := runtime.WindowGetPosition(ctx)
		state.Width = width
		state.Height = height
		state.X = x
		state.Y = y
	}

	if !state.IsValid() {
		return
	}

	if err := saveWindowState(a.layout.WindowStateFile, state); err != nil && a.logger != nil {
		a.logger.Warn("failed to save window state", "error", err)
	}
}

func (a *App) shouldOpenLinksOnClose() bool {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings.OpenLinksOnClose
}

func (a *App) StartChecker(usernameLength int, threads int) error {
	if a.checkerService == nil {
		return fmt.Errorf("checker service not initialized")
	}
	return a.checkerService.Start(usernameLength, threads)
}

func (a *App) StopChecker() {
	if a.checkerService != nil {
		a.checkerService.Stop()
	}
}

func (a *App) PauseChecker() {
	if a.checkerService != nil {
		a.checkerService.Pause()
	}
}

func (a *App) ResumeChecker() {
	if a.checkerService != nil {
		a.checkerService.Resume()
	}
}

func (a *App) GetCheckerStatus() services.CheckerStatus {
	if a.checkerService == nil {
		return services.CheckerStatus{}
	}
	return a.checkerService.GetStatus()
}

func (a *App) LoadProxies(proxies []string) error {
	if a.proxyService == nil {
		return fmt.Errorf("proxy service not initialized")
	}
	return a.proxyService.AddProxies(proxies)
}

func (a *App) GetProxies() []services.ProxyInfo {
	if a.proxyService == nil {
		return []services.ProxyInfo{}
	}
	return a.proxyService.GetAllProxies()
}

func (a *App) TestProxy(proxyURL string) services.ProxyTestResult {
	if a.proxyService == nil {
		return services.ProxyTestResult{Success: false, Error: "proxy service not initialized"}
	}
	return a.proxyService.TestProxy(proxyURL)
}

func (a *App) CheckAllProxies() services.ProxyBatchCheckResult {
	if a.proxyService == nil {
		return services.ProxyBatchCheckResult{}
	}
	return a.proxyService.CheckAllProxies()
}

func (a *App) RemoveProxy(proxyURL string) {
	if a.proxyService != nil {
		a.proxyService.RemoveProxy(proxyURL)
	}
}

func (a *App) RemoveBadProxies() services.ProxyCleanupResult {
	if a.proxyService == nil {
		return services.ProxyCleanupResult{}
	}
	return a.proxyService.RemoveBadProxies()
}

func (a *App) GetProxyStats() services.ProxyStats {
	if a.proxyService == nil {
		return services.ProxyStats{}
	}
	return a.proxyService.GetStats()
}

func (a *App) GetLiveStats() services.LiveStats {
	if a.statsService == nil {
		return services.LiveStats{}
	}
	return a.statsService.GetLiveStats()
}

func (a *App) GetHistoricalStats(from, to string) ([]services.HistoricalStats, error) {
	if a.statsService == nil {
		return []services.HistoricalStats{}, nil
	}
	return a.statsService.GetHistoricalStats(from, to)
}

func (a *App) ExportStats(format string) (string, error) {
	if a.statsService == nil {
		return "", fmt.Errorf("stats service not initialized")
	}
	return a.statsService.Export(format)
}

func (a *App) GetConfig() config.AppConfig {
	if a.config == nil {
		return config.AppConfig{}
	}
	return a.config.App
}

func (a *App) GetAppSettings() AppSettings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

func (a *App) UpdateAppSettings(settings AppSettings) error {
	a.settingsMu.Lock()
	a.settings = settings
	a.settingsMu.Unlock()

	if a.layout.AppSettingsFile == "" {
		return nil
	}

	if err := saveAppSettings(a.layout.AppSettingsFile, settings); err != nil {
		return err
	}
	return nil
}

func (a *App) UpdateCheckerSettings(settings services.CheckerSettings) error {
	if a.checkerService == nil {
		return fmt.Errorf("checker service not initialized")
	}
	return a.checkerService.UpdateSettings(settings)
}

func (a *App) GetCheckerSettings() services.CheckerSettings {
	if a.checkerService == nil {
		return services.CheckerSettings{}
	}
	return a.checkerService.GetSettings()
}

func (a *App) GetWebhookSettings() services.WebhookSettings {
	if a.webhookService == nil {
		return services.WebhookSettings{}
	}
	return a.webhookService.GetSettings()
}

func (a *App) UpdateWebhookSettings(settings services.WebhookSettings) error {
	if a.webhookService == nil {
		return fmt.Errorf("webhook service not initialized")
	}
	return a.webhookService.UpdateSettings(settings)
}

func (a *App) SendTestWebhook(username string) error {
	if a.webhookService == nil {
		return fmt.Errorf("webhook service not initialized")
	}
	return a.webhookService.SendTest(username)
}

func (a *App) GetFoundUsernames() []string {
	if a.storage == nil {
		return []string{}
	}
	return a.storage.GetFoundUsernames()
}

func (a *App) ClearFoundUsernames() error {
	if a.storage == nil {
		return nil
	}
	return a.storage.ClearFound()
}

func (a *App) ClearDashboardData() error {
	if a.checkerService == nil {
		return fmt.Errorf("checker service not initialized")
	}

	if err := a.checkerService.ResetStats(); err != nil {
		return err
	}

	if a.statsService != nil {
		a.statsService.ClearDashboardData()
	}

	if a.storage != nil {
		if err := a.storage.ClearFound(); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) Ping() bool {
	return a.ctx != nil && a.checkerService != nil && a.statsService != nil
}

func (a *App) ExportFoundUsernames() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context not initialized")
	}
	if a.storage == nil {
		return "", fmt.Errorf("storage not initialized")
	}

	usernames := a.storage.GetFoundUsernames()
	if len(usernames) == 0 {
		return "", fmt.Errorf("no usernames to export")
	}
	sort.Strings(usernames)

	defaultName := fmt.Sprintf("found-usernames-%s.txt", time.Now().Format("20060102-150405"))
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export Found Usernames",
		DefaultFilename:  defaultName,
		DefaultDirectory: a.layout.ExportsDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(savePath) == "" {
		return "", fmt.Errorf("export cancelled")
	}

	var content string
	if strings.EqualFold(filepath.Ext(savePath), ".csv") {
		lines := make([]string, 0, len(usernames)+1)
		lines = append(lines, "username")
		lines = append(lines, usernames...)
		content = strings.Join(lines, "\n") + "\n"
	} else {
		content = strings.Join(usernames, "\n") + "\n"
	}

	if err := os.WriteFile(savePath, []byte(content), 0o644); err != nil {
		return "", err
	}

	return savePath, nil
}

func (a *App) ExportAppConfiguration() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context not initialized")
	}

	snapshot := ConfigExportSnapshot{
		Version:     "1",
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		AppSettings: a.GetAppSettings(),
	}
	if a.checkerService != nil {
		snapshot.CheckerSettings = a.checkerService.GetSettings()
	}
	if a.webhookService != nil {
		snapshot.WebhookSettings = a.webhookService.GetSettings()
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}

	defaultName := fmt.Sprintf("tellonym-config-%s.json", time.Now().Format("20060102-150405"))
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Export Configuration",
		DefaultFilename:  defaultName,
		DefaultDirectory: a.layout.ExportsDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(savePath) == "" {
		return "", fmt.Errorf("export cancelled")
	}

	if err := os.WriteFile(savePath, data, 0o644); err != nil {
		return "", err
	}

	return savePath, nil
}

func (a *App) ImportAppConfiguration() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("application context not initialized")
	}

	openPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Import Configuration",
		DefaultDirectory: a.layout.ExportsDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(openPath) == "" {
		return "", fmt.Errorf("import cancelled")
	}

	data, err := os.ReadFile(openPath)
	if err != nil {
		return "", err
	}

	var snapshot ConfigImportSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return "", fmt.Errorf("invalid config file: %w", err)
	}

	if snapshot.CheckerSettings != nil && a.checkerService != nil {
		if err := a.checkerService.UpdateSettings(*snapshot.CheckerSettings); err != nil {
			return "", err
		}
	}
	if snapshot.WebhookSettings != nil && a.webhookService != nil {
		if err := a.webhookService.UpdateSettings(*snapshot.WebhookSettings); err != nil {
			return "", err
		}
	}
	if snapshot.AppSettings != nil {
		if err := a.UpdateAppSettings(*snapshot.AppSettings); err != nil {
			return "", err
		}
	}
	if snapshot.AppSettings == nil && snapshot.CheckerSettings == nil && snapshot.WebhookSettings == nil {
		return "", fmt.Errorf("config file does not contain importable settings")
	}

	return openPath, nil
}
