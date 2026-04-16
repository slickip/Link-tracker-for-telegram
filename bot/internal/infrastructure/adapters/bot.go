package adapters

import (
	"context"
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
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

func (b *Bot) SetCommands() error {
	commands := []tg.BotCommand{
		{Command: "start", Description: "Начать работу с ботом"},
		{Command: "help", Description: "Показать список команд"},
		{Command: "track", Description: "Начать отслеживание ссылки"},
		{Command: "untrack", Description: "Перестать отслеживать ссылку"},
		{Command: "list", Description: "Показать список ссылок"},
		{Command: "cancel", Description: "Отменить текущую операцию"},
	}

	cfg := tg.NewSetMyCommands(commands...)

	_, err := b.api.Request(cfg)
	return err
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

func (b *Bot) Run(
	dispatcher *dispatch.Dispatcher,
	log domain.Logger,
) {
	updates := b.ListenUpdates()
	for update := range updates {
		if update.Message == nil {
			continue
		}

		var (
			msg    = update.Message
			chatID = msg.Chat.ID
			text   = msg.Text
		)

		log.Info("received message", "chat_id", chatID, "text", text)

		ctx := context.Background()

		response, err := dispatcher.Dispatch(ctx, chatID, text)
		if err != nil {
			log.Warn("command execution error", "error", err)
			continue
		}

		if response != "" {
			if err := b.SendMessage(chatID, response); err != nil {
				log.Warn("failed to send message", "error", err)
			}
		}
	}
}
