package commands

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/services"

type TrackCommand struct {
	service *services.TrackService
}

func NewTrackCommand(service *services.TrackService) *TrackCommand {
	return &TrackCommand{service: service}
}

func (c *TrackCommand) Name() string {
	return "/track"
}

func (c *TrackCommand) Execute(chatID int64) (string, error) {
	return c.service.Start(chatID), nil
}
