package commands

import (
	"github.com/slickip/link-tracker/bot/internal/application/clients"
	"github.com/slickip/link-tracker/bot/internal/application/dispatch"
	"github.com/slickip/link-tracker/bot/internal/application/services"
	"github.com/slickip/link-tracker/bot/internal/domain"
	"github.com/slickip/link-tracker/bot/internal/domain/repositories"
)

func NewDefaultDispatcher(
	client clients.ScrapperClient,
	trackService *services.TrackService,
	repo repositories.TrackSessionRepository,
) *dispatch.Dispatcher {

	commands := []domain.Command{
		NewStartCommand(client),
		NewHelpCommand(),
		NewTrackCommand(trackService),
		NewUntrackCommand(client),
		NewListCommand(client),
		NewCancelCommand(),
		NewUnknownCommand(),
	}

	return dispatch.NewDispatcher(commands, trackService, repo)
}
