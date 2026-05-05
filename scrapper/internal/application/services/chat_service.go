package services

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

type ChatService struct {
	repo repositories.ChatRepository
}

func NewChatService(repo repositories.ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) RegisterChat(ctx context.Context, chatID int64) error {
	return s.repo.Add(ctx, chatID)
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID int64) error {
	err := s.repo.Remove(ctx, chatID)
	if err != nil {
		return err
	}
	return nil
}

func (s *ChatService) Exists(ctx context.Context, chatID int64) (bool, error) {
	return s.repo.Exists(ctx, chatID)
}
