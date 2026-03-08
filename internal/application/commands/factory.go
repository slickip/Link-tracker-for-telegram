package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
)

func NewDefaultDispatcher(client clients.ScrapperClient, trackService *services.TrackService) *dispatch.Dispatcher {
	return dispatch.NewDispatcher(
		[]domain.Command{
			NewStartCommand(client),
			NewHelpCommand(),
			NewTrackCommand(),
			NewUntrackCommand(),
			NewListCommand(),
			NewCancelCommand(),
			NewUnknownCommand(),
		},
	)
}
