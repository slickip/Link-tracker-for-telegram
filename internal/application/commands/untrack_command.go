package commands

type UntrackCommand struct{}

func NewUntrackCommand() *UntrackCommand {
	return &UntrackCommand{}
}

func (c *UntrackCommand) Name() string {
	return "/untrack"
}

func (c *UntrackCommand) Execute(chatID int64) (string, error) {
	return "Send the link you want to stop tracking", nil
}
