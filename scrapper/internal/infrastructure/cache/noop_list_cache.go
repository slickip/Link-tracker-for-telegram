package cache

import (
	"context"
	"time"
)

type NoopListCache struct{}

func NewNoopListCache() *NoopListCache {
	return &NoopListCache{}
}

func (c *NoopListCache) Get(ctx context.Context, chatID int64) ([]byte, bool, error) {
	return nil, false, nil
}

func (c *NoopListCache) Set(ctx context.Context, chatID int64, body []byte, ttl time.Duration) error {
	return nil
}

func (c *NoopListCache) Delete(ctx context.Context, chatID int64) error {
	return nil
}
