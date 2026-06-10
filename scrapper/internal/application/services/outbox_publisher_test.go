package services

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/slickip/link-tracker/pkg/logger"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
)

type fakeOutboxRepository struct {
	pendingMessages []domain.OutboxMessage
	sentIDs         []int64
	failedIDs       []int64
}

func (r *fakeOutboxRepository) SaveLinkUpdateOutbox(
	_ context.Context,
	_ int64,
	_ time.Time,
	_ []domain.OutboxMessage,
) error {
	return nil
}

func (r *fakeOutboxRepository) SaveOutboxMessages(
	_ context.Context,
	_ []domain.OutboxMessage,
) error {
	return nil
}

func (r *fakeOutboxRepository) GetPendingMessages(
	_ context.Context,
	_ int,
) ([]domain.OutboxMessage, error) {
	return r.pendingMessages, nil
}

func (r *fakeOutboxRepository) MarkAsSent(_ context.Context, id int64) error {
	r.sentIDs = append(r.sentIDs, id)
	return nil
}

func (r *fakeOutboxRepository) MarkPublishFailed(
	_ context.Context,
	id int64,
	_ error,
) error {
	r.failedIDs = append(r.failedIDs, id)
	return nil
}

type fakeRawProducer struct {
	err error
}

func (p *fakeRawProducer) ProduceRaw(
	_ context.Context,
	_ string,
	_ string,
	_ []byte,
) error {
	return p.err
}

func TestOutboxPublisherMarksMessageAsSentAfterSuccessfulPublish(t *testing.T) {
	repo := &fakeOutboxRepository{
		pendingMessages: []domain.OutboxMessage{
			{
				ID:         1,
				Topic:      "link-updates",
				MessageKey: "10",
				Payload:    []byte(`{"id":10}`),
				Status:     domain.OutboxStatusPending,
			},
		},
	}

	producer := &fakeRawProducer{}

	publisher := NewOutboxPublisher(
		repo,
		producer,
		logger.New(slog.LevelError),
		time.Second,
		10,
	)

	publisher.PublishPending(context.Background())

	if len(repo.sentIDs) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(repo.sentIDs))
	}

	if repo.sentIDs[0] != 1 {
		t.Fatalf("expected sent id 1, got %d", repo.sentIDs[0])
	}

	if len(repo.failedIDs) != 0 {
		t.Fatalf("expected 0 failed messages, got %d", len(repo.failedIDs))
	}
}

func TestOutboxPublisherKeepsMessagePendingAfterPublishError(t *testing.T) {
	repo := &fakeOutboxRepository{
		pendingMessages: []domain.OutboxMessage{
			{
				ID:         2,
				Topic:      "link-updates",
				MessageKey: "20",
				Payload:    []byte(`{"id":20}`),
				Status:     domain.OutboxStatusPending,
			},
		},
	}

	producer := &fakeRawProducer{
		err: errors.New("kafka unavailable"),
	}

	publisher := NewOutboxPublisher(
		repo,
		producer,
		logger.New(slog.LevelError),
		time.Second,
		10,
	)

	publisher.PublishPending(context.Background())

	if len(repo.sentIDs) != 0 {
		t.Fatalf("expected 0 sent messages, got %d", len(repo.sentIDs))
	}

	if len(repo.failedIDs) != 1 {
		t.Fatalf("expected 1 failed message, got %d", len(repo.failedIDs))
	}

	if repo.failedIDs[0] != 2 {
		t.Fatalf("expected failed id 2, got %d", repo.failedIDs[0])
	}
}
