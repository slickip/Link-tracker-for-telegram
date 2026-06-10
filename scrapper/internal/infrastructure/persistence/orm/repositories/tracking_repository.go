package repositories

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"

	"github.com/slickip/link-tracker/scrapper/internal/domain"
	"github.com/slickip/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	repo "github.com/slickip/link-tracker/scrapper/internal/repositories"
)

type GormTrackingRepository struct {
	db *gorm.DB
}

func NewGormTrackingRepository(db *gorm.DB) repo.TrackingRepository {
	return &GormTrackingRepository{db: db}
}

func (r *GormTrackingRepository) FindSubscribers(ctx context.Context, linkID int64) ([]int64, error) {
	var result []int64

	err := r.db.WithContext(ctx).
		Table("subscriptions").
		Select("chat_id").
		Where("link_id = ?", linkID).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *GormTrackingRepository) GetTrackedLinksBatch(ctx context.Context, limit, offset int) ([]domain.Link, error) {
	type linkRow struct {
		ID          int64
		URL         string
		LastUpdated sql.NullTime
	}

	var rows []linkRow

	err := r.db.WithContext(ctx).
		Table("links l").
		Select("DISTINCT l.id, l.url, l.last_updated").
		Joins("JOIN subscriptions s ON s.link_id = l.id").
		Order("l.id").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	links := make([]domain.Link, 0, len(rows))

	for _, row := range rows {
		link := domain.Link{
			ID:  row.ID,
			URL: row.URL,
		}

		if row.LastUpdated.Valid {
			link.LastUpdatedAt = row.LastUpdated.Time
		}

		links = append(links, link)
	}

	return links, nil
}

func (r *GormTrackingRepository) UpdateLastUpdated(ctx context.Context, linkID int64, t time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.LinkModel{}).
		Where("id = ?", linkID).
		Update("last_updated", t).Error
}
