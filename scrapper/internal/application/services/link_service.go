package services

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/slickip/link-tracker/pkg"
	appcache "github.com/slickip/link-tracker/scrapper/internal/application/cache"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
	"github.com/slickip/link-tracker/scrapper/internal/repositories"

	scrappermetrics "github.com/slickip/link-tracker/scrapper/internal/infrastructure/metrics"
)

type LinkService struct {
	linkRepo     repositories.LinkRepository
	chatRepo     repositories.ChatRepository
	listCache    appcache.ListCache
	listCacheTTL time.Duration
}

func NewLinkService(linkRepo repositories.LinkRepository, chatRepo repositories.ChatRepository) *LinkService {
	return &LinkService{
		linkRepo: linkRepo,
		chatRepo: chatRepo,
	}
}

func NewLinkServiceWithCache(
	linkRepo repositories.LinkRepository,
	chatRepo repositories.ChatRepository,
	listCache appcache.ListCache,
	listCacheTTL time.Duration,
) *LinkService {
	return &LinkService{
		linkRepo:     linkRepo,
		chatRepo:     chatRepo,
		listCache:    listCache,
		listCacheTTL: listCacheTTL,
	}
}

func (s *LinkService) AddLink(ctx context.Context, chatID int64, link domain.Link) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	if err := s.linkRepo.Add(ctx, chatID, link); err != nil {
		return err
	}

	scrappermetrics.LinksOnTrackTotal.
		WithLabelValues(trackedSource(link.URL)).
		Inc()

	s.invalidateListCache(ctx, chatID)

	return nil
}

func (s *LinkService) RemoveLink(ctx context.Context, chatID int64, url string) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	if err := s.linkRepo.Remove(ctx, chatID, url); err != nil {
		return err
	}

	scrappermetrics.LinksOnTrackTotal.
		WithLabelValues(trackedSource(url)).
		Dec()

	s.invalidateListCache(ctx, chatID)

	return nil
}

func (s *LinkService) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	if s.cacheEnabled() {
		cachedBody, ok, err := s.listCache.Get(ctx, chatID)
		if err == nil && ok {
			var cachedLinks []domain.Link
			if err := json.Unmarshal(cachedBody, &cachedLinks); err == nil {
				return cachedLinks, nil
			}

			s.invalidateListCache(ctx, chatID)
		}
	}

	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, pkg.ErrChatNotFound
	}

	links, err := s.linkRepo.List(ctx, chatID)
	if err != nil {
		return nil, err
	}

	if s.cacheEnabled() {
		if body, err := json.Marshal(links); err == nil {
			_ = s.listCache.Set(ctx, chatID, body, s.listCacheTTL)
		}
	}

	return links, nil
}

func (s *LinkService) RemoveLinksByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, pkg.ErrChatNotFound
	}

	removedCount, err := s.linkRepo.RemoveByTag(ctx, chatID, tag)
	if err != nil {
		return 0, err
	}

	if removedCount > 0 {
		s.invalidateListCache(ctx, chatID)
	}

	return removedCount, nil
}

func (s *LinkService) cacheEnabled() bool {
	return s.listCache != nil && s.listCacheTTL > 0
}

func (s *LinkService) invalidateListCache(ctx context.Context, chatID int64) {
	if !s.cacheEnabled() {
		return
	}

	_ = s.listCache.Delete(ctx, chatID)
}

func trackedSource(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		return "unknown"
	}

	host := strings.ToLower(parsedURL.Host)
	host = strings.TrimPrefix(host, "www.")

	switch {
	case strings.Contains(host, "github.com"):
		return "github"
	case strings.Contains(host, "stackoverflow.com"):
		return "stackoverflow"
	default:
		return host
	}
}
