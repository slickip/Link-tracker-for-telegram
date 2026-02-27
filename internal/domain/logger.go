package domain

type Logger interface {
	Debug(msg string, arg ...any)
	Info(msg string, arg ...any)
	Warn(msg string, arg ...any)
	Error(msg string, arg ...any)
}
