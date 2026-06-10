package dispatch

import (
	"context"
	"strings"
	"time"

	"github.com/slickip/link-tracker/bot/internal/application/services"
	"github.com/slickip/link-tracker/bot/internal/domain"
	"github.com/slickip/link-tracker/bot/internal/domain/repositories"
	botmetrics "github.com/slickip/link-tracker/bot/internal/infrastructure/metrics"
)

const (
	defaultCommandName = "message"
	unknownCommandKey  = "unknown"

	commandDurationScope = "scrapper_sync_api"
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

func (d *Dispatcher) Dispatch(ctx context.Context, chatID int64, text string) (string, error) {
	commandName := resolveCommandName(text)
	started := time.Now()

	defer func() {
		botmetrics.CommandDurationMs.
			WithLabelValues(commandDurationScope, commandName).
			Observe(float64(time.Since(started).Milliseconds()))
	}()

	botmetrics.CommandRequestsTotal.WithLabelValues(commandName).Inc()

	session, active, err := d.repo.Get(ctx, chatID)
	if err != nil {
		return "Не удалось получить состояние. Попробуй позже", err
	}

	if strings.HasPrefix(text, "/") {
		if active && !strings.HasPrefix(text, "/cancel") {
			return "Сначала завершите текущую операцию или используйте /cancel", nil
		}

		if strings.HasPrefix(text, "/cancel") {
			if err := d.repo.Reset(ctx, chatID); err != nil {
				return "Не удалось отменить операцию. Попробуй позже", err
			}

			return "Операция отменена", nil
		}

		cmdName := strings.Split(text, " ")[0]

		cmd, ok := d.commands[cmdName]
		if !ok {
			cmd = d.commands[unknownCommandKey]
		}

		return cmd.Execute(ctx, chatID, text)
	}

	if active {
		switch session.State {
		case domain.StateWaitingForURL:
			return d.trackService.HandleURL(ctx, chatID, text), nil

		case domain.StateWaitingForTags:
			return d.trackService.HandleTags(ctx, chatID, text)
		}
	}

	return "Неизвестная команда. Используй /help", nil
}

func resolveCommandName(text string) string {
	if !strings.HasPrefix(text, "/") {
		return defaultCommandName
	}

	return strings.Split(text, " ")[0]
}
