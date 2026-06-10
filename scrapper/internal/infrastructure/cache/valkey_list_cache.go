package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

const decimalNum = 10

type ValkeyListCache struct {
	client valkey.Client

	clientSideCacheEnabled bool
	clientSideCacheTTL     time.Duration
}

func NewValkeyListCache(
	addresses []string,
	username string,
	password string,
	clientSideCacheEnabled bool,
	clientSideCacheTTL time.Duration,
) (*ValkeyListCache, error) {
	if len(addresses) == 0 {
		return nil, errors.New("valkey addresses must not be empty")
	}

	if clientSideCacheEnabled && clientSideCacheTTL <= 0 {
		return nil, errors.New("client side cache ttl must be positive")
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
		client:                 client,
		clientSideCacheEnabled: clientSideCacheEnabled,
		clientSideCacheTTL:     clientSideCacheTTL,
	}, nil
}

func (c *ValkeyListCache) Close() {
	c.client.Close()
}

func (c *ValkeyListCache) Get(ctx context.Context, chatID int64) ([]byte, bool, error) {
	if c.clientSideCacheEnabled {
		return c.getWithClientSideCache(ctx, chatID)
	}

	return c.getFromValkey(ctx, chatID)
}

func (c *ValkeyListCache) getFromValkey(ctx context.Context, chatID int64) ([]byte, bool, error) {
	key := cacheKey(chatID)

	resp := c.client.Do(ctx, c.client.B().Get().Key(key).Build())
	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	value, err := resp.AsBytes()
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (c *ValkeyListCache) getWithClientSideCache(ctx context.Context, chatID int64) ([]byte, bool, error) {
	key := cacheKey(chatID)

	resp := c.client.DoCache(
		ctx,
		c.client.B().Get().Key(key).Cache(),
		c.clientSideCacheTTL,
	)

	if err := resp.Error(); err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	value, err := resp.AsBytes()
	if err != nil {
		return nil, false, err
	}

	return value, true, nil
}

func (c *ValkeyListCache) Set(ctx context.Context, chatID int64, body []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("cache ttl must be positive")
	}

	key := cacheKey(chatID)

	return c.client.Do(
		ctx,
		c.client.B().
			Set().
			Key(key).
			Value(string(body)).
			Ex(ttl).
			Build(),
	).Error()
}

func (c *ValkeyListCache) Delete(ctx context.Context, chatID int64) error {
	key := cacheKey(chatID)

	return c.client.Do(
		ctx,
		c.client.B().Del().Key(key).Build(),
	).Error()
}

func cacheKey(chatID int64) string {
	return strconv.FormatInt(chatID, decimalNum)
}
