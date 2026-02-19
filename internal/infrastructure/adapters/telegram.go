package adapters

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api *tg.BotAPI
}

func NewBot(token string) (*Bot, error) {
	api, err := tg.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return &Bot{
		api: api,
	}, nil
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tg.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) ListenUpdates() tg.UpdatesChannel {
	u := tg.NewUpdate(0)
	u.Timeout = 60

	return b.api.GetUpdatesChan(u)
}
