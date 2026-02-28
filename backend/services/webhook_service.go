package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"tellonym-checker/backend/core/config"
	"tellonym-checker/backend/utils/logger"
)

const (
	fixedWebhookUsername    = "Gon"
	fixedWebhookAvatar      = "https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg"
	webhookFooterText       = "github: @00ie | discord.gg/2asv4rEhGh"
	defaultWebhookTimeoutMs = int64(10000)
)

type WebhookConfig struct {
	Label     string `json:"label"`
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url"`
	TimeoutMs int64  `json:"timeoutMs"`
}

type WebhookSettings struct {
	Enabled       bool            `json:"enabled"`
	URL           string          `json:"url"`
	Username      string          `json:"username"`
	AvatarURL     string          `json:"avatarURL"`
	TimeoutMs     int64           `json:"timeoutMs"`
	ActiveWebhook int             `json:"activeWebhook"`
	Webhooks      []WebhookConfig `json:"webhooks"`
}

type webhookJob struct {
	username string
}

type WebhookService struct {
	mu       sync.RWMutex
	logger   *logger.Logger
	config   WebhookSettings
	queue    chan webhookJob
	quit     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
	sent     map[string]time.Time
}

func NewWebhookService(cfg *config.WebhookConfig, logger *logger.Logger) *WebhookService {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	initialURL, err := normalizeWebhookURL(cfg.URL)
	if err != nil {
		initialURL = strings.TrimSpace(cfg.URL)
	}

	initialSettings := WebhookSettings{
		Enabled:       cfg.Enabled,
		URL:           initialURL,
		Username:      fixedWebhookUsername,
		AvatarURL:     fixedWebhookAvatar,
		TimeoutMs:     timeout.Milliseconds(),
		ActiveWebhook: 0,
		Webhooks: []WebhookConfig{
			{
				Label:     webhookLabel(0),
				Enabled:   cfg.Enabled,
				URL:       initialURL,
				TimeoutMs: timeout.Milliseconds(),
			},
		},
	}

	service := &WebhookService{
		logger: logger.Named("webhook-service"),
		config: normalizeWebhookSettings(initialSettings),
		queue:  make(chan webhookJob, 512),
		quit:   make(chan struct{}),
		sent:   make(map[string]time.Time),
	}

	service.wg.Add(1)
	go service.runQueue()

	return service
}

func (s *WebhookService) Stop() {
	s.stopOnce.Do(func() {
		close(s.quit)
		s.wg.Wait()
	})
}

func (s *WebhookService) GetSettings() WebhookSettings {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	return normalizeWebhookSettings(cfg)
}

func (s *WebhookService) UpdateSettings(settings WebhookSettings) error {
	normalized := normalizeWebhookSettings(settings)

	for index, webhook := range normalized.Webhooks {
		normalizedURL, err := normalizeWebhookURL(webhook.URL)
		if err != nil {
			return fmt.Errorf("webhook %d URL is invalid: %w", index+1, err)
		}
		normalized.Webhooks[index].URL = normalizedURL
		if normalized.Webhooks[index].Enabled && normalizedURL == "" {
			return fmt.Errorf("webhook %d URL is required when enabled", index+1)
		}
	}

	if normalized.ActiveWebhook < 0 || normalized.ActiveWebhook >= len(normalized.Webhooks) {
		normalized.ActiveWebhook = 0
	}
	selected := normalized.Webhooks[normalized.ActiveWebhook]
	normalized.Enabled = selected.Enabled
	normalized.URL = selected.URL
	normalized.TimeoutMs = selected.TimeoutMs

	s.mu.Lock()
	s.config = normalized
	s.mu.Unlock()

	s.logger.Info(
		"webhook settings updated",
		"webhooks", len(normalized.Webhooks),
		"active_webhook", normalized.ActiveWebhook+1,
	)
	return nil
}

func (s *WebhookService) SendUsernameAvailable(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}

	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if !hasEnabledWebhook(cfg.Webhooks) {
		return nil
	}

	if s.isDuplicate(username) {
		return nil
	}

	select {
	case s.queue <- webhookJob{username: username}:
		return nil
	default:
		return fmt.Errorf("webhook queue is full")
	}
}

func (s *WebhookService) SendTest(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "available_name"
	}

	s.mu.RLock()
	cfg := normalizeWebhookSettings(s.config)
	s.mu.RUnlock()

	if len(cfg.Webhooks) == 0 {
		return fmt.Errorf("no webhook configured")
	}

	active := cfg.Webhooks[cfg.ActiveWebhook]
	if active.URL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	timeout := normalizeTimeoutMs(active.TimeoutMs)
	client := &http.Client{Timeout: time.Duration(timeout) * time.Millisecond}

	return s.sendWithRetry(client, WebhookSettings{
		URL:       active.URL,
		TimeoutMs: timeout,
	}, username, true)
}

func (s *WebhookService) runQueue() {
	defer s.wg.Done()

	var lastSent time.Time

	for {
		select {
		case <-s.quit:
			return
		case job := <-s.queue:
			if !lastSent.IsZero() {
				wait := time.Second - time.Since(lastSent)
				if wait > 0 {
					select {
					case <-time.After(wait):
					case <-s.quit:
						return
					}
				}
			}

			s.mu.RLock()
			cfg := normalizeWebhookSettings(s.config)
			s.mu.RUnlock()

			for _, webhook := range cfg.Webhooks {
				if !webhook.Enabled || webhook.URL == "" {
					continue
				}

				timeout := normalizeTimeoutMs(webhook.TimeoutMs)
				client := &http.Client{Timeout: time.Duration(timeout) * time.Millisecond}
				if err := s.sendWithRetry(client, WebhookSettings{
					URL:       webhook.URL,
					TimeoutMs: timeout,
				}, job.username, false); err != nil {
					s.logger.Warn("failed to send webhook", "webhook", webhook.Label, "error", err)
				}
			}

			lastSent = time.Now()
		}
	}
}

func (s *WebhookService) isDuplicate(username string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if sentAt, exists := s.sent[username]; exists && now.Sub(sentAt) < 24*time.Hour {
		return true
	}

	s.sent[username] = now

	if len(s.sent) > 3000 {
		for key, sentAt := range s.sent {
			if now.Sub(sentAt) > 24*time.Hour {
				delete(s.sent, key)
			}
		}
	}

	return false
}

func (s *WebhookService) sendWithRetry(client *http.Client, cfg WebhookSettings, username string, test bool) error {
	maxAttempts := 4
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		retryAfter, retryable, err := sendWebhookRequest(client, cfg, username, test)
		if err == nil {
			return nil
		}

		lastErr = err
		if !retryable || attempt == maxAttempts-1 {
			break
		}

		if retryAfter <= 0 {
			retryAfter = time.Duration(attempt+1) * time.Second
		}

		time.Sleep(retryAfter)
	}

	return lastErr
}

func sendWebhookRequest(client *http.Client, cfg WebhookSettings, username string, test bool) (time.Duration, bool, error) {
	payload := buildWebhookPayload(cfg, username, test)
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, false, err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(data))
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0, false, nil
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		if retryAfter <= 0 {
			retryAfter = 2 * time.Second
		}
		return retryAfter, true, fmt.Errorf("webhook returned status 429")
	}

	if resp.StatusCode >= 500 {
		return 0, true, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return 0, false, fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

func parseRetryAfter(raw string) time.Duration {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(value)
	if err == nil {
		if seconds < 0 {
			seconds = 0
		}
		return time.Duration(seconds) * time.Second
	}

	when, err := time.Parse(time.RFC1123, value)
	if err != nil {
		return 0
	}

	delta := time.Until(when)
	if delta < 0 {
		return 0
	}

	return delta
}

func buildWebhookPayload(cfg WebhookSettings, username string, test bool) map[string]any {
	profileLink := fmt.Sprintf("https://tellonym.me/%s", username)
	title := fmt.Sprintf("Available: @%s | by @00ie", username)
	if test {
		title = fmt.Sprintf("Test Alert: @%s | by @00ie", username)
	}

	foundAt := time.Now().Format("15:04:05")

	return map[string]any{
		"username":   fixedWebhookUsername,
		"avatar_url": fixedWebhookAvatar,
		"allowed_mentions": map[string]any{
			"parse": []string{},
		},
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": "Username candidate found by checker",
				"color":       15548997,
				"thumbnail": map[string]string{
					"url": fixedWebhookAvatar,
				},
				"fields": []map[string]any{
					{
						"name":   "Username",
						"value":  fmt.Sprintf("@%s", username),
						"inline": true,
					},
					{
						"name":   "Found at",
						"value":  foundAt,
						"inline": true,
					},
					{
						"name":   "Platform",
						"value":  "Tellonym",
						"inline": true,
					},
					{
						"name":   "Profile Link",
						"value":  fmt.Sprintf("[Check now](%s)", profileLink),
						"inline": true,
					},
				},
				"footer": map[string]string{
					"text": webhookFooterText,
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

func normalizeWebhookSettings(settings WebhookSettings) WebhookSettings {
	settings.Username = fixedWebhookUsername
	settings.AvatarURL = fixedWebhookAvatar
	settings.URL = strings.TrimSpace(settings.URL)
	settings.TimeoutMs = normalizeTimeoutMs(settings.TimeoutMs)

	if len(settings.Webhooks) == 0 {
		settings.Webhooks = []WebhookConfig{
			{
				Label:     webhookLabel(0),
				Enabled:   settings.Enabled,
				URL:       settings.URL,
				TimeoutMs: settings.TimeoutMs,
			},
		}
	} else {
		normalized := make([]WebhookConfig, 0, len(settings.Webhooks))
		for index, webhook := range settings.Webhooks {
			timeout := normalizeTimeoutMs(webhook.TimeoutMs)
			if index == settings.ActiveWebhook && settings.TimeoutMs > 0 {
				timeout = normalizeTimeoutMs(settings.TimeoutMs)
			}
			normalized = append(normalized, WebhookConfig{
				Label:     webhookLabel(index),
				Enabled:   webhook.Enabled,
				URL:       strings.TrimSpace(webhook.URL),
				TimeoutMs: timeout,
			})
		}
		settings.Webhooks = normalized
	}

	if settings.ActiveWebhook < 0 || settings.ActiveWebhook >= len(settings.Webhooks) {
		settings.ActiveWebhook = 0
	}

	active := settings.Webhooks[settings.ActiveWebhook]
	settings.Enabled = active.Enabled
	settings.URL = active.URL
	settings.TimeoutMs = normalizeTimeoutMs(active.TimeoutMs)

	return settings
}

func normalizeWebhookURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", fmt.Errorf("malformed URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("missing host")
	}

	return parsed.String(), nil
}

func normalizeTimeoutMs(timeoutMs int64) int64 {
	if timeoutMs <= 0 {
		return defaultWebhookTimeoutMs
	}
	return timeoutMs
}

func hasEnabledWebhook(webhooks []WebhookConfig) bool {
	for _, webhook := range webhooks {
		if webhook.Enabled && webhook.URL != "" {
			return true
		}
	}
	return false
}

func webhookLabel(index int) string {
	return fmt.Sprintf("Webhook %d", index+1)
}
