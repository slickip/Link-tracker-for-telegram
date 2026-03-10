package commands

import (
	"context"
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/clients"
)

type ListCommand struct {
	client clients.ScrapperClient
}

func NewListCommand(client clients.ScrapperClient) *ListCommand {
	return &ListCommand{client: client}
}

func (c *ListCommand) Name() string {
	return "/list"
}

func (c *ListCommand) Execute(chatID int64, text string) (string, error) {

	parts := strings.Fields(text)

	// На случай если пользователь не вызывал /start.
	if err := c.client.RegisterChat(context.Background(), chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	links, err := c.client.ListLinks(
		context.Background(),
		chatID,
	)

	if err != nil {
		return "Ошибка получения ссылок", err
	}

	if len(links) == 0 {
		return "Список отслеживаемых ссылок пуст", nil
	}

	// Optional tag filter: /list <tag>
	if len(parts) > 1 {
		tag := parts[1]

		filtered := links[:0]

		for _, link := range links {
			for _, t := range link.Tags {
				if t == tag {
					filtered = append(filtered, link)
					break
				}
			}
		}

		links = filtered

		if len(links) == 0 {
			return "Ссылки с указанным тегом не найдены", nil
		}
	}

	var builder strings.Builder

	builder.WriteString("Отслеживаемые ссылки:\n")

	for _, link := range links {
		builder.WriteString(fmt.Sprintf("- %s\n", link.URL))
	}

	return builder.String(), nil
}

