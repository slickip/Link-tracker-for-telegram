package repositories

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type TagRepository interface {
	Create(ctx context.Context, chatID int64, name string) error
	List(ctx context.Context, chatID int64) ([]domain.Tag, error)
	Rename(ctx context.Context, chatID int64, oldName, newName string) error
	Delete(ctx context.Context, chatID int64, name string) error
}
