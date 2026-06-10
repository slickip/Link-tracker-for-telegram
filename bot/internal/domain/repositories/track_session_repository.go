package repositories

import (
	"context"

	"github.com/slickip/link-tracker/bot/internal/domain"
)

type TrackSessionRepository interface {
	Get(ctx context.Context, chatID int64) (domain.TrackSession, bool, error)
	Set(ctx context.Context, chatID int64, session domain.TrackSession) error
	Reset(ctx context.Context, chatID int64) error
}
