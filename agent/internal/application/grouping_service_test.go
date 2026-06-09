package application

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type fakeProducer struct {
	updates []api.LinkUpdate
}

func (p *fakeProducer) Produce(ctx context.Context, update api.LinkUpdate) error {
	p.updates = append(p.updates, update)
	return nil
}

func TestGroupingService_ShouldGroupSeveralUpdates_TC_2_1(t *testing.T) {
	t.Parallel()

	producer := &fakeProducer{}
	service := NewGroupingService(
		time.Hour,
		producer,
		logger.New(slog.LevelInfo),
	)

	service.Add(context.Background(), api.LinkUpdate{
		ID:          1,
		TgChatIDs:   []int64{111},
		Description: "first update",
		Priority:    string(PriorityLow),
	})

	service.Add(context.Background(), api.LinkUpdate{
		ID:          2,
		TgChatIDs:   []int64{111},
		Description: "second update",
		Priority:    string(PriorityHigh),
	})

	service.FlushAll(context.Background())

	require.Len(t, producer.updates, 1)
	require.Equal(t, []int64{111}, producer.updates[0].TgChatIDs)
	require.Equal(t, string(PriorityHigh), producer.updates[0].Priority)
	require.Contains(t, producer.updates[0].Description, "1. first update")
	require.Contains(t, producer.updates[0].Description, "2. second update")
}

func TestGroupingService_ShouldNotGroupSingleUpdate_TC_2_2(t *testing.T) {
	t.Parallel()

	producer := &fakeProducer{}
	service := NewGroupingService(
		time.Hour,
		producer,
		logger.New(slog.LevelInfo),
	)

	update := api.LinkUpdate{
		ID:          1,
		TgChatIDs:   []int64{111},
		Description: "single update",
		Priority:    string(PriorityMedium),
	}

	service.Add(context.Background(), update)
	service.FlushAll(context.Background())

	require.Len(t, producer.updates, 1)
	require.Equal(t, update.Description, producer.updates[0].Description)
	require.Equal(t, update.Priority, producer.updates[0].Priority)
	require.Equal(t, []int64{111}, producer.updates[0].TgChatIDs)
}
