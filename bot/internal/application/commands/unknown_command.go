package commands

import (
	"context"

	"github.com/slickip/link-tracker/bot/internal/domain"
)

type UnknownCommand struct{}

var _ domain.Command = (*UnknownCommand)(nil)

func NewUnknownCommand() domain.Command {
	return &UnknownCommand{}
}

func (c *UnknownCommand) Name() string {
	return "unknown"
}

func (c *UnknownCommand) Execute(ctx context.Context, chatID int64, _ string) (string, error) {
	return "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть доступные команды", nil
}
