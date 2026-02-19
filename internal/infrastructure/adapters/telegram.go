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
