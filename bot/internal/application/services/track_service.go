package services

import (
	"context"
	"strings"

	"github.com/slickip/link-tracker/bot/internal/application/clients"
	"github.com/slickip/link-tracker/bot/internal/domain"
	"github.com/slickip/link-tracker/bot/internal/domain/repositories"
	"github.com/slickip/link-tracker/bot/internal/infrastructure/parsers"
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

func (s *TrackService) Start(ctx context.Context, chatID int64) string {
	if err := s.client.RegisterChat(ctx, chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже"
	}

	err := s.repo.Set(ctx, chatID, domain.TrackSession{
		ChatID: chatID,
		State:  domain.StateWaitingForURL,
	})
	if err != nil {
		return "Не удалось сохранить состояние. Попробуй позже"
	}

	return "Отправь мне ссылку, которую хочешь отслеживать"
}

func (s *TrackService) HandleURL(ctx context.Context, chatID int64, url string) string {
	session, ok, err := s.repo.Get(ctx, chatID)
	if err != nil || !ok {
		return ""
	}

	if _, err := parsers.ParseLink(url); err != nil {
		return "Некорректная ссылка. Поддерживаются только ссылки GitHub и StackOverflow"
	}

	session.URL = url
	session.State = domain.StateWaitingForTags

	if err := s.repo.Set(ctx, chatID, session); err != nil {
		return "Не удалось сохранить состояние. Попробуй позже"
	}

	return "Отправь мне теги, разделенные запятой, или напиши \"-\" если хочешь пропустить этот этап"
}

func (s *TrackService) HandleTags(ctx context.Context, chatID int64, text string) (string, error) {
	session, ok, err := s.repo.Get(ctx, chatID)
	if err != nil {
		return "Не удалось получить состояние. Попробуй позже", err
	}
	if !ok {
		return "", nil
	}

	var tags []string

	if text != "-" {
		raw := strings.Split(text, ",")
		for _, t := range raw {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	if err := s.client.RegisterChat(ctx, chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	err = s.client.AddLink(ctx, chatID, session.URL, tags)
	if err != nil {
		if strings.Contains(err.Error(), "AlreadyExists") ||
			strings.Contains(err.Error(), "already tracked") {
			_ = s.repo.Reset(ctx, chatID)
			return "Ссылка уже отслеживается", nil
		}

		return "Не получилось добавить ссылку", err
	}

	if err := s.repo.Reset(ctx, chatID); err != nil {
		return "Ссылка добавлена, но не удалось очистить состояние", err
	}

	return "Ссылка добавлена успешно!", nil
}

func (s *TrackService) Cancel(ctx context.Context, chatID int64) string {
	_ = s.repo.Reset(ctx, chatID)
	return "Отслеживание отменено"
}
