package logger

import (
	"log/slog"
	"os"
)

type Slog struct {
	logger *slog.Logger
}

func New(level slog.Level) *Slog {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &Slog{logger: slog.New(h)}
}

func (s *Slog) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}
func (s *Slog) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}
func (s *Slog) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}
func (s *Slog) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}
