package services

import (
	"fmt"
	"sync/atomic"
	"time"

	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/utils/logger"
)

type ProxyInfo struct {
	URL           string `json:"url"`
	Type          string `json:"type"`
	IsAlive       bool   `json:"isAlive"`
	FailCount     int32  `json:"failCount"`
	SuccessCount  int32  `json:"successCount"`
	AvgResponseMs int64  `json:"avgResponseMs"`
	LastCheck     string `json:"lastCheck"`
}

type ProxyStats struct {
	Total   int `json:"total"`
	Healthy int `json:"healthy"`
	Dead    int `json:"dead"`
}

type ProxyTestResult struct {
	Success bool   `json:"success"`
	Latency int64  `json:"latency"`
	Error   string `json:"error"`
}

type ProxyBatchCheckResult struct {
	Checked    int   `json:"checked"`
	Healthy    int   `json:"healthy"`
	Dead       int   `json:"dead"`
	DurationMs int64 `json:"durationMs"`
}

type ProxyCleanupResult struct {
	Removed   int `json:"removed"`
	Remaining int `json:"remaining"`
}

type ProxyService struct {
	manager *proxy.Manager
	logger  *logger.Logger
}

func NewProxyService(manager *proxy.Manager, logger *logger.Logger) *ProxyService {
	return &ProxyService{manager: manager, logger: logger.Named("proxy-service")}
}

func (s *ProxyService) AddProxies(proxies []string) error {
	if s.manager == nil {
		return fmt.Errorf("proxy manager is not initialized")
	}

	added := s.manager.AddProxies(proxies)
	if added == 0 {
		return fmt.Errorf("no valid proxies added")
	}

	return nil
}

func (s *ProxyService) GetAllProxies() []ProxyInfo {
	if s.manager == nil {
		return []ProxyInfo{}
	}

	items := s.manager.GetAllProxies()
	out := make([]ProxyInfo, 0, len(items))

	for _, p := range items {
		lastCheck := ""
		if !p.LastCheck.IsZero() {
			lastCheck = p.LastCheck.Format(time.RFC3339)
		}

		out = append(out, ProxyInfo{
			URL:           p.URL.String(),
			Type:          p.Type,
			IsAlive:       p.IsAlive,
			FailCount:     atomic.LoadInt32(&p.FailCount),
			SuccessCount:  atomic.LoadInt32(&p.SuccessCount),
			AvgResponseMs: p.AvgResponseMs,
			LastCheck:     lastCheck,
		})
	}

	return out
}

func (s *ProxyService) TestProxy(proxyURL string) ProxyTestResult {
	if s.manager == nil {
		return ProxyTestResult{Success: false, Error: "proxy manager is not initialized"}
	}

	ok, latency, err := s.manager.TestProxy(proxyURL)
	if err != nil {
		return ProxyTestResult{Success: false, Error: err.Error()}
	}

	return ProxyTestResult{Success: ok, Latency: latency}
}

func (s *ProxyService) CheckAllProxies() ProxyBatchCheckResult {
	if s.manager == nil {
		return ProxyBatchCheckResult{}
	}

	started := time.Now()
	total, healthy, dead := s.manager.CheckAllProxies()

	return ProxyBatchCheckResult{
		Checked:    total,
		Healthy:    healthy,
		Dead:       dead,
		DurationMs: time.Since(started).Milliseconds(),
	}
}

func (s *ProxyService) RemoveProxy(proxyURL string) {
	if s.manager != nil {
		s.manager.RemoveProxy(proxyURL)
	}
}

func (s *ProxyService) RemoveBadProxies() ProxyCleanupResult {
	if s.manager == nil {
		return ProxyCleanupResult{}
	}

	removed := s.manager.RemoveDeadProxies()
	total, _, _ := s.manager.GetStats()

	return ProxyCleanupResult{
		Removed:   removed,
		Remaining: total,
	}
}

func (s *ProxyService) GetStats() ProxyStats {
	if s.manager == nil {
		return ProxyStats{}
	}

	total, healthy, dead := s.manager.GetStats()
	return ProxyStats{Total: total, Healthy: healthy, Dead: dead}
}

func (s *ProxyService) Stop() {
	if s.manager != nil {
		s.manager.Stop()
	}
}
