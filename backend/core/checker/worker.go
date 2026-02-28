package checker

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/utils/logger"
	"tellonym-checker/backend/utils/ratelimit"
)

type Worker struct {
	ID           int
	config       *Config
	logger       *logger.Logger
	client       *http.Client
	proxyManager *proxy.Manager
	limiter      *ratelimit.Limiter
	transport    *http.Transport
	userAgents   []string
}

func NewWorker(id int, cfg *Config, logger *logger.Logger, pm *proxy.Manager, limiter *ratelimit.Limiter) *Worker {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
		DisableCompression: false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
	}

	if !cfg.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Worker{
		ID:           id,
		config:       cfg,
		logger:       logger.Named(fmt.Sprintf("worker-%d", id)),
		client:       client,
		proxyManager: pm,
		limiter:      limiter,
		transport:    transport,
		userAgents:   cfg.UserAgents,
	}
}

func (w *Worker) Check(task Task) Result {
	if w.limiter != nil {
		w.limiter.Wait()
	}

	var proxyURL *url.URL
	var proxyStr string

	if w.proxyManager != nil {
		if p := w.proxyManager.GetNextHealthy(); p != nil {
			proxyURL = p.URL
			proxyStr = p.URL.String()
			w.updateTransportProxy(proxyURL)
		} else {
			w.updateTransportProxy(nil)
		}
	}

	retries := task.Retries
	if retries <= 0 {
		retries = w.config.MaxRetries
	}

	var resp *http.Response
	var lastErr error
	lastStatusCode := 0

	target := fmt.Sprintf("https://tellonym.me/%s", task.Username)

	for attempt := 0; attempt <= retries; attempt++ {
		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			return Result{
				Username:  task.Username,
				Error:     fmt.Errorf("failed to create request: %w", err),
				WorkerID:  w.ID,
				Timestamp: time.Now(),
			}
		}

		w.setHeaders(req)

		if attempt > 0 {
			backoff := w.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		resp, lastErr = w.client.Do(req)
		if lastErr != nil {
			w.logger.Debug(
				"request failed",
				"attempt",
				attempt+1,
				"username",
				task.Username,
				"error",
				lastErr,
			)
			continue
		}

		lastStatusCode = resp.StatusCode

		if isTransientStatusCode(resp.StatusCode) {
			lastErr = fmt.Errorf("transient status code %d", resp.StatusCode)
			resp.Body.Close()
			w.logger.Debug(
				"transient response status",
				"attempt",
				attempt+1,
				"username",
				task.Username,
				"status_code",
				resp.StatusCode,
			)
			continue
		}

		lastErr = nil
		break
	}

	if lastErr != nil {
		if proxyStr != "" && w.proxyManager != nil {
			w.proxyManager.ReportFailure(proxyStr)
		}
		return Result{
			Username:   task.Username,
			Error:      lastErr,
			StatusCode: lastStatusCode,
			WorkerID:   w.ID,
			ProxyUsed:  proxyStr,
			Timestamp:  time.Now(),
		}
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	found, confident := w.isUsernameAvailable(resp.StatusCode, string(body))
	if found {
		if apiResult, decided := w.checkAvailabilityByAPI(task.Username); decided {
			found = apiResult
			confident = true
		}
	}

	if !confident {
		return Result{
			Username:   task.Username,
			Error:      fmt.Errorf("indeterminate response status %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
			ProxyUsed:  proxyStr,
			WorkerID:   w.ID,
			Timestamp:  time.Now(),
		}
	}

	return Result{
		Username:   task.Username,
		Found:      found,
		StatusCode: resp.StatusCode,
		ProxyUsed:  proxyStr,
		WorkerID:   w.ID,
		Timestamp:  time.Now(),
	}
}

func (w *Worker) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", w.getUserAgent())
	req.Header.Set("Cookie", fmt.Sprintf("__cf_bm=%s", uuid.New().String()))
}

func (w *Worker) getUserAgent() string {
	if len(w.userAgents) == 0 {
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	}
	idx := time.Now().UnixNano() % int64(len(w.userAgents))
	return w.userAgents[idx]
}

func (w *Worker) updateTransportProxy(proxyURL *url.URL) {
	if err := proxy.ApplyProxyToTransport(w.transport, proxyURL, w.config.RequestTimeout); err != nil {
		w.logger.Debug("failed to apply proxy to transport", "error", err)
		w.transport.Proxy = func(*http.Request) (*url.URL, error) {
			return nil, err
		}
	}
}

func (w *Worker) isUsernameAvailable(statusCode int, body string) (bool, bool) {
	if statusCode == http.StatusNotFound {
		return true, true
	}

	if statusCode != http.StatusOK {
		return false, false
	}

	normalized := strings.ToLower(body)

	availableSignals := []string{
		"this user doesn't exist",
		"page not found",
		"user not found",
		"profile not found",
		"we can't find this page",
	}
	for _, signal := range availableSignals {
		if strings.Contains(normalized, signal) {
			return true, true
		}
	}

	takenSignals := []string{
		"ask me anything",
		"tellonym",
		"username",
	}
	for _, signal := range takenSignals {
		if strings.Contains(normalized, signal) {
			return false, true
		}
	}

	return false, false
}

func (w *Worker) checkAvailabilityByAPI(username string) (bool, bool) {
	endpoint := fmt.Sprintf("https://api.tellonym.me/accounts/check?username=%s&limit=13", url.QueryEscape(username))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return false, false
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", w.getUserAgent())

	resp, err := w.client.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, false
	}

	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, false
	}

	value, exists := payload["username"]
	if !exists {
		return false, false
	}

	available, ok := value.(bool)
	if !ok {
		return false, false
	}

	return available, true
}

func isTransientStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusRequestTimeout, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return statusCode >= 500
	}
}
