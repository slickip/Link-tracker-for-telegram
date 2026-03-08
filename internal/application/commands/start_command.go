package commands

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
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

func (c *StartCommand) Execute(chatID int64) (string, error) {
	err := c.client.RegisterChat(context.Background(), chatID)
	if err != nil {
		return "Не удалось зарегистрировать пользователя", err
	}
	return "Доброе утро!!!! Напишите /help, чтобы увидеть список комманд", nil
}
