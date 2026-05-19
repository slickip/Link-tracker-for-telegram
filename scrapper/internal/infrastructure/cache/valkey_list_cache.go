package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"
)

type ValkeyListCache struct {
	client valkey.Client
	rdb    valkeycompat.Cmdable
}

func NewValkeyListCache(
	addresses []string,
	username string,
	password string,
	clientSideCacheEnabled bool,
) (*ValkeyListCache, error) {
	if len(addresses) == 0 {
		return nil, errors.New("valkey addresses must not be empty")
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  addresses,
		Username:     username,
		Password:     password,
		DisableCache: !clientSideCacheEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create valkey client: %w", err)
	}

	return &ValkeyListCache{
		client: client,
		rdb:    valkeycompat.NewAdapter(client),
	}, nil
}

func (c *ValkeyListCache) Close() {
	c.client.Close()
}

func (c *ValkeyListCache) Get(ctx context.Context, chatID int64) ([]byte, bool, error) {
	key := cacheKey(chatID)

	value, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, valkeycompat.Nil) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return []byte(value), true, nil
}

func (c *ValkeyListCache) Set(ctx context.Context, chatID int64, body []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("cache ttl must be positive")
	}

	key := cacheKey(chatID)

	return c.rdb.Set(ctx, key, string(body), ttl).Err()
}

func (c *ValkeyListCache) Delete(ctx context.Context, chatID int64) error {
	key := cacheKey(chatID)

	return c.rdb.Del(ctx, key).Err()
}

func cacheKey(chatID int64) string {
	return strconv.FormatInt(chatID, 10)
}
