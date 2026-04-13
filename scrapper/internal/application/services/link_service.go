package services

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/repositories"
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

func (s *LinkService) AddLink(chatID int64, link domain.Link) error {
	if !s.chatRepo.Exists(chatID) {
		return pkg.ErrChatNotFound
	}

	return s.linkRepo.Add(chatID, link)
}

func (s *LinkService) RemoveLink(chatID int64, url string) error {
	if !s.chatRepo.Exists(chatID) {
		return pkg.ErrChatNotFound
	}

	return s.linkRepo.Remove(chatID, url)
}

func (s *LinkService) ListLinks(chatID int64) ([]domain.Link, error) {
	if !s.chatRepo.Exists(chatID) {
		return nil, pkg.ErrChatNotFound
	}

	return s.linkRepo.List(chatID)
}
