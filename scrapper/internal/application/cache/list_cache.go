package cache

import (
	"context"
	"time"
)

type ListCache interface {
	Get(ctx context.Context, chatID int64) ([]byte, bool, error)
	Set(ctx context.Context, chatID int64, body []byte, ttl time.Duration) error
	Delete(ctx context.Context, chatID int64) error
}
