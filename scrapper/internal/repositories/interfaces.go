package repositories

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type ChatRepository interface {
	Add(ctx context.Context, chatID int64) error
	Remove(ctx context.Context, chatID int64) error
	Exists(ctx context.Context, chatID int64) (bool, error)
}

type LinkRepository interface {
	Add(ctx context.Context, chatID int64, link domain.Link) error
	Remove(ctx context.Context, chatID int64, url string) error
	RemoveByTag(ctx context.Context, chatID int64, tag string) (int64, error)
	List(ctx context.Context, chatID int64) ([]domain.Link, error)
	ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error)
}

type TagRepository interface {
	Create(ctx context.Context, chatID int64, name string) error
	List(ctx context.Context, chatID int64) ([]domain.Tag, error)
	Rename(ctx context.Context, chatID int64, oldName, newName string) error
	Delete(ctx context.Context, chatID int64, name string) error
}

type TrackingRepository interface {
	FindSubscribers(ctx context.Context, linkID int64) ([]int64, error)
	GetTrackedLinksBatch(ctx context.Context, limit, offset int) ([]domain.Link, error)
	UpdateLastUpdated(ctx context.Context, linkID int64, t time.Time) error
}

type OutboxRepository interface {
	SaveLinkUpdateOutbox(ctx context.Context, linkID int64, newUpdatedAt time.Time, messages []domain.OutboxMessage) error
	SaveOutboxMessages(ctx context.Context, messages []domain.OutboxMessage) error
	GetPendingMessages(ctx context.Context, limit int) ([]domain.OutboxMessage, error)
	MarkAsSent(ctx context.Context, id int64) error
	MarkPublishFailed(ctx context.Context, id int64, err error) error
}
