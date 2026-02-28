package checker

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"tellonym-checker/backend/core/proxy"
	"tellonym-checker/backend/utils/logger"
	"tellonym-checker/backend/utils/ratelimit"
)

type Config struct {
	RequestTimeout  time.Duration `yaml:"request_timeout"`
	MaxRetries      int           `yaml:"max_retries"`
	RetryBackoff    time.Duration `yaml:"retry_backoff"`
	JitterMax       time.Duration `yaml:"jitter_max"`
	BatchSize       int           `yaml:"batch_size"`
	QueueSize       int           `yaml:"queue_size"`
	UserAgents      []string      `yaml:"user_agents"`
	FollowRedirects bool          `yaml:"follow_redirects"`
	MaxConcurrent   int           `yaml:"max_concurrent"`
	UsernameRules   UsernameRules `yaml:"username_rules"`
}

type Checker struct {
	config        *Config
	logger        *logger.Logger
	proxyManager  *proxy.Manager
	workers       []*Worker
	taskQueue     chan Task
	resultQueue   chan Result
	quit          chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	stats         Stats
	limiter       *ratelimit.Limiter
	isRunning     bool
	isPaused      bool
	attempts      uint64
	found         uint64
	errors        uint64
	rateLimited   uint64
	responseTotal uint64
	responseCount uint64
	onFound       []func(string)
	onResult      []func(Result)
}

func NewChecker(cfg *Config, logger *logger.Logger, pm *proxy.Manager) *Checker {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 10000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	cfg.UsernameRules = NormalizeUsernameRules(cfg.UsernameRules)

	return &Checker{
		config:       cfg,
		logger:       logger.Named("checker"),
		proxyManager: pm,
		taskQueue:    make(chan Task, cfg.QueueSize),
		resultQueue:  make(chan Result, cfg.QueueSize),
		quit:         make(chan struct{}),
		limiter:      ratelimit.NewLimiter(100, 10),
	}
}

func (c *Checker) SetOnFound(fn func(string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if fn != nil {
		c.onFound = append(c.onFound, fn)
	}
}

func (c *Checker) SetOnResult(fn func(Result)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if fn != nil {
		c.onResult = append(c.onResult, fn)
	}
}

func (c *Checker) Start(usernameLength int, numWorkers int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return fmt.Errorf("checker is already running")
	}

	if numWorkers <= 0 {
		numWorkers = 1
	}

	rules := NormalizeUsernameRules(c.config.UsernameRules)
	if usernameLength <= 0 {
		usernameLength = rules.MinLength
	}
	if usernameLength < rules.MinLength {
		usernameLength = rules.MinLength
	}
	if usernameLength > rules.MaxLength {
		usernameLength = rules.MaxLength
	}

	c.taskQueue = make(chan Task, c.config.QueueSize)
	c.resultQueue = make(chan Result, c.config.QueueSize)
	c.quit = make(chan struct{})

	atomic.StoreUint64(&c.attempts, 0)
	atomic.StoreUint64(&c.found, 0)
	atomic.StoreUint64(&c.errors, 0)
	atomic.StoreUint64(&c.rateLimited, 0)
	atomic.StoreUint64(&c.responseTotal, 0)
	atomic.StoreUint64(&c.responseCount, 0)

	c.stats = Stats{StartTime: time.Now()}
	c.isRunning = true
	c.isPaused = false
	c.workers = make([]*Worker, numWorkers)

	c.logger.Info("starting checker", "workers", numWorkers, "username_length", usernameLength)

	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(i, c.config, c.logger, c.proxyManager, c.limiter)
		c.workers[i] = worker
		c.wg.Add(1)
		go c.runWorker(worker)
	}

	c.wg.Add(1)
	go c.generateTasks(usernameLength, rules)

	c.wg.Add(1)
	go c.processResults()

	c.wg.Add(1)
	go c.calculateStats()

	return nil
}

func (c *Checker) Stop() {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = false
	close(c.quit)
	c.mu.Unlock()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("checker stopped")
	case <-time.After(10 * time.Second):
		c.logger.Warn("checker stop timeout")
	}
}

func (c *Checker) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning || c.isPaused {
		return
	}

	c.isPaused = true
	c.logger.Info("checker paused")
}

func (c *Checker) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning || !c.isPaused {
		return
	}

	c.isPaused = false
	c.logger.Info("checker resumed")
}

func (c *Checker) UpdateConfig(settings Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if settings.RequestTimeout > 0 {
		c.config.RequestTimeout = settings.RequestTimeout
	}
	if settings.MaxRetries >= 0 {
		c.config.MaxRetries = settings.MaxRetries
	}
	if settings.RetryBackoff > 0 {
		c.config.RetryBackoff = settings.RetryBackoff
	}
	if settings.BatchSize > 0 {
		c.config.BatchSize = settings.BatchSize
	}
	if settings.UsernameRules.MinLength > 0 || settings.UsernameRules.MaxLength > 0 {
		c.config.UsernameRules = NormalizeUsernameRules(settings.UsernameRules)
	}
}

func (c *Checker) runWorker(worker *Worker) {
	defer c.wg.Done()

	for {
		select {
		case <-c.quit:
			return
		default:
		}

		if c.IsPaused() {
			select {
			case <-c.quit:
				return
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}

		select {
		case <-c.quit:
			return
		case task := <-c.taskQueue:
			atomic.AddUint64(&c.attempts, 1)

			start := time.Now()
			result := worker.Check(task)
			result.ResponseTime = time.Since(start)

			atomic.AddUint64(&c.responseTotal, uint64(result.ResponseTime.Nanoseconds()))
			atomic.AddUint64(&c.responseCount, 1)

			if result.Error != nil {
				atomic.AddUint64(&c.errors, 1)
			}
			if result.StatusCode == http.StatusTooManyRequests {
				atomic.AddUint64(&c.rateLimited, 1)
			}

			if result.Found {
				atomic.AddUint64(&c.found, 1)
				c.mu.RLock()
				onFound := append([]func(string){}, c.onFound...)
				c.mu.RUnlock()
				for _, fn := range onFound {
					fn(result.Username)
				}
			}

			select {
			case c.resultQueue <- result:
			case <-c.quit:
				return
			}
		}
	}
}

func (c *Checker) generateTasks(length int, rules UsernameRules) {
	defer c.wg.Done()

	generator := NewUsernameGenerator(length, rules)

	for {
		select {
		case <-c.quit:
			return
		default:
		}

		select {
		case <-c.quit:
			return
		default:
			for i := 0; i < c.config.BatchSize; i++ {
				task := Task{
					Username: generator.Generate(),
					ID:       fmt.Sprintf("%d-%d", time.Now().UnixNano(), i),
					Priority: 1,
					Retries:  c.config.MaxRetries,
				}

				select {
				case <-c.quit:
					return
				case c.taskQueue <- task:
				}
			}
		}
	}
}

func (c *Checker) GetConfig() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cfg := *c.config
	cfg.UserAgents = append([]string{}, c.config.UserAgents...)
	cfg.UsernameRules = NormalizeUsernameRules(c.config.UsernameRules)

	return cfg
}

func (c *Checker) processResults() {
	defer c.wg.Done()

	for {
		select {
		case <-c.quit:
			return
		default:
		}

		select {
		case <-c.quit:
			return
		case result := <-c.resultQueue:
			c.mu.RLock()
			onResult := append([]func(Result){}, c.onResult...)
			c.mu.RUnlock()
			for _, fn := range onResult {
				fn(result)
			}
		}
	}
}

func (c *Checker) calculateStats() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastAttempts uint64
	lastTick := time.Now()

	for {
		select {
		case <-c.quit:
			return
		case <-ticker.C:
			now := time.Now()
			attempts := atomic.LoadUint64(&c.attempts)
			elapsed := now.Sub(lastTick).Seconds()

			c.mu.Lock()
			if elapsed > 0 {
				c.stats.Rate = float64(attempts-lastAttempts) / elapsed
			}
			c.stats.ElapsedTime = now.Sub(c.stats.StartTime)
			c.mu.Unlock()

			lastAttempts = attempts
			lastTick = now
		}
	}
}

func (c *Checker) GetStats() Stats {
	c.mu.RLock()
	s := c.stats
	s.IsRunning = c.isRunning
	s.IsPaused = c.isPaused
	c.mu.RUnlock()

	s.Attempts = atomic.LoadUint64(&c.attempts)
	s.Found = atomic.LoadUint64(&c.found)
	s.Errors = atomic.LoadUint64(&c.errors)
	s.RateLimited = atomic.LoadUint64(&c.rateLimited)

	total := atomic.LoadUint64(&c.responseTotal)
	count := atomic.LoadUint64(&c.responseCount)
	if count > 0 {
		s.AvgResponseTime = time.Duration(total / count)
	}

	return s
}

func (c *Checker) ResetStats() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return fmt.Errorf("checker must be stopped before clearing dashboard data")
	}

	atomic.StoreUint64(&c.attempts, 0)
	atomic.StoreUint64(&c.found, 0)
	atomic.StoreUint64(&c.errors, 0)
	atomic.StoreUint64(&c.rateLimited, 0)
	atomic.StoreUint64(&c.responseTotal, 0)
	atomic.StoreUint64(&c.responseCount, 0)

	c.stats = Stats{}

	return nil
}

func (c *Checker) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *Checker) IsPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isPaused
}
