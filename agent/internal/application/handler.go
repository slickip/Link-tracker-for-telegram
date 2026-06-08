package application

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type Handler struct {
	processor *Processor
	producer  kafka.LinkUpdateProducer
	log       *logger.Slog
}

func NewHandler(
	processor *Processor,
	producer kafka.LinkUpdateProducer,
	log *logger.Slog,
) *Handler {
	return &Handler{
		processor: processor,
		producer:  producer,
		log:       log,
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

	if err := h.producer.Produce(ctx, processedUpdate); err != nil {
		return err
	}

	h.log.Info(
		"link update processed",
		"id", processedUpdate.ID,
		"priority", processedUpdate.Priority,
	)

	return nil
}
