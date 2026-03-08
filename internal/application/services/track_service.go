package services

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/repositories"
)

type TrackService struct {
	client clients.ScrapperClient
	repo   repositories.TrackSessionRepository
}

func NewTrackService(
	client clients.ScrapperClient,
	repo repositories.TrackSessionRepository,
) *TrackService {
	return &TrackService{
		client: client,
		repo:   repo,
	}
}

func (s *TrackService) Start(chatID int64) string {

	s.repo.Set(chatID, domain.TrackSession{
		State: domain.StateWaitingForURL,
	})

	return "Отправь мне ссылку, которую хочешь отслеживать"
}

func (s *TrackService) HandleURL(chatID int64, url string) string {

	session, ok := s.repo.Get(chatID)
	if !ok {
		return ""
	}

	session.URL = url
	session.State = domain.StateWaitingForTags

	s.repo.Set(chatID, session)

	return "Отправь мне теги, разделенные запятой, или напиши \"-\" если хочешь пропустить этот этап"
}

func (s *TrackService) HandleTags(chatID int64, text string) (string, error) {

	session, ok := s.repo.Get(chatID)
	if !ok {
		return "", nil
	}

	var tags []string

	if text != "-" {
		tags = strings.Split(text, ",")
	}

	err := s.client.AddLink(
		context.Background(),
		chatID,
		session.URL,
		tags,
	)

	if err != nil {
		return "Не получилось добавить ссылку", err
	}

	s.repo.Reset(chatID)

	return "Ссылка добавлена успешно!", nil
}

func (s *TrackService) Cancel(chatID int64) string {

	s.repo.Reset(chatID)

	return "Отслеживание отменено"
}
