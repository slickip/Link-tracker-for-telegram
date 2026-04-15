package repositories

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type TrackingRepository interface {
	FindSubscribers(ctx context.Context, url string) ([]int64, error)
	GetAllTrackedLinks(ctx context.Context) ([]domain.Link, error)
	UpdateLastUpdated(ctx context.Context, url string, t time.Time) error
}
