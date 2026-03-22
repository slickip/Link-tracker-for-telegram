package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/repositories"
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
