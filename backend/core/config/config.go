package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"tellonym-checker/backend/core/checker"
	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/core/storage"
)

type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
}

type APIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Auth    struct {
		Enabled bool     `yaml:"enabled"`
		APIKeys []string `yaml:"api_keys"`
	} `yaml:"auth"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	OutputPath string `yaml:"output_path"`
	Encoding   string `yaml:"encoding"`
}

type RateLimitConfig struct {
	Enabled        bool `yaml:"enabled"`
	RequestsPerSec int  `yaml:"requests_per_sec"`
	Burst          int  `yaml:"burst"`
	PerIP          bool `yaml:"per_ip"`
}

type WebhookConfig struct {
	Enabled   bool          `yaml:"enabled"`
	URL       string        `yaml:"url"`
	Username  string        `yaml:"username"`
	AvatarURL string        `yaml:"avatar_url"`
	Timeout   time.Duration `yaml:"timeout"`
}

type Config struct {
	App       AppConfig       `yaml:"app"`
	Checker   checker.Config  `yaml:"checker"`
	Proxy     proxy.Config    `yaml:"proxy"`
	Storage   storage.Config  `yaml:"storage"`
	API       APIConfig       `yaml:"api"`
	Metrics   MetricsConfig   `yaml:"metrics"`
	Logging   LoggingConfig   `yaml:"logging"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Webhook   WebhookConfig   `yaml:"webhook"`
}

func Load() (*Config, error) {
	cfg := Default()

	path := os.Getenv("TELLONYM_CONFIG")
	if path == "" {
		path = "configs/config.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.normalize()
	return cfg, nil
}

func Default() *Config {
	cfg := &Config{}

	cfg.App = AppConfig{
		Name:        "Tellonym Username Checker",
		Version:     "1.0.0",
		Environment: "production",
	}

	cfg.Checker = checker.Config{
		RequestTimeout:  10 * time.Second,
		MaxRetries:      3,
		RetryBackoff:    time.Second,
		JitterMax:       500 * time.Millisecond,
		BatchSize:       100,
		QueueSize:       10000,
		FollowRedirects: false,
		MaxConcurrent:   50,
		UsernameRules: checker.UsernameRules{
			MinLength:           3,
			MaxLength:           30,
			AllowLetters:        true,
			AllowNumbers:        true,
			AllowUnderscore:     true,
			AllowDot:            true,
			DisallowLeadingDot:  true,
			DisallowTrailingDot: true,
			MaxConsecutiveDots:  1,
		},
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		},
	}

	cfg.Proxy = proxy.Config{
		Enabled:             true,
		Types:               []string{"*"},
		MaxConsecutiveFails: 3,
		HealthCheckInterval: 5 * time.Minute,
		RotationStrategy:    "round-robin",
		ValidationURL:       "https://httpbin.org/ip",
		ValidationTimeout:   10 * time.Second,
		MaxLifetime:         time.Hour,
	}

	cfg.Storage = storage.Config{
		Type:          "file",
		FilePath:      "found_usernames.txt",
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
		Postgresql: storage.PostgresConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			DBName:   "tellonym",
			SSLMode:  "disable",
		},
	}

	cfg.API = APIConfig{Enabled: false, Host: "localhost", Port: 8080}
	cfg.API.Auth.Enabled = true
	cfg.API.Auth.APIKeys = []string{"your-api-key-here"}

	cfg.Metrics = MetricsConfig{Enabled: true, Port: 2112, Path: "/metrics"}
	cfg.Logging = LoggingConfig{Level: "info", OutputPath: "logs/app.log", Encoding: "console"}
	cfg.RateLimit = RateLimitConfig{Enabled: true, RequestsPerSec: 50, Burst: 10, PerIP: false}
	cfg.Webhook = WebhookConfig{
		Enabled:   false,
		URL:       "",
		Username:  "Gon",
		AvatarURL: "https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg",
		Timeout:   10 * time.Second,
	}

	return cfg
}

func (c *Config) normalize() {
	if c.Checker.RequestTimeout <= 0 {
		c.Checker.RequestTimeout = 10 * time.Second
	}
	if c.Checker.BatchSize <= 0 {
		c.Checker.BatchSize = 100
	}
	if c.Checker.QueueSize <= 0 {
		c.Checker.QueueSize = 10000
	}
	c.Checker.UsernameRules = checker.NormalizeUsernameRules(c.Checker.UsernameRules)

	if c.Proxy.HealthCheckInterval <= 0 {
		c.Proxy.HealthCheckInterval = 5 * time.Minute
	}
	if c.Proxy.ValidationTimeout <= 0 {
		c.Proxy.ValidationTimeout = 10 * time.Second
	}
	if len(c.Proxy.Types) == 0 {
		c.Proxy.Types = []string{"*"}
	}

	if c.Storage.BufferSize <= 0 {
		c.Storage.BufferSize = 100
	}
	if c.Storage.FlushInterval <= 0 {
		c.Storage.FlushInterval = 5 * time.Second
	}
	if c.Storage.FilePath == "" {
		c.Storage.FilePath = "found_usernames.txt"
	}

	if c.Webhook.Timeout <= 0 {
		c.Webhook.Timeout = 10 * time.Second
	}
	if c.Webhook.Username == "" {
		c.Webhook.Username = "Gon"
	}
	if c.Webhook.AvatarURL == "" {
		c.Webhook.AvatarURL = "https://i.pinimg.com/736x/dd/f4/75/ddf475e4b9767235362fc1cf3a16ed1c.jpg"
	}
}
