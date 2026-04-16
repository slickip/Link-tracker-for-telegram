package commands

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
)

type TrackCommand struct {
	service *services.TrackService
}

func NewTrackCommand(service *services.TrackService) *TrackCommand {
	return &TrackCommand{service: service}
}

func (c *TrackCommand) Name() string {
	return "/track"
}

func (c *TrackCommand) Execute(ctx context.Context, chatID int64, _ string) (string, error) {
	return c.service.Start(ctx, chatID), nil
}
