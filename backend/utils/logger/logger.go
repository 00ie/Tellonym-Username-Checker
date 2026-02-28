package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	Level      string
	OutputPath string
	Encoding   string
}

type Logger struct {
	logger *slog.Logger
	file   *os.File
}

func NewLogger(cfg Config) *Logger {
	level := &slog.LevelVar{}
	switch cfg.Level {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	output := io.Writer(os.Stdout)
	var file *os.File

	if cfg.OutputPath != "" {
		_ = os.MkdirAll(filepath.Dir(cfg.OutputPath), 0o755)
		f, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			file = f
			output = io.MultiWriter(os.Stdout, f)
		}
	}

	handler := slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})

	return &Logger{logger: slog.New(handler), file: file}
}

func (l *Logger) Named(name string) *Logger {
	if l == nil || l.logger == nil {
		return NewLogger(Config{})
	}
	return &Logger{logger: l.logger.With("component", name), file: l.file}
}

func (l *Logger) Debug(msg string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Debug(msg, args...)
	}
}

func (l *Logger) Info(msg string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Info(msg, args...)
	}
}

func (l *Logger) Warn(msg string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Warn(msg, args...)
	}
}

func (l *Logger) Error(msg string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Error(msg, args...)
	}
}

func (l *Logger) Sync() {
	if l != nil && l.file != nil {
		_ = l.file.Close()
	}
}
