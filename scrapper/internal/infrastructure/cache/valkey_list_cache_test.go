package cache

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"
)

func TestValkeyListCacheSetGetDeleteWithTestcontainers(t *testing.T) {
	ctx := context.Background()
	addr := startValkeyContainer(t, ctx)

	listCache, err := NewValkeyListCache(
		[]string{addr},
		"",
		"",
		false,
		0,
	)
	if err != nil {
		t.Fatalf("create valkey list cache: %v", err)
	}
	defer listCache.Close()

	body := []byte(`[{"url":"https://github.com/golang/go","tags":["go"]}]`)

	if err := listCache.Set(ctx, 1, body, time.Minute); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	got, ok, err := listCache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if string(got) != string(body) {
		t.Fatalf("unexpected cached body: got %s, want %s", string(got), string(body))
	}

	if err := listCache.Delete(ctx, 1); err != nil {
		t.Fatalf("delete cache: %v", err)
	}

	_, ok, err = listCache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if ok {
		t.Fatalf("expected cache miss after delete")
	}
}

func TestValkeyListCacheTTLWithTestcontainers(t *testing.T) {
	ctx := context.Background()
	addr := startValkeyContainer(t, ctx)

	listCache, err := NewValkeyListCache(
		[]string{addr},
		"",
		"",
		false,
		0,
	)
	if err != nil {
		t.Fatalf("create valkey list cache: %v", err)
	}
	defer listCache.Close()

	body := []byte(`[]`)

	if err := listCache.Set(ctx, 2, body, time.Second); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	_, ok, err := listCache.Get(ctx, 2)
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if !ok {
		t.Fatalf("expected cache hit before ttl expiration")
	}

	time.Sleep(1500 * time.Millisecond)

	_, ok, err = listCache.Get(ctx, 2)
	if err != nil {
		t.Fatalf("get cache after ttl: %v", err)
	}
	if ok {
		t.Fatalf("expected cache miss after ttl expiration")
	}
}

func TestValkeyListCacheClientSideCachingWithTestcontainers(t *testing.T) {
	ctx := context.Background()
	addr := startValkeyContainer(t, ctx)

	listCache, err := NewValkeyListCache(
		[]string{addr},
		"",
		"",
		true,
		time.Minute,
	)
	if err != nil {
		t.Fatalf("create valkey list cache with client-side caching: %v", err)
	}
	defer listCache.Close()

	body := []byte(`[{"url":"https://github.com/valkey-io/valkey","tags":["cache"]}]`)

	if err := listCache.Set(ctx, 3, body, time.Minute); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	got, ok, err := listCache.Get(ctx, 3)
	if err != nil {
		t.Fatalf("first get cache: %v", err)
	}
	if !ok {
		t.Fatalf("expected first cache hit from Valkey")
	}
	if string(got) != string(body) {
		t.Fatalf("unexpected cached body: got %s, want %s", string(got), string(body))
	}

	got, ok, err = listCache.Get(ctx, 3)
	if err != nil {
		t.Fatalf("second get cache: %v", err)
	}
	if !ok {
		t.Fatalf("expected second cache hit")
	}
	if string(got) != string(body) {
		t.Fatalf("unexpected cached body on second get: got %s, want %s", string(got), string(body))
	}
}

func startValkeyContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	valkeyContainer, err := tcvalkey.Run(ctx, "valkey/valkey:7.2.5")
	if err != nil {
		t.Fatalf("start valkey container: %v", err)
	}

	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(valkeyContainer); err != nil {
			t.Fatalf("terminate valkey container: %v", err)
		}
	})

	connectionString, err := valkeyContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("get valkey connection string: %v", err)
	}

	return extractAddress(connectionString)
}

func extractAddress(connectionString string) string {
	parsed, err := url.Parse(connectionString)
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}

	connectionString = strings.TrimPrefix(connectionString, "redis://")
	connectionString = strings.TrimPrefix(connectionString, "valkey://")

	return connectionString
}
