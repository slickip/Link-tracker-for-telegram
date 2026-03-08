package commands

type ListCommand struct{}

func NewListCommand() *ListCommand {
	return &ListCommand{}
}

func (c *ListCommand) Name() string {
	return "/list"
}

func (c *ListCommand) Execute(chatID int64) (string, error) {
	return "Your tracked links will appear here", nil
}
