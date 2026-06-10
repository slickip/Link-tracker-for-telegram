package services

import (
	"context"
	"errors"

	"github.com/slickip/link-tracker/pkg/api"
)

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

	text := update.SubscriberNotificationBody()

	for _, chatID := range update.TgChatIDs {
		if err := s.bot.SendMessage(chatID, text); err != nil {
			return err
		}
	}

	return nil
}
