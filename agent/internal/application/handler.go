package application

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type Handler struct {
	processor       *Processor
	groupingService *GroupingService
	log             *logger.Slog
}

func NewHandler(
	processor *Processor,
	groupingService *GroupingService,
	log *logger.Slog,
) *Handler {
	return &Handler{
		processor:       processor,
		groupingService: groupingService,
		log:             log,
	}
}

func (h *Handler) HandleLinkUpdate(
	ctx context.Context,
	update api.LinkUpdate,
) error {
	processedUpdate, ok, err := h.processor.Process(ctx, update)
	if err != nil {
		return err
	}

	if !ok {
		h.log.Info(
			"link update filtered",
			"id", update.ID,
			"username", update.Username,
		)
		return nil
	}

	h.groupingService.Add(ctx, processedUpdate)

	h.log.Info(
		"link update accepted for grouping",
		"id", processedUpdate.ID,
		"priority", processedUpdate.Priority,
	)

	return nil
}
