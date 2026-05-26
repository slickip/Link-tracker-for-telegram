package services

import (
	"context"

	appcache "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/cache"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

type ChatService struct {
	repo      repositories.ChatRepository
	listCache appcache.ListCache
}

func NewChatService(repo repositories.ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func NewChatServiceWithCache(repo repositories.ChatRepository, listCache appcache.ListCache) *ChatService {
	return &ChatService{
		repo:      repo,
		listCache: listCache,
	}
}

func (s *ChatService) RegisterChat(ctx context.Context, chatID int64) error {
	return s.repo.Add(ctx, chatID)
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	err := s.repo.Remove(ctx, chatID)
	if err != nil {
		return err
	}

	if s.listCache != nil {
		_ = s.listCache.Delete(ctx, chatID)
	}

	return nil
}

func (s *ChatService) Exists(ctx context.Context, chatID int64) (bool, error) {
	return s.repo.Exists(ctx, chatID)
}
