package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"tellonym-checker/backend/core/config"
	"tellonym-checker/backend/utils/logger"
)

const (
	fixedWebhookUsername = "Gon"
	fixedWebhookAvatar   = "https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg"
	webhookFooterText    = "github: @00ie | discord.gg/2asv4rEhGh"
)

type WebhookSettings struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarURL"`
	TimeoutMs int64  `json:"timeoutMs"`
}

type webhookJob struct {
	username string
}

type WebhookService struct {
	mu       sync.RWMutex
	logger   *logger.Logger
	config   WebhookSettings
	client   *http.Client
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

	service := &WebhookService{
		logger: logger.Named("webhook-service"),
		config: WebhookSettings{
			Enabled:   cfg.Enabled,
			URL:       strings.TrimSpace(cfg.URL),
			Username:  fixedWebhookUsername,
			AvatarURL: fixedWebhookAvatar,
			TimeoutMs: timeout.Milliseconds(),
		},
		client: &http.Client{Timeout: timeout},
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

	cfg.Username = fixedWebhookUsername
	cfg.AvatarURL = fixedWebhookAvatar
	return cfg
}

func (s *WebhookService) UpdateSettings(settings WebhookSettings) error {
	settings.URL = strings.TrimSpace(settings.URL)
	settings.Username = fixedWebhookUsername
	settings.AvatarURL = fixedWebhookAvatar

	if settings.TimeoutMs <= 0 {
		settings.TimeoutMs = 10000
	}
	if settings.Enabled && settings.URL == "" {
		return fmt.Errorf("webhook URL is required when enabled")
	}

	s.mu.Lock()
	s.config = settings
	s.client = &http.Client{Timeout: time.Duration(settings.TimeoutMs) * time.Millisecond}
	s.mu.Unlock()

	s.logger.Info("webhook settings updated", "enabled", settings.Enabled)
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

	if !cfg.Enabled || cfg.URL == "" {
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
	cfg := s.config
	client := s.client
	s.mu.RUnlock()

	if cfg.URL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	return s.sendWithRetry(client, cfg, username, true)
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
			cfg := s.config
			client := s.client
			s.mu.RUnlock()

			if !cfg.Enabled || cfg.URL == "" {
				continue
			}

			if err := s.sendWithRetry(client, cfg, job.username, false); err != nil {
				s.logger.Warn("failed to send webhook", "error", err)
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
