package commands

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type StartCommand struct{}

var _ domain.Command = (*StartCommand)(nil)

func NewStartCommand() domain.Command {
	return &StartCommand{}
}

func (c *StartCommand) Name() string {
	return "/start"
}

func (c *StartCommand) Execute(chatID int64) (string, error) {
	return "Доброе утро!!!! Напишите /help, чтобы увидеть список комманд", nil
}
