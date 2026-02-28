package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
