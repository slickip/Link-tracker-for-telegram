package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

func NewDefaultDispatcher(
	client clients.ScrapperClient,
	trackService *services.TrackService,
	repo repositories.TrackSessionRepository,
) *dispatch.Dispatcher {
	return dispatch.NewDispatcher(
		[]domain.Command{
			NewStartCommand(client),
			NewHelpCommand(),
			NewTrackCommand(trackService),
			NewUntrackCommand(client),
			NewListCommand(client),
			NewCancelCommand(),
			NewUnknownCommand(),
		},
		trackService,
		repo,
	)
}

