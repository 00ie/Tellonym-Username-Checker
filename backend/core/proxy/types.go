package proxy

import (
	"net/url"
	"time"
)

type Proxy struct {
	URL           *url.URL
	Type          string
	LastUsed      time.Time
	FailCount     int32
	SuccessCount  int32
	AvgResponseMs int64
	IsAlive       bool
	LastCheck     time.Time
	Country       string
}

type Config struct {
	Enabled             bool          `yaml:"enabled"`
	Types               []string      `yaml:"types"`
	MaxConsecutiveFails int           `yaml:"max_consecutive_fails"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	RotationStrategy    string        `yaml:"rotation_strategy"`
	ValidationURL       string        `yaml:"validation_url"`
	ValidationTimeout   time.Duration `yaml:"validation_timeout"`
	MaxLifetime         time.Duration `yaml:"max_lifetime"`
}
