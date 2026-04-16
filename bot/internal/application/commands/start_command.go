package commands

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
)

type StartCommand struct {
	client clients.ScrapperClient
}

var _ domain.Command = (*StartCommand)(nil)

func NewStartCommand(client clients.ScrapperClient) *StartCommand {
	return &StartCommand{client: client}
}

func (c *StartCommand) Name() string {
	return "/start"
}

func (c *StartCommand) Execute(ctx context.Context, chatID int64, _ string) (string, error) {
	err := c.client.RegisterChat(context.Background(), chatID)
	if err != nil {
		return "Не удалось зарегистрировать пользователя", err
	}
	return "Добро пожаловать!\n\n" +
		"/help - посмотреть список команд и примеры использования", nil
}
