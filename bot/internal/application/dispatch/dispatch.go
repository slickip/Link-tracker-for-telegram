package dispatch

import (
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/repositories"
)

type Dispatcher struct {
	commands     map[string]domain.Command
	trackService *services.TrackService
	repo         repositories.TrackSessionRepository
}

func NewDispatcher(
	commands []domain.Command,
	trackService *services.TrackService,
	repo repositories.TrackSessionRepository,
) *Dispatcher {

	cmdMap := make(map[string]domain.Command)

	for _, cmd := range commands {
		cmdMap[cmd.Name()] = cmd
	}

	return &Dispatcher{
		commands:     cmdMap,
		trackService: trackService,
		repo:         repo,
	}
}

func (d *Dispatcher) Dispatch(chatID int64, text string) (string, error) {
	session, active := d.repo.Get(chatID)

	if strings.HasPrefix(text, "/") {
		if active && !strings.HasPrefix(text, "/cancel") {
			return "Сначала завершите текущую операцию или используйте /cancel", nil
		}

		if strings.HasPrefix(text, "/cancel") {
			d.repo.Reset(chatID)
			return "Операция отменена", nil
		}

		cmdName := strings.Split(text, " ")[0]

		cmd, ok := d.commands[cmdName]
		if !ok {
			cmd = d.commands["unknown"]
		}

		return cmd.Execute(chatID, text)
	}

	if strings.HasPrefix(text, "/") {
		if active {
			d.repo.Reset(chatID)
		}

		cmdName := strings.Split(text, " ")[0]

		cmd, ok := d.commands[cmdName]
		if !ok {
			cmd = d.commands["unknown"]
		}

		return cmd.Execute(chatID, text)
	}

	if active {
		switch session.State {

		case domain.StateWaitingForURL:
			return d.trackService.HandleURL(chatID, text), nil

		case domain.StateWaitingForTags:
			return d.trackService.HandleTags(chatID, text)
		}
	}

	return "Неизвестная команда. Используй /help", nil
}
