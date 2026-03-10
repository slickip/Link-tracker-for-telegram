package pkg

import "errors"

var (
	ErrUnexpectedError      = errors.New("unexpected error")
	ErrEmptyStartResponse   = errors.New("expected non-empty response for /start")
	ErrEmptyHelpResponse    = errors.New("expected non-empty response for /help")
	ErrEmptyUnknownResponse = errors.New("expected error message for unknown command")
)
