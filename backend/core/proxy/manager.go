package proxy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tellonym-checker/backend/utils/logger"
)

type Manager struct {
	config     *Config
	logger     *logger.Logger
	proxies    []*Proxy
	healthy    []*Proxy
	sourceFile string
	mu         sync.RWMutex
	nextIndex  uint64
	validator  *Validator
	quit       chan struct{}
	wg         sync.WaitGroup
	started    bool
}

func NewManager(cfg *Config, logger *logger.Logger) *Manager {
	if cfg.HealthCheckInterval <= 0 {
		cfg.HealthCheckInterval = 5 * time.Minute
	}
	if cfg.ValidationTimeout <= 0 {
		cfg.ValidationTimeout = 10 * time.Second
	}
	if cfg.ValidationURL == "" {
		cfg.ValidationURL = "https://httpbin.org/ip"
	}
	if cfg.MaxConsecutiveFails <= 0 {
		cfg.MaxConsecutiveFails = 3
	}
	if len(cfg.Types) == 0 {
		cfg.Types = []string{"*"}
	}

	return &Manager{
		config:    cfg,
		logger:    logger.Named("proxy-manager"),
		proxies:   make([]*Proxy, 0),
		healthy:   make([]*Proxy, 0),
		validator: NewValidator(cfg, logger),
		quit:      make(chan struct{}),
	}
}

func (m *Manager) LoadFromFile(filename string) error {
	m.mu.Lock()
	m.sourceFile = filename
	m.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open proxy file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	input := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		input = append(input, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	added := m.AddProxies(input)
	if added == 0 {
		return fmt.Errorf("no valid proxies found")
	}

	m.logger.Info("loaded proxies from file", "count", added, "file", filename)
	return nil
}

func (m *Manager) AddProxies(proxies []string) int {
	normalized := NormalizeProxyLines(proxies)
	if len(normalized) == 0 {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]struct{}, len(m.proxies))
	for _, p := range m.proxies {
		existing[p.URL.String()] = struct{}{}
	}

	added := 0
	for _, raw := range normalized {
		p := m.parseProxy(raw)
		if p == nil {
			continue
		}
		if _, ok := existing[p.URL.String()]; ok {
			continue
		}
		m.proxies = append(m.proxies, p)
		existing[p.URL.String()] = struct{}{}
		added++
	}

	m.rebuildHealthyListLocked()
	m.persistProxiesLocked()

	if !m.started && len(m.proxies) > 0 {
		m.started = true
		m.wg.Add(1)
		go m.healthCheckLoop()
	}

	m.logger.Info("added proxies", "count", added)
	return added
}

func (m *Manager) parseProxy(raw string) *Proxy {
	parsed, err := ParseProxy(raw)
	if err != nil {
		return nil
	}

	proxyType := parsed.Scheme
	if proxyType == "" {
		proxyType = "http"
	}

	if !m.isTypeAllowed(proxyType) {
		return nil
	}

	return &Proxy{
		URL:      parsed,
		Type:     proxyType,
		LastUsed: time.Now(),
		IsAlive:  true,
	}
}

func (m *Manager) isTypeAllowed(proxyType string) bool {
	if len(m.config.Types) == 0 {
		return true
	}

	normalizedType := strings.ToLower(strings.TrimSpace(proxyType))
	for _, allowed := range m.config.Types {
		value := strings.ToLower(strings.TrimSpace(allowed))
		if value == "" {
			continue
		}
		if value == "*" || value == "all" {
			return true
		}
		if value == normalizedType {
			return true
		}
	}

	return false
}

func (m *Manager) healthCheckLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	m.checkAllProxies()

	for {
		select {
		case <-m.quit:
			return
		case <-ticker.C:
			m.checkAllProxies()
		}
	}
}

type checkResult struct {
	Alive    bool
	Latency  int64
	Checked  time.Time
	ProxyRef *Proxy
}

func (m *Manager) checkAllProxies() (int, int) {
	m.mu.RLock()
	proxies := make([]*Proxy, len(m.proxies))
	copy(proxies, m.proxies)
	m.mu.RUnlock()

	if len(proxies) == 0 {
		m.mu.Lock()
		m.healthy = m.healthy[:0]
		m.mu.Unlock()
		return 0, 0
	}

	results := make(chan checkResult, len(proxies))
	var checks sync.WaitGroup

	for _, item := range proxies {
		checks.Add(1)
		go func(p *Proxy) {
			defer checks.Done()
			ok, latency := m.validator.Validate(p)
			results <- checkResult{
				Alive:    ok,
				Latency:  latency,
				Checked:  time.Now(),
				ProxyRef: p,
			}
		}(item)
	}

	checks.Wait()
	close(results)

	healthyCount := 0

	m.mu.Lock()
	healthy := make([]*Proxy, 0, len(m.proxies))

	for result := range results {
		proxyItem := result.ProxyRef
		if proxyItem == nil {
			continue
		}

		proxyItem.LastCheck = result.Checked
		proxyItem.AvgResponseMs = result.Latency
		proxyItem.IsAlive = result.Alive

		if result.Alive {
			atomic.StoreInt32(&proxyItem.FailCount, 0)
			atomic.AddInt32(&proxyItem.SuccessCount, 1)
			healthy = append(healthy, proxyItem)
			healthyCount++
		} else {
			atomic.AddInt32(&proxyItem.FailCount, 1)
		}
	}

	m.healthy = healthy
	total := len(m.proxies)
	m.mu.Unlock()

	m.logger.Debug("proxy health check complete", "total", total, "healthy", healthyCount)
	return total, healthyCount
}

func (m *Manager) CheckAllProxies() (int, int, int) {
	total, healthy := m.checkAllProxies()
	return total, healthy, total - healthy
}

func (m *Manager) GetNextHealthy() *Proxy {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.healthy) == 0 {
		return nil
	}

	var selected *Proxy

	switch m.config.RotationStrategy {
	case "least-loaded":
		selected = m.healthy[0]
		for _, p := range m.healthy[1:] {
			if p.LastUsed.Before(selected.LastUsed) {
				selected = p
			}
		}
	case "round-robin":
		idx := atomic.AddUint64(&m.nextIndex, 1) % uint64(len(m.healthy))
		selected = m.healthy[idx]
	default:
		idx := time.Now().UnixNano() % int64(len(m.healthy))
		selected = m.healthy[idx]
	}

	selected.LastUsed = time.Now()
	return selected
}

func (m *Manager) ReportFailure(proxyURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.proxies {
		if p.URL.String() != proxyURL {
			continue
		}

		fails := atomic.AddInt32(&p.FailCount, 1)
		if fails >= int32(m.config.MaxConsecutiveFails) {
			p.IsAlive = false
			m.rebuildHealthyListLocked()
			m.logger.Debug("proxy marked as dead", "proxy", proxyURL, "fail_count", fails)
		}
		return
	}
}

func (m *Manager) TestProxy(proxyRaw string) (bool, int64, error) {
	return m.validator.ValidateRaw(proxyRaw)
}

func (m *Manager) RemoveProxy(proxyURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := m.proxies[:0]
	for _, p := range m.proxies {
		if p.URL.String() == proxyURL {
			continue
		}
		filtered = append(filtered, p)
	}

	m.proxies = filtered
	m.rebuildHealthyListLocked()
	m.persistProxiesLocked()
	m.logger.Info("proxy removed", "proxy", proxyURL)
}

func (m *Manager) RemoveDeadProxies() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxies) == 0 {
		return 0
	}

	threshold := int32(m.config.MaxConsecutiveFails)
	if threshold <= 0 {
		threshold = 1
	}

	kept := make([]*Proxy, 0, len(m.proxies))
	removed := 0

	for _, item := range m.proxies {
		if item == nil {
			removed++
			continue
		}

		if !item.IsAlive || atomic.LoadInt32(&item.FailCount) >= threshold {
			removed++
			continue
		}

		kept = append(kept, item)
	}

	m.proxies = kept
	m.rebuildHealthyListLocked()
	m.persistProxiesLocked()

	if removed > 0 {
		m.logger.Info("removed dead proxies", "removed", removed, "remaining", len(m.proxies))
	}

	return removed
}

func (m *Manager) GetAllProxies() []*Proxy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Proxy, 0, len(m.proxies))
	for _, proxyItem := range m.proxies {
		if proxyItem == nil {
			continue
		}
		copyItem := *proxyItem
		if proxyItem.URL != nil {
			urlCopy := *proxyItem.URL
			copyItem.URL = &urlCopy
		}
		out = append(out, &copyItem)
	}
	return out
}

func (m *Manager) GetStats() (total, healthy, dead int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.proxies)
	healthy = len(m.healthy)
	dead = total - healthy
	return
}

func (m *Manager) rebuildHealthyListLocked() {
	healthy := make([]*Proxy, 0, len(m.proxies))
	for _, p := range m.proxies {
		if p.IsAlive {
			healthy = append(healthy, p)
		}
	}
	m.healthy = healthy
}

func (m *Manager) persistProxiesLocked() {
	target := strings.TrimSpace(m.sourceFile)
	if target == "" {
		return
	}

	lines := make([]string, 0, len(m.proxies))
	for _, item := range m.proxies {
		if item == nil || item.URL == nil {
			continue
		}
		lines = append(lines, item.URL.String())
	}

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		m.logger.Warn("failed to persist proxies file", "file", target, "error", err)
	}
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if m.started {
		close(m.quit)
		m.started = false
	}
	m.mu.Unlock()

	m.wg.Wait()
}
