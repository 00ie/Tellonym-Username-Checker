package models

import "time"

type UsernameResult struct {
	Username   string    `json:"username"`
	Available  bool      `json:"available"`
	CheckedAt  time.Time `json:"checkedAt"`
	ProxyUsed  string    `json:"proxyUsed"`
	StatusCode int       `json:"statusCode"`
}

type ProxyMetrics struct {
	URL          string `json:"url"`
	Type         string `json:"type"`
	IsAlive      bool   `json:"isAlive"`
	FailCount    int32  `json:"failCount"`
	SuccessCount int32  `json:"successCount"`
	AvgResponse  int64  `json:"avgResponseMs"`
}
