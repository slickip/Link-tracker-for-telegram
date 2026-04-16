package commands

import "context"

type CancelCommand struct{}

func NewCancelCommand() *CancelCommand {
	return &CancelCommand{}
}

func (c *CancelCommand) Name() string {
	return "/cancel"
}

func (c *CancelCommand) Execute(ctx context.Context, chatID int64, _ string) (string, error) {
	return "Operation cancelled", nil
}
