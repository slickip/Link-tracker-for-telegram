package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/slickip/link-tracker/pkg/logger"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
	repo "github.com/slickip/link-tracker/scrapper/internal/repositories"
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

func (r *SqlTrackingRepository) FindSubscribers(ctx context.Context, linkID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT chat_id
		FROM subscriptions
		WHERE link_id = $1
	`, linkID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var result []int64

	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}

		result = append(result, chatID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *SqlTrackingRepository) GetTrackedLinksBatch(ctx context.Context, limit, offset int) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT l.id, l.url, l.last_updated
		FROM links l
		JOIN subscriptions s ON s.link_id = l.id
		ORDER BY l.id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var links []domain.Link

	for rows.Next() {
		var link domain.Link

		err := rows.Scan(
			&link.ID,
			&link.URL,
			&link.LastUpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func (r *SqlTrackingRepository) UpdateLastUpdated(ctx context.Context, linkID int64, t time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE links
		SET last_updated = $1
		WHERE id = $2
	`, t, linkID)

	return err
}
