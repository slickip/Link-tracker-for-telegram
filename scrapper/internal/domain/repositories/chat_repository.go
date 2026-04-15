package repositories

import "context"

type ChatRepository interface {
	Add(ctx context.Context, chatID int64) error
	Remove(ctx context.Context, chatID int64) error
	Exists(ctx context.Context, chatID int64) (bool, error)
}
