package repositories

import (
	"context"
	"database/sql"
	"errors"

	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

var ErrChatNotFound = errors.New("chat not found")

type SQLChatRepository struct {
	db *sql.DB
}

func NewSQLChatRepository(db *sql.DB) repo.ChatRepository {
	return &SQLChatRepository{db: db}
}

func (r *SQLChatRepository) Add(ctx context.Context, chatID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chats (id)
		VALUES ($1)
		ON CONFLICT DO NOTHING
	`, chatID)

	return err
}

func (r *SQLChatRepository) Remove(ctx context.Context, chatID int64) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM chats WHERE id = $1
	`, chatID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrChatNotFound
	}

	return nil
}

func (r *SQLChatRepository) Exists(ctx context.Context, chatID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM chats WHERE id = $1
		)
	`, chatID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
