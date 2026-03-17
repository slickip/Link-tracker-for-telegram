package services

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/repositories"

type ChatService struct {
	repo repositories.ChatRepository
}

func NewChatService(repo repositories.ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) RegisterChat(chatID int64) error {
	return s.repo.Add(chatID)
}

func (s *ChatService) DeleteChat(chatID int64) error {
	err := s.repo.Remove(chatID)
	if err != nil {
		return err
	}
	return nil
}

func (s *ChatService) Exists(chatID int64) bool {
	return s.repo.Exists(chatID)
}
