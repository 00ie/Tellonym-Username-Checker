package services

import (
	"fmt"
	"time"

	"tellonym-checker/backend/core/checker"
	"tellonym-checker/backend/utils/logger"
)

type CheckerStatus struct {
	Attempts    uint64  `json:"attempts"`
	Found       uint64  `json:"found"`
	Errors      uint64  `json:"errors"`
	Rate        float64 `json:"rate"`
	AvgResponse int64   `json:"avgResponse"`
	Uptime      int64   `json:"uptime"`
	IsRunning   bool    `json:"isRunning"`
	IsPaused    bool    `json:"isPaused"`
}

type CheckerSettings struct {
	RequestTimeoutMs    int  `json:"requestTimeoutMs"`
	MaxRetries          int  `json:"maxRetries"`
	BatchSize           int  `json:"batchSize"`
	MinLength           int  `json:"minLength"`
	MaxLength           int  `json:"maxLength"`
	AllowLetters        bool `json:"allowLetters"`
	AllowNumbers        bool `json:"allowNumbers"`
	AllowUnderscore     bool `json:"allowUnderscore"`
	AllowDot            bool `json:"allowDot"`
	DisallowLeadingDot  bool `json:"disallowLeadingDot"`
	DisallowTrailingDot bool `json:"disallowTrailingDot"`
	MaxConsecutiveDots  int  `json:"maxConsecutiveDots"`
}

type CheckerService struct {
	checker *checker.Checker
	logger  *logger.Logger
}

func NewCheckerService(c *checker.Checker, logger *logger.Logger) *CheckerService {
	return &CheckerService{checker: c, logger: logger.Named("checker-service")}
}

func (s *CheckerService) Start(usernameLength int, threads int) error {
	if s.checker == nil {
		return fmt.Errorf("checker is not initialized")
	}
	return s.checker.Start(usernameLength, threads)
}

func (s *CheckerService) Stop() {
	if s.checker != nil {
		s.checker.Stop()
	}
}

func (s *CheckerService) Pause() {
	if s.checker != nil {
		s.checker.Pause()
	}
}

func (s *CheckerService) Resume() {
	if s.checker != nil {
		s.checker.Resume()
	}
}

func (s *CheckerService) GetStatus() CheckerStatus {
	if s.checker == nil {
		return CheckerStatus{}
	}

	stats := s.checker.GetStats()

	return CheckerStatus{
		Attempts:    stats.Attempts,
		Found:       stats.Found,
		Errors:      stats.Errors,
		Rate:        stats.Rate,
		AvgResponse: stats.AvgResponseTime.Milliseconds(),
		Uptime:      int64(stats.ElapsedTime.Seconds()),
		IsRunning:   stats.IsRunning,
		IsPaused:    stats.IsPaused,
	}
}

func (s *CheckerService) UpdateSettings(settings CheckerSettings) error {
	if s.checker == nil {
		return fmt.Errorf("checker is not initialized")
	}
	if settings.MinLength > 0 && settings.MinLength < 3 {
		return fmt.Errorf("minimum username length must be at least 3")
	}
	if settings.MaxLength > 0 && settings.MaxLength > 30 {
		return fmt.Errorf("maximum username length must be at most 30")
	}
	if settings.MinLength > 0 && settings.MaxLength > 0 && settings.MinLength > settings.MaxLength {
		return fmt.Errorf("minimum length cannot be greater than maximum length")
	}
	if (settings.MinLength > 0 || settings.MaxLength > 0) &&
		!settings.AllowLetters &&
		!settings.AllowNumbers &&
		!settings.AllowUnderscore &&
		!settings.AllowDot {
		return fmt.Errorf("enable at least one allowed character group")
	}

	cfg := checker.Config{}

	if settings.RequestTimeoutMs > 0 {
		cfg.RequestTimeout = time.Duration(settings.RequestTimeoutMs) * time.Millisecond
	}
	if settings.MaxRetries >= 0 {
		cfg.MaxRetries = settings.MaxRetries
	}
	if settings.BatchSize > 0 {
		cfg.BatchSize = settings.BatchSize
	}
	if settings.MinLength > 0 || settings.MaxLength > 0 {
		cfg.UsernameRules = checker.UsernameRules{
			MinLength:           settings.MinLength,
			MaxLength:           settings.MaxLength,
			AllowLetters:        settings.AllowLetters,
			AllowNumbers:        settings.AllowNumbers,
			AllowUnderscore:     settings.AllowUnderscore,
			AllowDot:            settings.AllowDot,
			DisallowLeadingDot:  settings.DisallowLeadingDot,
			DisallowTrailingDot: settings.DisallowTrailingDot,
			MaxConsecutiveDots:  settings.MaxConsecutiveDots,
		}
	}

	s.checker.UpdateConfig(cfg)
	s.logger.Info("checker settings updated")

	return nil
}

func (s *CheckerService) GetSettings() CheckerSettings {
	if s.checker == nil {
		return CheckerSettings{}
	}

	cfg := s.checker.GetConfig()
	rules := checker.NormalizeUsernameRules(cfg.UsernameRules)

	return CheckerSettings{
		RequestTimeoutMs:    int(cfg.RequestTimeout.Milliseconds()),
		MaxRetries:          cfg.MaxRetries,
		BatchSize:           cfg.BatchSize,
		MinLength:           rules.MinLength,
		MaxLength:           rules.MaxLength,
		AllowLetters:        rules.AllowLetters,
		AllowNumbers:        rules.AllowNumbers,
		AllowUnderscore:     rules.AllowUnderscore,
		AllowDot:            rules.AllowDot,
		DisallowLeadingDot:  rules.DisallowLeadingDot,
		DisallowTrailingDot: rules.DisallowTrailingDot,
		MaxConsecutiveDots:  rules.MaxConsecutiveDots,
	}
}
