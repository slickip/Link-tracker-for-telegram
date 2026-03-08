package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/clients"
)

func NewDefaultDispatcher(client clients.ScrapperClient) *dispatch.Dispatcher {
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
