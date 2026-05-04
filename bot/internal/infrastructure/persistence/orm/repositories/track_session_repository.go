package repositories

import (
	"context"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain/repositories"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/persistence/orm/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ORMTrackSessionRepository struct {
	db *gorm.DB
}

func NewORMTrackSessionRepository(db *gorm.DB) repo.TrackSessionRepository {
	return &ORMTrackSessionRepository{db: db}
}

func (r *ORMTrackSessionRepository) Get(ctx context.Context, chatID int64) (domain.TrackSession, bool, error) {
	var model models.TrackSessionModel

	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return domain.TrackSession{}, false, nil
		}
		return domain.TrackSession{}, false, err
	}

	return domain.TrackSession{
		ChatID: model.ChatID,
		State:  domain.TrackState(model.State),
		URL:    model.URL,
	}, true, nil
}

func (r *ORMTrackSessionRepository) Set(ctx context.Context, chatID int64, session domain.TrackSession) error {
	model := models.TrackSessionModel{
		ChatID: chatID,
		State:  string(session.State),
		URL:    session.URL,
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chat_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"state", "url"}),
		}).
		Create(&model).Error
}

func (r *ORMTrackSessionRepository) Reset(ctx context.Context, chatID int64) error {
	return r.db.WithContext(ctx).
		Delete(&models.TrackSessionModel{}, "chat_id = ?", chatID).Error
}
