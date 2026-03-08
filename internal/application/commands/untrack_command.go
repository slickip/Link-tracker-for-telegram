package commands

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
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

func (c *UntrackCommand) Execute(chatID int64) (string, error) {

	text := ""

	parts := strings.Split(text, " ")

	if len(parts) < 2 {
		return "Использование: /untrack <url>", nil
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
