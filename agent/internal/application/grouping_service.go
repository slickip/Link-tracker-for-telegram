package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/kafka"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
)

type GroupingService struct {
	window   time.Duration
	producer kafka.LinkUpdateProducer
	log      *logger.Slog

	mu      sync.Mutex
	buffers map[int64]*chatUpdateBuffer
}

type chatUpdateBuffer struct {
	updates []api.LinkUpdate
	timer   *time.Timer
}

func NewGroupingService(
	window time.Duration,
	producer kafka.LinkUpdateProducer,
	log *logger.Slog,
) *GroupingService {
	return &GroupingService{
		window:   window,
		producer: producer,
		log:      log,
		buffers:  make(map[int64]*chatUpdateBuffer),
	}
}

func (s *GroupingService) Add(ctx context.Context, update api.LinkUpdate) {
	if s.window <= 0 {
		if err := s.producer.Produce(ctx, update); err != nil {
			s.log.Error("failed to publish update", "error", err)
		}
		return
	}

	for _, chatID := range update.TgChatIDs {
		s.addForChat(ctx, chatID, update)
	}
}

func (s *GroupingService) addForChat(
	ctx context.Context,
	chatID int64,
	update api.LinkUpdate,
) {
	s.mu.Lock()
	defer s.mu.Unlock()

	update.TgChatIDs = []int64{chatID}

	buffer, ok := s.buffers[chatID]
	if !ok {
		buffer = &chatUpdateBuffer{
			updates: make([]api.LinkUpdate, 0, 1),
		}
		s.buffers[chatID] = buffer

		buffer.timer = time.AfterFunc(s.window, func() {
			s.flushChat(context.Background(), chatID)
		})
	}

	buffer.updates = append(buffer.updates, update)

	select {
	case <-ctx.Done():
		s.log.Warn("context cancelled while grouping update", "error", ctx.Err())
	default:
	}
}

func (s *GroupingService) FlushAll(ctx context.Context) {
	s.mu.Lock()

	chatIDs := make([]int64, 0, len(s.buffers))
	for chatID := range s.buffers {
		chatIDs = append(chatIDs, chatID)
	}

	s.mu.Unlock()

	for _, chatID := range chatIDs {
		s.flushChat(ctx, chatID)
	}
}

func (s *GroupingService) flushChat(ctx context.Context, chatID int64) {
	s.mu.Lock()

	buffer, ok := s.buffers[chatID]
	if !ok {
		s.mu.Unlock()
		return
	}

	delete(s.buffers, chatID)

	if buffer.timer != nil {
		buffer.timer.Stop()
	}

	updates := append([]api.LinkUpdate(nil), buffer.updates...)

	s.mu.Unlock()

	if len(updates) == 0 {
		return
	}

	result := buildGroupedUpdate(chatID, updates)

	if err := s.producer.Produce(ctx, result); err != nil {
		s.log.Error(
			"failed to publish grouped update",
			"chat_id", chatID,
			"error", err,
		)
	}
}

func buildGroupedUpdate(chatID int64, updates []api.LinkUpdate) api.LinkUpdate {
	if len(updates) == 1 {
		return updates[0]
	}

	result := updates[0]
	result.TgChatIDs = []int64{chatID}
	result.Description = buildNumberedDescription(updates)
	result.Priority = string(maxPriority(updates))

	return result
}

func buildNumberedDescription(updates []api.LinkUpdate) string {
	var builder strings.Builder

	for i, update := range updates {
		if i > 0 {
			builder.WriteString("\n\n")
		}

		_, _ = fmt.Fprintf(&builder, "%d. %s", i+1, update.Description)
	}

	return builder.String()
}

func maxPriority(updates []api.LinkUpdate) Priority {
	result := PriorityLow

	for _, update := range updates {
		priority := Priority(update.Priority)

		if priorityRank(priority) > priorityRank(result) {
			result = priority
		}
	}

	return result
}

func priorityRank(priority Priority) int {
	switch priority {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}
