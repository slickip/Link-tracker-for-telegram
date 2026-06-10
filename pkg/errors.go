package pkg

import "errors"

var (
	ErrUnexpectedError      = errors.New("unexpected error")
	ErrEmptyStartResponse   = errors.New("expected non-empty response for /start")
	ErrEmptyHelpResponse    = errors.New("expected non-empty response for /help")
	ErrEmptyUnknownResponse = errors.New("expected error message for unknown command")
	ErrChatNotFound         = errors.New("chat not found")
	ErrLinkAlreadyTracked   = errors.New("link already tracked")
	ErrLinkNotTracked       = errors.New("link not tracked")
	ErrInvalidURL           = errors.New("invalid url")
	ErrInvalidAPIResponse   = errors.New("invalid api response")
	ErrLinkNotFound         = errors.New("link is not tracked by this chat")
	ErrTagNotFound          = errors.New("tag not found")
	ErrTagExists            = errors.New("tag already exists")
	ErrInvalidRequest       = errors.New("invalid request")
)
