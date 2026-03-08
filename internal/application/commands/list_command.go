package commands

import (
	"context"
	"fmt"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
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

func (c *ListCommand) Execute(chatID int64) (string, error) {

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

	var builder strings.Builder

	builder.WriteString("Отслеживаемые ссылки:\n")

	for _, link := range links {
		builder.WriteString(fmt.Sprintf("- %s\n", link.URL))
	}

	return builder.String(), nil
}
