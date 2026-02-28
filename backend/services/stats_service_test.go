package services

import (
	"testing"
	"time"

	"tellonym-checker/backend/utils/logger"
)

func TestStatsServiceClearDashboardData(t *testing.T) {
	service := NewStatsService(nil, nil, logger.NewLogger(logger.Config{Level: "error"}), "")
	service.recentFinds = []string{"one", "two"}
	service.history = []HistoricalStats{
		{
			Timestamp: "2026-02-28T00:00:00Z",
			Attempts:  12,
			Found:     1,
			Errors:    3,
			Rate:      2.3,
		},
	}

	service.ClearDashboardData()

	if len(service.recentFinds) != 0 {
		t.Fatalf("expected recent finds to be cleared")
	}
	if len(service.history) != 0 {
		t.Fatalf("expected history to be cleared")
	}
}

func TestGetHistoricalStatsRejectsInvalidRange(t *testing.T) {
	service := NewStatsService(nil, nil, logger.NewLogger(logger.Config{Level: "error"}), "")
	_, err := service.GetHistoricalStats("2026-02-28T10:00:00Z", "2026-02-28T09:00:00Z")
	if err == nil {
		t.Fatalf("expected range validation error")
	}
}

func TestGetHistoricalStatsCapsResult(t *testing.T) {
	service := NewStatsService(nil, nil, logger.NewLogger(logger.Config{Level: "error"}), "")
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]HistoricalStats, 0, maxHistoricalRowsResponse+300)
	for i := 0; i < maxHistoricalRowsResponse+300; i++ {
		rows = append(rows, HistoricalStats{
			Timestamp: base.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Attempts:  uint64(i),
			Found:     0,
			Errors:    0,
			Rate:      1,
		})
	}
	service.history = rows

	result, err := service.GetHistoricalStats("", "")
	if err != nil {
		t.Fatalf("get history failed: %v", err)
	}
	if len(result) != maxHistoricalRowsResponse {
		t.Fatalf("expected %d rows, got %d", maxHistoricalRowsResponse, len(result))
	}
	if result[0].Attempts != 300 {
		t.Fatalf("expected capped history to keep latest rows")
	}
}
