package checker

import "time"

type Result struct {
	Username     string
	Found        bool
	Error        error
	ResponseTime time.Duration
	StatusCode   int
	ProxyUsed    string
	WorkerID     int
	Timestamp    time.Time
}

type Stats struct {
	Attempts        uint64
	Found           uint64
	Errors          uint64
	RateLimited     uint64
	Rate            float64
	AvgResponseTime time.Duration
	StartTime       time.Time
	ElapsedTime     time.Duration
	IsRunning       bool
	IsPaused        bool
}
