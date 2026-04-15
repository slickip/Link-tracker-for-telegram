package repositories

import (
	"context"
	"errors"

	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrChatNotFound = errors.New("chat not found")

type ORMChatRepository struct {
	db *gorm.DB
}

func NewORMChatRepository(db *gorm.DB) repo.ChatRepository {
	return &ORMChatRepository{db: db}
}

func (r *ORMChatRepository) Add(ctx context.Context, chatID int64) error {
	chat := models.ChatModel{ID: chatID}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&chat).Error
}

func (r *ORMChatRepository) Remove(ctx context.Context, chatID int64) error {
	res := r.db.WithContext(ctx).
		Delete(&models.ChatModel{}, "id = ?", chatID)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return ErrChatNotFound
	}

	return nil
}

func (r *ORMChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&models.ChatModel{}).
		Where("id = ?", chatID).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
