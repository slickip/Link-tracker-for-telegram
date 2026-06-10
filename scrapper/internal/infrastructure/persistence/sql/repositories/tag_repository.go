package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/slickip/link-tracker/pkg"
	"github.com/slickip/link-tracker/pkg/logger"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
	repo "github.com/slickip/link-tracker/scrapper/internal/repositories"
)

type SqlTagRepository struct {
	db     *sql.DB
	logger *logger.Slog
}

func NewSQLTagRepository(db *sql.DB, log *logger.Slog) repo.TagRepository {
	return &SqlTagRepository{
		db:     db,
		logger: log,
	}
}

func (r *SqlTagRepository) Create(ctx context.Context, chatID int64, name string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tags (chat_id, name)
		VALUES ($1, $2)
	`, chatID, name)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return pkg.ErrTagExists
		}
		return err
	}

	return nil
}

func (r *SqlTagRepository) List(ctx context.Context, chatID int64) ([]domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, chat_id, name
		FROM tags
		WHERE chat_id = $1
		ORDER BY name
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var result []domain.Tag
	for rows.Next() {
		var tag domain.Tag
		if err := rows.Scan(&tag.ID, &tag.ChatID, &tag.Name); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *SqlTagRepository) Rename(ctx context.Context, chatID int64, oldName, newName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tags
		SET name = $3
		WHERE chat_id = $1 AND name = $2
	`, chatID, oldName, newName)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return pkg.ErrTagExists
		}
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return pkg.ErrTagNotFound
	}

	return nil
}

func (r *SqlTagRepository) Delete(ctx context.Context, chatID int64, name string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM tags
		WHERE chat_id = $1 AND name = $2
	`, chatID, name)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return pkg.ErrTagNotFound
	}

	return nil
}
