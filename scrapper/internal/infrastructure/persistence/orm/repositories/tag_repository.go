package repositories

import (
	"context"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	"gorm.io/gorm"
)

type ORMTagRepository struct {
	db *gorm.DB
}

func NewORMTagRepository(db *gorm.DB) repo.TagRepository {
	return &ORMTagRepository{db: db}
}

func (r *ORMTagRepository) Create(ctx context.Context, chatID int64, name string) error {
	tag := models.TagModel{
		ChatID: chatID,
		Name:   name,
	}

	err := r.db.WithContext(ctx).Create(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return pkg.ErrTagExists
		}
		return err
	}

	return nil
}

func (r *ORMTagRepository) List(ctx context.Context, chatID int64) ([]domain.Tag, error) {
	var modelsList []models.TagModel

	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("name ASC").
		Find(&modelsList).Error
	if err != nil {
		return nil, err
	}

	result := make([]domain.Tag, 0, len(modelsList))
	for _, m := range modelsList {
		result = append(result, domain.Tag{
			ID:     m.ID,
			ChatID: m.ChatID,
			Name:   m.Name,
		})
	}

	return result, nil
}

func (r *ORMTagRepository) Rename(ctx context.Context, chatID int64, oldName, newName string) error {
	res := r.db.WithContext(ctx).
		Model(&models.TagModel{}).
		Where("chat_id = ? AND name = ?", chatID, oldName).
		Update("name", newName)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return pkg.ErrTagExists
		}
		return res.Error
	}

	if res.RowsAffected == 0 {
		return pkg.ErrTagNotFound
	}

	return nil
}

func (r *ORMTagRepository) Delete(ctx context.Context, chatID int64, name string) error {
	res := r.db.WithContext(ctx).
		Where("chat_id = ? AND name = ?", chatID, name).
		Delete(&models.TagModel{})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return pkg.ErrTagNotFound
	}

	return nil
}
