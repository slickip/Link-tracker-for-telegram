package repositories

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type LinkRepository interface {
	Add(ctx context.Context, chatID int64, link domain.Link) error
	Remove(ctx context.Context, chatID int64, url string) error
	List(ctx context.Context, chatID int64) ([]domain.Link, error)
	ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error)
}
