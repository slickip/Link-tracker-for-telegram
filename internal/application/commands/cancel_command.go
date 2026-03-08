package commands

type CancelCommand struct{}

func NewCancelCommand() *CancelCommand {
	return &CancelCommand{}
}

func (c *CancelCommand) Name() string {
	return "/cancel"
}

func (c *CancelCommand) Execute(chatID int64) (string, error) {
	return "Operation cancelled", nil
}
