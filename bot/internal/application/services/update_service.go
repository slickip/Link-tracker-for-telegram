package services

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

const defaultLinkUpdateMessagePrefix = "Обнаружено обновление по ссылке: "

var ErrInvalidLinkUpdate = errors.New("invalid link update")

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type UpdateService struct {
	bot MessageSender
}

func NewUpdateService(bot MessageSender) *UpdateService {
	return &UpdateService{
		bot: bot,
	}
}

func (s *UpdateService) HandleLinkUpdate(_ context.Context, update api.LinkUpdate) error {
	if update.URL == "" || len(update.TgChatIDs) == 0 {
		return ErrInvalidLinkUpdate
	}

	text := update.Description
	if text == "" {
		text = defaultLinkUpdateMessagePrefix + update.URL
	}

	for _, chatID := range update.TgChatIDs {
		if err := s.bot.SendMessage(chatID, text); err != nil {
			return err
		}
	}

	return nil
}
