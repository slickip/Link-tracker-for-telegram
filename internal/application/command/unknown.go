package command

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type UnknownCommand struct{}

var _ domain.Command = (*UnknownCommand)(nil)

func (c *UnknownCommand) Name() string {
	return "unknown"
}

func (c *UnknownCommand) Execute(chatID int64) (string, error) {
	return "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть доступные команды", nil
}
