package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

func TestLinkServiceListLinksCachesResult(t *testing.T) {
	ctx := context.Background()

	linkRepo := &fakeLinkRepository{
		links: []domain.Link{
			{
				URL:  "https://github.com/golang/go",
				Tags: []string{"go"},
			},
		},
	}

	chatRepo := &fakeChatRepository{exists: true}
	listCache := newMemoryListCache()

	service := NewLinkServiceWithCache(
		linkRepo,
		chatRepo,
		listCache,
		time.Minute,
	)

	first, err := service.ListLinks(ctx, 1)
	if err != nil {
		t.Fatalf("first list links: %v", err)
	}

	second, err := service.ListLinks(ctx, 1)
	if err != nil {
		t.Fatalf("second list links: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cached result mismatch: first=%v second=%v", first, second)
	}

	if linkRepo.listCalls != 1 {
		t.Fatalf("expected repository List to be called once, got %d", linkRepo.listCalls)
	}

	if listCache.setCalls != 1 {
		t.Fatalf("expected cache Set to be called once, got %d", listCache.setCalls)
	}
}

func TestLinkServiceAddLinkInvalidatesCache(t *testing.T) {
	ctx := context.Background()

	linkRepo := &fakeLinkRepository{}
	chatRepo := &fakeChatRepository{exists: true}
	listCache := newMemoryListCache()

	service := NewLinkServiceWithCache(
		linkRepo,
		chatRepo,
		listCache,
		time.Minute,
	)

	if err := listCache.Set(ctx, 1, []byte(`[{"URL":"old"}]`), time.Minute); err != nil {
		t.Fatalf("prepare cache: %v", err)
	}

	err := service.AddLink(ctx, 1, domain.Link{
		URL:  "https://github.com/valkey-io/valkey",
		Tags: []string{"cache"},
	})
	if err != nil {
		t.Fatalf("add link: %v", err)
	}

	_, ok, err := listCache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get cache after invalidation: %v", err)
	}
	if ok {
		t.Fatalf("expected cache miss after AddLink")
	}

	if listCache.deleteCalls != 1 {
		t.Fatalf("expected cache Delete to be called once, got %d", listCache.deleteCalls)
	}
}

func TestLinkServiceRemoveLinkInvalidatesCache(t *testing.T) {
	ctx := context.Background()

	linkRepo := &fakeLinkRepository{}
	chatRepo := &fakeChatRepository{exists: true}
	listCache := newMemoryListCache()

	service := NewLinkServiceWithCache(
		linkRepo,
		chatRepo,
		listCache,
		time.Minute,
	)

	if err := listCache.Set(ctx, 1, []byte(`[{"URL":"old"}]`), time.Minute); err != nil {
		t.Fatalf("prepare cache: %v", err)
	}

	err := service.RemoveLink(ctx, 1, "https://github.com/golang/go")
	if err != nil {
		t.Fatalf("remove link: %v", err)
	}

	_, ok, err := listCache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get cache after invalidation: %v", err)
	}
	if ok {
		t.Fatalf("expected cache miss after RemoveLink")
	}

	if listCache.deleteCalls != 1 {
		t.Fatalf("expected cache Delete to be called once, got %d", listCache.deleteCalls)
	}
}

func TestLinkServiceRemoveLinksByTagInvalidatesCache(t *testing.T) {
	ctx := context.Background()

	linkRepo := &fakeLinkRepository{
		removeByTagCount: 2,
	}
	chatRepo := &fakeChatRepository{exists: true}
	listCache := newMemoryListCache()

	service := NewLinkServiceWithCache(
		linkRepo,
		chatRepo,
		listCache,
		time.Minute,
	)

	if err := listCache.Set(ctx, 1, []byte(`[{"URL":"old"}]`), time.Minute); err != nil {
		t.Fatalf("prepare cache: %v", err)
	}

	removed, err := service.RemoveLinksByTag(ctx, 1, "go")
	if err != nil {
		t.Fatalf("remove links by tag: %v", err)
	}
	if removed != 2 {
		t.Fatalf("unexpected removed count: got %d, want 2", removed)
	}

	_, ok, err := listCache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get cache after invalidation: %v", err)
	}
	if ok {
		t.Fatalf("expected cache miss after RemoveLinksByTag")
	}

	if listCache.deleteCalls != 1 {
		t.Fatalf("expected cache Delete to be called once, got %d", listCache.deleteCalls)
	}
}

type fakeChatRepository struct {
	exists bool
}

func (r *fakeChatRepository) Add(ctx context.Context, chatID int64) error {
	return nil
}

func (r *fakeChatRepository) Remove(ctx context.Context, chatID int64) error {
	return nil
}

func (r *fakeChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	return r.exists, nil
}

type fakeLinkRepository struct {
	links            []domain.Link
	listCalls        int
	addCalls         int
	removeCalls      int
	removeByTagCalls int
	removeByTagCount int64
}

func (r *fakeLinkRepository) Add(ctx context.Context, chatID int64, link domain.Link) error {
	r.addCalls++
	r.links = append(r.links, link)
	return nil
}

func (r *fakeLinkRepository) Remove(ctx context.Context, chatID int64, url string) error {
	r.removeCalls++
	return nil
}

func (r *fakeLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	r.listCalls++
	return r.links, nil
}

func (r *fakeLinkRepository) RemoveByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	r.removeByTagCalls++
	return r.removeByTagCount, nil
}

type memoryListCache struct {
	values      map[int64][]byte
	getCalls    int
	setCalls    int
	deleteCalls int
}

func newMemoryListCache() *memoryListCache {
	return &memoryListCache{
		values: make(map[int64][]byte),
	}
}

func (c *memoryListCache) Get(ctx context.Context, chatID int64) ([]byte, bool, error) {
	c.getCalls++

	value, ok := c.values[chatID]
	if !ok {
		return nil, false, nil
	}

	copyValue := append([]byte(nil), value...)
	return copyValue, true, nil
}

func (c *memoryListCache) Set(ctx context.Context, chatID int64, body []byte, ttl time.Duration) error {
	c.setCalls++

	copyBody := append([]byte(nil), body...)
	c.values[chatID] = copyBody

	return nil
}

func (c *memoryListCache) Delete(ctx context.Context, chatID int64) error {
	c.deleteCalls++

	delete(c.values, chatID)

	return nil
}

func (r *fakeLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	result := make([]domain.Link, 0)

	for _, link := range r.links {
		for _, linkTag := range link.Tags {
			if linkTag == tag {
				result = append(result, link)
				break
			}
		}
	}

	return result, nil
}
