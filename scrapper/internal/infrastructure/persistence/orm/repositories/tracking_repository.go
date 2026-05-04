package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

type GormTrackingRepository struct {
	db *gorm.DB
}

func NewGormTrackingRepository(db *gorm.DB) repo.TrackingRepository {
	return &GormTrackingRepository{db: db}
}

func (r *GormTrackingRepository) FindSubscribers(ctx context.Context, url string) ([]int64, error) {
	var result []int64

	err := r.db.WithContext(ctx).
		Table("subscriptions s").
		Select("s.chat_id").
		Joins("JOIN links l ON l.id = s.link_id").
		Where("l.url = ?", url).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *GormTrackingRepository) GetAllTrackedLinks(ctx context.Context) ([]domain.Link, error) {
	var links []models.LinkModel

	err := r.db.WithContext(ctx).
		Select("url", "last_updated", "last_checked").
		Find(&links).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(links))
	for _, link := range links {
		result = append(result, domain.Link{
			URL:           link.URL,
			LastUpdatedAt: link.LastUpdated,
		})
	}

	return result, nil
}

func (r *GormTrackingRepository) UpdateLastUpdated(ctx context.Context, url string, t time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.LinkModel{}).
		Where("url = ?", url).
		Update("last_updated", t).Error
}
