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
	FindSubscribers(ctx context.Context, url string) ([]int64, error)
	GetAllTrackedLinks(ctx context.Context) ([]domain.Link, error)
	UpdateLastUpdated(ctx context.Context, url string, t time.Time) error
}
