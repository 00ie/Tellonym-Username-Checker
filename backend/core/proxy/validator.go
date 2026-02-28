package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tellonym-checker/backend/utils/logger"
)

type Validator struct {
	config *Config
	logger *logger.Logger
}

func NewValidator(cfg *Config, logger *logger.Logger) *Validator {
	return &Validator{config: cfg, logger: logger.Named("proxy-validator")}
}

func (v *Validator) Validate(p *Proxy) (bool, int64) {
	if p == nil || p.URL == nil {
		return false, 0
	}

	transport := &http.Transport{}
	if err := ApplyProxyToTransport(transport, p.URL, v.config.ValidationTimeout); err != nil {
		v.logger.Debug("proxy transport setup failed", "proxy", p.URL.String(), "error", err)
		return false, 0
	}

	client := &http.Client{
		Timeout:   v.config.ValidationTimeout,
		Transport: transport,
	}

	started := time.Now()
	req, err := http.NewRequest(http.MethodGet, v.config.ValidationURL, nil)
	if err != nil {
		return false, 0
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()

	latency := time.Since(started).Milliseconds()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 400

	if !ok {
		v.logger.Debug("proxy validation failed", "proxy", p.URL.String(), "status", resp.StatusCode)
	} else {
		v.logger.Debug("proxy validation ok", "proxy", p.URL.String(), "latency_ms", latency)
	}

	return ok, latency
}

func (v *Validator) ValidateRaw(raw string) (bool, int64, error) {
	p, err := ParseProxy(raw)
	if err != nil {
		return false, 0, err
	}
	ok, latency := v.Validate(&Proxy{URL: p, Type: p.Scheme})
	return ok, latency, nil
}

func ParseProxy(raw string) (*url.URL, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return nil, fmt.Errorf("empty proxy")
	}

	if !strings.Contains(candidate, "://") {
		candidate = "http://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil, err
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid proxy host")
	}

	return parsed, nil
}
