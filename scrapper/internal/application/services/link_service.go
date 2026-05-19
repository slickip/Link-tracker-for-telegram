package services

import (
	"context"
	"encoding/json"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	appcache "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
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

	s.invalidateListCache(ctx, chatID)

	return nil
}

func (s *LinkService) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, pkg.ErrChatNotFound
	}

	if s.cacheEnabled() {
		if cachedBody, ok, err := s.listCache.Get(ctx, chatID); err == nil && ok {
			var cachedLinks []domain.Link
			if err := json.Unmarshal(cachedBody, &cachedLinks); err == nil {
				return cachedLinks, nil
			}

			s.invalidateListCache(ctx, chatID)
		}
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
