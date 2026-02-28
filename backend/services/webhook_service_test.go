package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tellonym-checker/backend/core/config"
	"tellonym-checker/backend/utils/logger"
)

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter("5"); got != 5*time.Second {
		t.Fatalf("expected 5s, got %s", got)
	}

	if got := parseRetryAfter("invalid"); got != 0 {
		t.Fatalf("expected 0 for invalid value, got %s", got)
	}
}

func TestBuildWebhookPayloadFixedIdentity(t *testing.T) {
	payload := buildWebhookPayload(WebhookSettings{}, "myuser", false)

	username, _ := payload["username"].(string)
	if username != fixedWebhookUsername {
		t.Fatalf("expected fixed username %q, got %q", fixedWebhookUsername, username)
	}

	avatar, _ := payload["avatar_url"].(string)
	if avatar != fixedWebhookAvatar {
		t.Fatalf("expected fixed avatar %q, got %q", fixedWebhookAvatar, avatar)
	}

	embeds, ok := payload["embeds"].([]map[string]any)
	if !ok || len(embeds) == 0 {
		t.Fatalf("expected embeds payload")
	}

	fields, ok := embeds[0]["fields"].([]map[string]any)
	if !ok || len(fields) == 0 {
		t.Fatalf("expected embed fields")
	}

	foundProfileLink := false
	for _, field := range fields {
		value, _ := field["value"].(string)
		if strings.Contains(value, "https://tellonym.me/myuser") {
			foundProfileLink = true
			break
		}
	}
	if !foundProfileLink {
		t.Fatalf("expected payload to include profile link for username")
	}
}

func TestSendWebhookRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST method, got %s", r.Method)
		}
		if contentType := r.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Fatalf("expected application/json content-type, got %q", contentType)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}

		if payload["username"] != fixedWebhookUsername {
			t.Fatalf("expected fixed username in payload")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := WebhookSettings{
		Enabled: true,
		URL:     server.URL,
	}

	retryAfter, retryable, err := sendWebhookRequest(server.Client(), cfg, "available_name", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retryAfter != 0 {
		t.Fatalf("expected retryAfter=0, got %s", retryAfter)
	}
	if retryable {
		t.Fatalf("expected retryable=false on success")
	}
}

func TestSendWebhookRequestRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := WebhookSettings{
		Enabled: true,
		URL:     server.URL,
	}

	retryAfter, retryable, err := sendWebhookRequest(server.Client(), cfg, "available_name", false)
	if err == nil {
		t.Fatalf("expected error for rate-limited response")
	}
	if !retryable {
		t.Fatalf("expected retryable=true for 429")
	}
	if retryAfter != 2*time.Second {
		t.Fatalf("expected retryAfter=2s, got %s", retryAfter)
	}
}

func TestWebhookServiceUpdateSettingsRoundTripMultipleWebhooks(t *testing.T) {
	service := NewWebhookService(&config.WebhookConfig{}, logger.NewLogger(logger.Config{Level: "error"}))
	defer service.Stop()

	err := service.UpdateSettings(WebhookSettings{
		ActiveWebhook: 1,
		Webhooks: []WebhookConfig{
			{
				Label:     "Webhook 1",
				Enabled:   true,
				URL:       "https://discord.com/api/webhooks/111/aaa",
				TimeoutMs: 8000,
			},
			{
				Label:     "Webhook 2",
				Enabled:   true,
				URL:       "canary.discord.com/api/webhooks/222/bbb",
				TimeoutMs: 9000,
			},
			{
				Label:     "Webhook 3",
				Enabled:   false,
				URL:       "https://ptb.discord.com/api/webhooks/333/ccc",
				TimeoutMs: 10000,
			},
		},
	})
	if err != nil {
		t.Fatalf("update settings failed: %v", err)
	}

	settings := service.GetSettings()
	if len(settings.Webhooks) != 3 {
		t.Fatalf("expected 3 webhooks, got %d", len(settings.Webhooks))
	}
	if settings.ActiveWebhook != 1 {
		t.Fatalf("expected active webhook 1, got %d", settings.ActiveWebhook)
	}
	if settings.Webhooks[1].Label != "Webhook 2" {
		t.Fatalf("expected relabeled webhook 2, got %q", settings.Webhooks[1].Label)
	}
	if settings.Webhooks[1].URL != "https://canary.discord.com/api/webhooks/222/bbb" {
		t.Fatalf("unexpected normalized canary URL: %q", settings.Webhooks[1].URL)
	}
	if settings.URL != settings.Webhooks[1].URL {
		t.Fatalf("expected top-level URL to match active webhook")
	}
}

func TestNormalizeWebhookURLAcceptsDiscordVariants(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal discord",
			input: "https://discord.com/api/webhooks/123/token",
			want:  "https://discord.com/api/webhooks/123/token",
		},
		{
			name:  "canary discord without scheme",
			input: "canary.discord.com/api/webhooks/123/token",
			want:  "https://canary.discord.com/api/webhooks/123/token",
		},
		{
			name:  "ptb discord",
			input: "https://ptb.discord.com/api/webhooks/123/token",
			want:  "https://ptb.discord.com/api/webhooks/123/token",
		},
		{
			name:  "custom https endpoint",
			input: "https://webhook.site/abc",
			want:  "https://webhook.site/abc",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeWebhookURL(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestNormalizeWebhookURLRejectsUnsupportedScheme(t *testing.T) {
	_, err := normalizeWebhookURL("ftp://discord.com/api/webhooks/123/token")
	if err == nil {
		t.Fatalf("expected error for unsupported scheme")
	}
}
