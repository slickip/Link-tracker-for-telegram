package commands

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/services"

type TrackCommand struct {
	service *services.TrackService
}

func NewTrackCommand(service *services.TrackService) *TrackCommand {
	return &TrackCommand{service: service}
}

func (c *TrackCommand) Name() string {
	return "/track"
}

func (c *TrackCommand) Execute(chatID int64, _ string) (string, error) {
	return c.service.Start(chatID), nil
}

