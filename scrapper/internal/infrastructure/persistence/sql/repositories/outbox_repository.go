package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/slickip/link-tracker/pkg/logger"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
	repo "github.com/slickip/link-tracker/scrapper/internal/repositories"
)

type SQLOutboxRepository struct {
	db     *sql.DB
	logger *logger.Slog
}

func NewSQLOutboxRepository(db *sql.DB, log *logger.Slog) repo.OutboxRepository {
	return &SQLOutboxRepository{
		db:     db,
		logger: log,
	}
}

func (r *SQLOutboxRepository) SaveLinkUpdateOutbox(
	ctx context.Context,
	linkID int64,
	newUpdatedAt time.Time,
	messages []domain.OutboxMessage,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Error("failed to rollback tx", "error", err)
		}
	}()

	if !newUpdatedAt.IsZero() {
		_, err = tx.ExecContext(ctx, `
			UPDATE links
			SET last_updated = $1
			WHERE id = $2
		`, newUpdatedAt, linkID)
		if err != nil {
			return err
		}
	}

	for _, message := range messages {
		if err := insertOutboxMessage(ctx, tx, message); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLOutboxRepository) SaveOutboxMessages(
	ctx context.Context,
	messages []domain.OutboxMessage,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Error("failed to rollback tx", "error", err)
		}
	}()

	for _, message := range messages {
		if err := insertOutboxMessage(ctx, tx, message); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLOutboxRepository) GetPendingMessages(
	ctx context.Context,
	limit int,
) ([]domain.OutboxMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, topic, message_key, payload, status, attempts, last_error, created_at, updated_at, sent_at
		FROM outbox_messages
		WHERE status = $1
		ORDER BY created_at
		LIMIT $2
	`, domain.OutboxStatusPending, limit)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var messages []domain.OutboxMessage

	for rows.Next() {
		var (
			message domain.OutboxMessage
			status  string
		)

		err := rows.Scan(
			&message.ID,
			&message.Topic,
			&message.MessageKey,
			&message.Payload,
			&status,
			&message.Attempts,
			&message.LastError,
			&message.CreatedAt,
			&message.UpdatedAt,
			&message.SentAt,
		)
		if err != nil {
			return nil, err
		}

		message.Status = domain.OutboxStatus(status)
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (r *SQLOutboxRepository) MarkAsSent(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_messages
		SET status = $1,
		    sent_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`, domain.OutboxStatusSent, id)

	return err
}

func (r *SQLOutboxRepository) MarkPublishFailed(
	ctx context.Context,
	id int64,
	publishErr error,
) error {
	errorText := ""
	if publishErr != nil {
		errorText = publishErr.Error()
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_messages
		SET attempts = attempts + 1,
		    last_error = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, errorText, id)

	return err
}

func insertOutboxMessage(
	ctx context.Context,
	tx *sql.Tx,
	message domain.OutboxMessage,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox_messages (
			topic,
			message_key,
			payload,
			status,
			attempts,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)
	`,
		message.Topic,
		message.MessageKey,
		string(message.Payload),
		message.Status,
		message.Attempts,
		message.CreatedAt,
		message.UpdatedAt,
	)

	return err
}
