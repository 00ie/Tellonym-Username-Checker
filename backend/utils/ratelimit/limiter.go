package ratelimit

import "time"

type Limiter struct {
	tokens chan struct{}
	ticker *time.Ticker
	quit   chan struct{}
}

func NewLimiter(requestsPerSecond int, burst int) *Limiter {
	if requestsPerSecond <= 0 {
		return nil
	}
	if burst <= 0 {
		burst = 1
	}

	interval := time.Second / time.Duration(requestsPerSecond)
	if interval <= 0 {
		interval = time.Nanosecond
	}

	l := &Limiter{
		tokens: make(chan struct{}, burst),
		ticker: time.NewTicker(interval),
		quit:   make(chan struct{}),
	}

	for i := 0; i < burst; i++ {
		l.tokens <- struct{}{}
	}

	go l.fill()

	return l
}

func (l *Limiter) fill() {
	for {
		select {
		case <-l.quit:
			return
		case <-l.ticker.C:
			select {
			case l.tokens <- struct{}{}:
			default:
			}
		}
	}
}

func (l *Limiter) Wait() {
	if l == nil {
		return
	}
	<-l.tokens
}

func (l *Limiter) Stop() {
	if l == nil {
		return
	}
	close(l.quit)
	l.ticker.Stop()
}
