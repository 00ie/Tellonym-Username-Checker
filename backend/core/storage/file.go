package storage

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileStorage struct {
	path          string
	queue         chan string
	flushInterval time.Duration
	quit          chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	pending       []string
}

func NewFileStorage(path string, bufferSize int, flushInterval time.Duration) (*FileStorage, error) {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	_ = file.Close()

	fs := &FileStorage{
		path:          path,
		queue:         make(chan string, bufferSize),
		flushInterval: flushInterval,
		quit:          make(chan struct{}),
		pending:       make([]string, 0, bufferSize),
	}

	fs.wg.Add(1)
	go fs.loop()

	return fs, nil
}

func (f *FileStorage) Add(username string) error {
	select {
	case f.queue <- username:
	default:
		f.mu.Lock()
		f.pending = append(f.pending, username)
		f.mu.Unlock()
	}
	return nil
}

func (f *FileStorage) loop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-f.quit:
			f.flush()
			return
		case username := <-f.queue:
			f.mu.Lock()
			f.pending = append(f.pending, username)
			f.mu.Unlock()
		case <-ticker.C:
			f.flush()
		}
	}
}

func (f *FileStorage) flush() {
	f.mu.Lock()
	if len(f.pending) == 0 {
		f.mu.Unlock()
		return
	}

	batch := make([]string, len(f.pending))
	copy(batch, f.pending)
	f.pending = f.pending[:0]
	f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range batch {
		_, _ = writer.WriteString(item + "\n")
	}
	_ = writer.Flush()
}

func (f *FileStorage) LoadAll() ([]string, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	items := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		items = append(items, line)
	}

	return items, scanner.Err()
}

func (f *FileStorage) Clear() error {
	f.mu.Lock()
	f.pending = f.pending[:0]
	f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func (f *FileStorage) Close() {
	close(f.quit)
	f.wg.Wait()
}
