package services

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
)

type LinkService struct {
	linkRepo repositories.LinkRepository
	chatRepo repositories.ChatRepository
}

func NewLinkService(linkRepo repositories.LinkRepository, chatRepo repositories.ChatRepository) *LinkService {
	return &LinkService{
		linkRepo: linkRepo,
		chatRepo: chatRepo,
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

	return s.linkRepo.Add(ctx, chatID, link)
}

func (s *LinkService) RemoveLink(ctx context.Context, chatID int64, url string) error {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return err
	}
	if !exists {
		return pkg.ErrChatNotFound
	}

	return s.linkRepo.Remove(ctx, chatID, url)
}

func (s *LinkService) ListLinks(ctx context.Context, chatID int64) ([]domain.Link, error) {
	exists, err := s.chatRepo.Exists(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, pkg.ErrChatNotFound
	}

	return s.linkRepo.List(ctx, chatID)
}
