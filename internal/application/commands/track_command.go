package commands

type TrackCommand struct{}

func NewTrackCommand() *TrackCommand {
	return &TrackCommand{}
}

func (c *TrackCommand) Name() string {
	return "/track"
}

func (c *TrackCommand) Execute(chatID int64) (string, error) {
	return "Send the link you want to track", nil
}
