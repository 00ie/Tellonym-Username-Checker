package backend

import (
	"context"
	"fmt"
	"os"

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
	checkerService *services.CheckerService
	proxyService   *services.ProxyService
	statsService   *services.StatsService
	webhookService *services.WebhookService
	storage        *storage.Storage
}

func NewApp() *App {
	return &App{}
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

	runtime.LogInfo(ctx, "application shutdown complete")
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
