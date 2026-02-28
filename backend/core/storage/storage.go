package storage

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Config struct {
	Type          string         `yaml:"type"`
	FilePath      string         `yaml:"file_path"`
	BufferSize    int            `yaml:"buffer_size"`
	FlushInterval time.Duration  `yaml:"flush_interval"`
	Postgresql    PostgresConfig `yaml:"postgresql"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type Storage struct {
	cfg            *Config
	fileStorage    *FileStorage
	dbStorage      *DatabaseStorage
	foundUsernames map[string]struct{}
	mu             sync.RWMutex
}

func NewStorage(cfg *Config) (*Storage, error) {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.FilePath == "" {
		cfg.FilePath = "found_usernames.txt"
	}
	if cfg.Type == "" {
		cfg.Type = "file"
	}

	s := &Storage{
		cfg:            cfg,
		foundUsernames: make(map[string]struct{}),
	}

	switch cfg.Type {
	case "postgres":
		db, err := NewDatabaseStorage(cfg.Postgresql)
		if err != nil {
			return nil, err
		}
		s.dbStorage = db
	default:
		fs, err := NewFileStorage(cfg.FilePath, cfg.BufferSize, cfg.FlushInterval)
		if err != nil {
			return nil, err
		}
		s.fileStorage = fs

		existing, err := fs.LoadAll()
		if err == nil {
			for _, username := range existing {
				s.foundUsernames[username] = struct{}{}
			}
		}
	}

	return s, nil
}

func (s *Storage) AddFound(username string) error {
	if username == "" {
		return nil
	}

	s.mu.Lock()
	if _, exists := s.foundUsernames[username]; exists {
		s.mu.Unlock()
		return nil
	}
	s.foundUsernames[username] = struct{}{}
	s.mu.Unlock()

	if s.fileStorage != nil {
		return s.fileStorage.Add(username)
	}

	if s.dbStorage != nil {
		return s.dbStorage.Add(username)
	}

	return nil
}

func (s *Storage) GetFoundUsernames() []string {
	s.mu.RLock()
	out := make([]string, 0, len(s.foundUsernames))
	for username := range s.foundUsernames {
		out = append(out, username)
	}
	s.mu.RUnlock()

	sort.Strings(out)
	return out
}

func (s *Storage) ClearFound() error {
	s.mu.Lock()
	s.foundUsernames = make(map[string]struct{})
	s.mu.Unlock()

	if s.fileStorage != nil {
		return s.fileStorage.Clear()
	}

	if s.dbStorage != nil {
		return s.dbStorage.Clear()
	}

	return nil
}

func (s *Storage) Close() {
	if s.fileStorage != nil {
		s.fileStorage.Close()
	}
	if s.dbStorage != nil {
		s.dbStorage.Close()
	}
}

func (s *Storage) String() string {
	return fmt.Sprintf("Storage{type=%s}", s.cfg.Type)
}
