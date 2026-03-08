package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/repositories"
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
