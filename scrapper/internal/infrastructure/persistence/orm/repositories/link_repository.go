package repositories

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
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

			subscriptionTag := models.SubscriptionTagModel{
				ChatID: chatID,
				LinkID: persistedLink.ID,
				TagID:  persistedTag.ID,
			}

			err = tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&subscriptionTag).Error
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

	result := r.db.WithContext(ctx).
		Where("chat_id = ? AND link_id = (?)", chatID, subQuery).
		Delete(&models.SubscriptionModel{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return pkg.ErrLinkNotFound
	}

	return nil
}

func (r *ORMChatLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	var linkModels []models.LinkModel

	err := r.db.WithContext(ctx).
		Table("links l").
		Select("l.id, l.url, l.last_updated").
		Joins("JOIN subscriptions s ON l.id = s.link_id").
		Where("s.chat_id = ?", chatID).
		Find(&linkModels).Error
	if err != nil {
		return nil, err
	}

	linkIDs := make([]int64, 0, len(linkModels))
	for _, m := range linkModels {
		linkIDs = append(linkIDs, m.ID)
	}

	tagsByLinkID, err := r.getTagsByLinkIDs(ctx, chatID, linkIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(linkModels))
	for _, m := range linkModels {
		result = append(result, domain.Link{
			URL:           m.URL,
			LastUpdatedAt: m.LastUpdated,
			Tags:          tagsByLinkID[m.ID],
		})
	}

	return result, nil
}

func (r *ORMChatLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	var linkModels []models.LinkModel

	err := r.db.WithContext(ctx).
		Table("links l").
		Select("DISTINCT l.id, l.url, l.last_updated").
		Joins("JOIN subscriptions s ON l.id = s.link_id").
		Joins("JOIN subscription_tags st ON st.link_id = l.id AND st.chat_id = s.chat_id").
		Joins("JOIN tags t ON t.id = st.tag_id").
		Where("s.chat_id = ? AND t.name = ?", chatID, tag).
		Find(&linkModels).Error
	if err != nil {
		return nil, err
	}

	linkIDs := make([]int64, 0, len(linkModels))
	for _, m := range linkModels {
		linkIDs = append(linkIDs, m.ID)
	}

	tagsByLinkID, err := r.getTagsByLinkIDs(ctx, chatID, linkIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(linkModels))
	for _, m := range linkModels {
		result = append(result, domain.Link{
			URL:           m.URL,
			LastUpdatedAt: m.LastUpdated,
			Tags:          tagsByLinkID[m.ID],
		})
	}

	return result, nil
}

func (r *ORMChatLinkRepository) getTagsByLinkIDs(ctx context.Context, chatID int64, linkIDs []int64) (map[int64][]string, error) {
	if len(linkIDs) == 0 {
		return map[int64][]string{}, nil
	}

	type tagRow struct {
		LinkID int64
		Name   string
	}

	var rows []tagRow
	err := r.db.WithContext(ctx).
		Table("subscription_tags st").
		Select("st.link_id, t.name").
		Joins("JOIN tags t ON t.id = st.tag_id").
		Where("st.chat_id = ? AND st.link_id IN ?", chatID, linkIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]string)
	for _, row := range rows {
		result[row.LinkID] = append(result[row.LinkID], row.Name)
	}

	return result, nil
}

func (r *ORMChatLinkRepository) RemoveByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	subQuery := r.db.WithContext(ctx).
		Table("subscription_tags st").
		Select("st.link_id").
		Joins("JOIN tags t ON t.id = st.tag_id").
		Where("st.chat_id = ? AND t.name = ?", chatID, tag)

	res := r.db.WithContext(ctx).
		Where("chat_id = ? AND link_id IN (?)", chatID, subQuery).
		Delete(&models.SubscriptionModel{})

	if res.Error != nil {
		return 0, res.Error
	}

	return res.RowsAffected, nil
}
