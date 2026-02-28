package storage

import (
	"errors"
)

type DatabaseStorage struct {
	items map[string]struct{}
}

func NewDatabaseStorage(cfg PostgresConfig) (*DatabaseStorage, error) {
	if cfg.Host == "" {
		return nil, errors.New("postgres host is required")
	}
	return &DatabaseStorage{items: make(map[string]struct{})}, nil
}

func (d *DatabaseStorage) Add(username string) error {
	d.items[username] = struct{}{}
	return nil
}

func (d *DatabaseStorage) GetAll() ([]string, error) {
	out := make([]string, 0, len(d.items))
	for item := range d.items {
		out = append(out, item)
	}
	return out, nil
}

func (d *DatabaseStorage) Clear() error {
	d.items = make(map[string]struct{})
	return nil
}

func (d *DatabaseStorage) Close() {}
