package services

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

const (
	defaultOutboxPublishInterval = 5 * time.Second
	defaultOutboxBatchSize       = 100
)

type OutboxPublisher struct {
	outboxRepo repositories.OutboxRepository
	producer   kafka.RawMessageProducer
	log        *logger.Slog

	interval  time.Duration
	batchSize int
}

func NewOutboxPublisher(
	outboxRepo repositories.OutboxRepository,
	producer kafka.RawMessageProducer,
	log *logger.Slog,
	interval time.Duration,
	batchSize int,
) *OutboxPublisher {
	if interval <= 0 {
		interval = defaultOutboxPublishInterval
	}
	if batchSize <= 0 {
		batchSize = defaultOutboxBatchSize
	}

	return &OutboxPublisher{
		outboxRepo: outboxRepo,
		producer:   producer,
		log:        log,
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (p *OutboxPublisher) Start(ctx context.Context) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		p.log.Error("outbox publisher init error", "error", err)
		return
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(p.interval),
		gocron.NewTask(func() {
			p.PublishPending(ctx)
		}),
	)
	if err != nil {
		p.log.Error("outbox publisher job error", "error", err)
		return
	}

	p.log.Info("outbox publisher started", "interval", p.interval.String())

	scheduler.Start()
}

func (p *OutboxPublisher) PublishPending(ctx context.Context) {
	messages, err := p.outboxRepo.GetPendingMessages(ctx, p.batchSize)
	if err != nil {
		p.log.Error("failed to get pending outbox messages", "error", err)
		return
	}

	for _, message := range messages {
		err := p.producer.ProduceRaw(
			ctx,
			message.Topic,
			message.MessageKey,
			message.Payload,
		)
		if err != nil {
			p.log.Warn(
				"failed to publish outbox message",
				"id", message.ID,
				"topic", message.Topic,
				"error", err,
			)

			if markErr := p.outboxRepo.MarkPublishFailed(ctx, message.ID, err); markErr != nil {
				p.log.Warn(
					"failed to mark outbox message publish failure",
					"id", message.ID,
					"error", markErr,
				)
			}

			continue
		}

		if err := p.outboxRepo.MarkAsSent(ctx, message.ID); err != nil {
			p.log.Warn(
				"failed to mark outbox message as sent",
				"id", message.ID,
				"error", err,
			)
		}
	}
}
