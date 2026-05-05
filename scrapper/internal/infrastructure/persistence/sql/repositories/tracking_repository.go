package repositories

import (
	"context"
	"database/sql"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/repositories"
)

type SqlTrackingRepository struct {
	db     *sql.DB
	logger *logger.Slog
}

func NewSQLTrackingRepository(db *sql.DB, log *logger.Slog) repo.TrackingRepository {
	return &SqlTrackingRepository{
		db:     db,
		logger: log,
	}
}

func (r *SqlTrackingRepository) FindSubscribers(ctx context.Context, url string) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.chat_id
		FROM subscriptions s
		JOIN links l ON l.id = s.link_id
		WHERE l.url = $1
	`, url)
	if err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		r.logger.Error("failed to close rows", "error", err)
	}

	var result []int64

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *SqlTrackingRepository) GetAllTrackedLinks(ctx context.Context) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT url, last_updated
		FROM links
	`)
	if err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		r.logger.Error("failed to close rows", "error", err)
	}

	var result []domain.Link

	for rows.Next() {
		var link domain.Link
		if err := rows.Scan(&link.URL, &link.LastUpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *SqlTrackingRepository) UpdateLastUpdated(ctx context.Context, url string, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE links
		SET last_updated = $1
		WHERE url = $2
	`, t, url)
	return err
}
