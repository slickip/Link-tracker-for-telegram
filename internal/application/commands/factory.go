package commands

import (
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application/dispatch"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

func NewDefaultDispatcher() *dispatch.Dispatcher {
	return dispatch.NewDispatcher(
		[]domain.Command{
			NewStartCommand(),
			NewHelpCommand(),
			NewTrackCommand(),
			NewUntrackCommand(),
			NewListCommand(),
			NewCancelCommand(),
			NewUnknownCommand(),
		},
	)
}
