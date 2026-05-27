package repositories

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/persistence/orm/models"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ORMOutboxRepository struct {
	db *gorm.DB
}

func NewORMOutboxRepository(db *gorm.DB) repo.OutboxRepository {
	return &ORMOutboxRepository{db: db}
}

func (r *ORMOutboxRepository) SaveLinkUpdateOutbox(
	ctx context.Context,
	linkID int64,
	newUpdatedAt time.Time,
	messages []domain.OutboxMessage,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !newUpdatedAt.IsZero() {
			err := tx.Model(&models.LinkModel{}).
				Where("id = ?", linkID).
				Update("last_updated", newUpdatedAt).Error
			if err != nil {
				return err
			}
		}

		for _, message := range messages {
			if err := tx.Create(toOutboxModel(message)).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ORMOutboxRepository) SaveOutboxMessages(
	ctx context.Context,
	messages []domain.OutboxMessage,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, message := range messages {
			if err := tx.Create(toOutboxModel(message)).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *ORMOutboxRepository) GetPendingMessages(
	ctx context.Context,
	limit int,
) ([]domain.OutboxMessage, error) {
	var messageModels []models.OutboxMessageModel

	err := r.db.WithContext(ctx).
		Where("status = ?", domain.OutboxStatusPending).
		Order("created_at").
		Limit(limit).
		Find(&messageModels).Error
	if err != nil {
		return nil, err
	}

	messages := make([]domain.OutboxMessage, 0, len(messageModels))
	for _, messageModel := range messageModels {
		messages = append(messages, toOutboxDomain(messageModel))
	}

	return messages, nil
}

func (r *ORMOutboxRepository) MarkAsSent(ctx context.Context, id int64) error {
	now := time.Now().UTC()

	return r.db.WithContext(ctx).
		Model(&models.OutboxMessageModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(domain.OutboxStatusSent),
			"sent_at":    &now,
			"updated_at": now,
		}).Error
}

func (r *ORMOutboxRepository) MarkPublishFailed(
	ctx context.Context,
	id int64,
	publishErr error,
) error {
	errorText := ""
	if publishErr != nil {
		errorText = publishErr.Error()
	}

	return r.db.WithContext(ctx).
		Model(&models.OutboxMessageModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": errorText,
			"updated_at": time.Now().UTC(),
		}).Error
}

func toOutboxModel(message domain.OutboxMessage) *models.OutboxMessageModel {
	return &models.OutboxMessageModel{
		Topic:      message.Topic,
		MessageKey: message.MessageKey,
		Payload:    datatypes.JSON(message.Payload),
		Status:     string(message.Status),
		Attempts:   message.Attempts,
		CreatedAt:  message.CreatedAt,
		UpdatedAt:  message.UpdatedAt,
	}
}

func toOutboxDomain(messageModel models.OutboxMessageModel) domain.OutboxMessage {
	return domain.OutboxMessage{
		ID:         messageModel.ID,
		Topic:      messageModel.Topic,
		MessageKey: messageModel.MessageKey,
		Payload:    []byte(messageModel.Payload),
		Status:     domain.OutboxStatus(messageModel.Status),
		Attempts:   messageModel.Attempts,
		LastError:  messageModel.LastError,
		CreatedAt:  messageModel.CreatedAt,
		UpdatedAt:  messageModel.UpdatedAt,
		SentAt:     messageModel.SentAt,
	}
}
