package repositories

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ORMChatLinkRepository struct {
	db *gorm.DB
}

func NewORMChatLinkRepository(db *gorm.DB) repo.LinkRepository {
	return &ORMChatLinkRepository{db: db}
}

func (r *ORMChatLinkRepository) Add(ctx context.Context, chatID int64, link domain.Link) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		linkModel := models.LinkModel{
			URL:         link.URL,
			LastUpdated: link.LastUpdatedAt,
		}

		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "url"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_updated"}),
		}).Create(&linkModel).Error
		if err != nil {
			return err
		}

		var persistedLink models.LinkModel
		err = tx.Where("url = ?", link.URL).First(&persistedLink).Error
		if err != nil {
			return err
		}

		subscription := models.SubscriptionModel{
			ChatID: chatID,
			LinkID: persistedLink.ID,
		}

		err = tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&subscription).Error
		if err != nil {
			return err
		}

		for _, tagName := range link.Tags {
			tag := models.TagModel{Name: tagName}

			err = tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&tag).Error
			if err != nil {
				return err
			}

			var persistedTag models.TagModel
			err = tx.Where("name = ?", tagName).First(&persistedTag).Error
			if err != nil {
				return err
			}

			linkTag := models.LinkTagModel{
				LinkID: persistedLink.ID,
				TagID:  persistedTag.ID,
			}

			err = tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&linkTag).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ORMChatLinkRepository) Remove(ctx context.Context, chatID int64, url string) error {
	subQuery := r.db.WithContext(ctx).
		Model(&models.LinkModel{}).
		Select("id").
		Where("url = ?", url)

	return r.db.WithContext(ctx).
		Where("chat_id = ? AND link_id = (?)", chatID, subQuery).
		Delete(&models.SubscriptionModel{}).Error
}

func (r *ORMChatLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	type row struct {
		URL         string
		LastUpdated interface{}
	}

	var models []models.LinkModel
	err := r.db.WithContext(ctx).
		Table("links l").
		Select("l.id, l.url, l.last_updated").
		Joins("JOIN subscriptions s ON l.id = s.link_id").
		Where("s.chat_id = ?", chatID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(models))
	for _, m := range models {
		result = append(result, domain.Link{
			URL:           m.URL,
			LastUpdatedAt: m.LastUpdated,
		})
	}

	return result, nil
}

func (r *ORMChatLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	var models []models.LinkModel

	err := r.db.WithContext(ctx).
		Table("links l").
		Select("DISTINCT l.id, l.url, l.last_updated").
		Joins("JOIN subscriptions s ON l.id = s.link_id").
		Joins("JOIN link_tags lt ON l.id = lt.link_id").
		Joins("JOIN tags t ON t.id = lt.tag_id").
		Where("s.chat_id = ? AND t.name = ?", chatID, tag).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(models))
	for _, m := range models {
		result = append(result, domain.Link{
			URL:           m.URL,
			LastUpdatedAt: m.LastUpdated,
		})
	}

	return result, nil
}
