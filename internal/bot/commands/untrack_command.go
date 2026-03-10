package commands

import (
	"context"
	"fmt"
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
		return "Использование:\n/untrack <url> — удалить по ссылке\n/untrack tag <tag> — удалить все ссылки с тегом", nil
	}

	if err := c.client.RegisterChat(context.Background(), chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	if parts[1] == "tag" {
		if len(parts) < 3 {
			return "Использование: /untrack tag <tag>", nil
		}

		tag := parts[2]

		links, err := c.client.ListLinks(context.Background(), chatID)
		if err != nil {
			return "Не получилось получить список ссылок", err
		}

		var removed int

		for _, link := range links {
			hasTag := false

			for _, t := range link.Tags {
				if t == tag {
					hasTag = true
					break
				}
			}

			if !hasTag {
				continue
			}

			if err := c.client.RemoveLink(context.Background(), chatID, link.URL); err == nil {
				removed++
			}
		}

		if removed == 0 {
			return "Ссылки с указанным тегом не найдены", nil
		}

		return fmt.Sprintf("Удалено ссылок с тегом %s: %d", tag, removed), nil
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

