package domain

import "context"

type Logger interface {
	Debug(msg string, arg ...any)
	Info(msg string, arg ...any)
	Warn(msg string, arg ...any)
	Error(msg string, arg ...any)
}

type Command interface {
	Name() string
	Execute(ctx context.Context, chatID int64, text string) (string, error)
}
