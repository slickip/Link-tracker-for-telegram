package commands

import (
	"context"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
)

const tagArgIndex = 1

type ListCommand struct {
	client clients.ScrapperClient
}

func NewListCommand(client clients.ScrapperClient) *ListCommand {
	return &ListCommand{client: client}
}

func (c *ListCommand) Name() string {
	return "/list"
}

func extractTag(text string) string {
	args := strings.Fields(text)

	if len(args) < 2 {
		return ""
	}

	return args[1]
}

func (c *ListCommand) Execute(chatID int64, text string) (string, error) {
	tag := extractTag(text)

	ctx := context.Background()

	if err := c.client.RegisterChat(ctx, chatID); err != nil {
		return "Не удалось зарегистрировать чат. Попробуй позже", err
	}

	links, err := c.client.ListLinks(ctx, chatID)
	if err != nil {
		return "Ошибка получения ссылок", err
	}

	if len(links) == 0 {
		return "Список отслеживаемых ссылок пуст", nil
	}

	if tag != "" {
		filtered := make([]domain.Link, 0, len(links))

		for _, link := range links {
			for _, t := range link.Tags {
				if t == tag {
					filtered = append(filtered, link)
					break
				}
			}
		}

		if len(filtered) == 0 {
			return "Ссылки с указанным тегом не найдены", nil
		}

		links = filtered
	}

	var builder strings.Builder
	builder.WriteString("Отслеживаемые ссылки:\n")

	for _, link := range links {
		builder.WriteString("- " + link.URL)

		if len(link.Tags) > 0 {
			builder.WriteString(" (" + strings.Join(link.Tags, ", ") + ")")
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}
