package command

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"

type HelpCommand struct{}

var _ domain.Command = (*HelpCommand)(nil)

func (c *HelpCommand) Name() string {
	return "/help"
}

func (c *HelpCommand) Execute(chatID int64) (string, error) {
	return "Пока ничего :(", nil
}
