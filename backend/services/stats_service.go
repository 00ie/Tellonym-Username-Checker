package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"tellonym-checker/backend/core/checker"
	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/utils/logger"
)

type LiveStats struct {
	Attempts    uint64   `json:"attempts"`
	Found       uint64   `json:"found"`
	Errors      uint64   `json:"errors"`
	RateLimited uint64   `json:"rateLimited"`
	Rate        float64  `json:"rate"`
	AvgResponse int64    `json:"avgResponse"`
	Uptime      int64    `json:"uptime"`
	IsRunning   bool     `json:"isRunning"`
	IsPaused    bool     `json:"isPaused"`
	RecentFinds []string `json:"recentFinds"`
}

type HistoricalStats struct {
	Timestamp string  `json:"timestamp"`
	Attempts  uint64  `json:"attempts"`
	Found     uint64  `json:"found"`
	Errors    uint64  `json:"errors"`
	Rate      float64 `json:"rate"`
}

type StatsService struct {
	checker      *checker.Checker
	proxyManager *proxy.Manager
	logger       *logger.Logger
	exportDir    string
	mu           sync.RWMutex
	recentFinds  []string
	history      []HistoricalStats
	stopChan     chan struct{}
	stopOnce     sync.Once
}

func NewStatsService(c *checker.Checker, pm *proxy.Manager, logger *logger.Logger, exportDir string) *StatsService {
	if exportDir == "" {
		exportDir = "exports"
	}

	s := &StatsService{
		checker:      c,
		proxyManager: pm,
		logger:       logger.Named("stats-service"),
		exportDir:    exportDir,
		recentFinds:  make([]string, 0, 50),
		history:      make([]HistoricalStats, 0, 4096),
		stopChan:     make(chan struct{}),
	}

	if c != nil {
		c.SetOnFound(s.pushRecentFind)
	}

	return s
}

func (s *StatsService) StartStreaming(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	historyInterval := 10 * time.Second
	maxHistoryRows := int((31 * 24 * time.Hour) / historyInterval)
	lastHistoryWrite := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			live := s.GetLiveStats()
			runtime.EventsEmit(ctx, "stats:update", live)

			now := time.Now()
			if lastHistoryWrite.IsZero() || now.Sub(lastHistoryWrite) >= historyInterval {
				s.mu.Lock()
				s.history = append(s.history, HistoricalStats{
					Timestamp: now.Format(time.RFC3339),
					Attempts:  live.Attempts,
					Found:     live.Found,
					Errors:    live.Errors,
					Rate:      live.Rate,
				})
				if len(s.history) > maxHistoryRows {
					s.history = s.history[len(s.history)-maxHistoryRows:]
				}
				s.mu.Unlock()
				lastHistoryWrite = now
			}
		}
	}
}

func (s *StatsService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

func (s *StatsService) GetLiveStats() LiveStats {
	if s.checker == nil {
		return LiveStats{}
	}

	stats := s.checker.GetStats()

	s.mu.RLock()
	recent := make([]string, len(s.recentFinds))
	copy(recent, s.recentFinds)
	s.mu.RUnlock()

	return LiveStats{
		Attempts:    stats.Attempts,
		Found:       stats.Found,
		Errors:      stats.Errors,
		RateLimited: stats.RateLimited,
		Rate:        stats.Rate,
		AvgResponse: stats.AvgResponseTime.Milliseconds(),
		Uptime:      int64(stats.ElapsedTime.Seconds()),
		IsRunning:   stats.IsRunning,
		IsPaused:    stats.IsPaused,
		RecentFinds: recent,
	}
}

func (s *StatsService) GetHistoricalStats(from, to string) ([]HistoricalStats, error) {
	s.mu.RLock()
	historyCopy := make([]HistoricalStats, len(s.history))
	copy(historyCopy, s.history)
	s.mu.RUnlock()

	if from == "" && to == "" {
		return historyCopy, nil
	}

	fromTime, err := parseOptionalTime(from)
	if err != nil {
		return nil, err
	}

	toTime, err := parseOptionalTime(to)
	if err != nil {
		return nil, err
	}

	filtered := make([]HistoricalStats, 0, len(historyCopy))
	for _, row := range historyCopy {
		rowTime, err := time.Parse(time.RFC3339, row.Timestamp)
		if err != nil {
			continue
		}
		if !fromTime.IsZero() && rowTime.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && rowTime.After(toTime) {
			continue
		}
		filtered = append(filtered, row)
	}

	return filtered, nil
}

func (s *StatsService) Export(format string) (string, error) {
	rows, err := s.GetHistoricalStats("", "")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(s.exportDir, 0o755); err != nil {
		return "", err
	}

	now := time.Now().Format("20060102-150405")

	switch format {
	case "csv":
		path := filepath.Join(s.exportDir, fmt.Sprintf("stats-%s.csv", now))
		if err := exportCSV(path, rows); err != nil {
			return "", err
		}
		return path, nil
	case "json":
		path := filepath.Join(s.exportDir, fmt.Sprintf("stats-%s.json", now))
		if err := exportJSON(path, rows); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *StatsService) pushRecentFind(username string) {
	if username == "" {
		return
	}

	s.mu.Lock()
	s.recentFinds = append(s.recentFinds, username)
	if len(s.recentFinds) > 30 {
		s.recentFinds = s.recentFinds[len(s.recentFinds)-30:]
	}
	s.mu.Unlock()
}

func parseOptionalTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return parsed, nil
	}

	parsed, err = time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %s", raw)
	}

	return parsed, nil
}

func exportCSV(path string, rows []HistoricalStats) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"timestamp", "attempts", "found", "errors", "rate"}); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write([]string{
			row.Timestamp,
			fmt.Sprintf("%d", row.Attempts),
			fmt.Sprintf("%d", row.Found),
			fmt.Sprintf("%d", row.Errors),
			fmt.Sprintf("%.2f", row.Rate),
		}); err != nil {
			return err
		}
	}

	return nil
}

func exportJSON(path string, rows []HistoricalStats) error {
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
