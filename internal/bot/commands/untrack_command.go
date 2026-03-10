package commands

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/clients"
)

type UntrackCommand struct {
	client clients.ScrapperClient
}

func NewUntrackCommand(client clients.ScrapperClient) *UntrackCommand {
	return &UntrackCommand{client: client}
}

func (c *UntrackCommand) Name() string {
	return "/untrack"
}

func (c *UntrackCommand) Execute(chatID int64, text string) (string, error) {

	parts := strings.Fields(text)

	if len(parts) < 2 {
		return "Использование: /untrack <url>", nil
	}

	// На случай если пользователь не вызывал /start.
	if err := c.client.RegisterChat(context.Background(), chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	url := parts[1]

	err := c.client.RemoveLink(
		context.Background(),
		chatID,
		url,
	)

	if err != nil {
		return "Не получилось удалить ссылку", err
	}

	return "Ссылка удалена", nil
}

