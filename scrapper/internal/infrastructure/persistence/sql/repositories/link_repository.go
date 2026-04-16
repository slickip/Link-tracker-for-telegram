package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/logger"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
)

type SqlChatLinkRepository struct {
	db     *sql.DB
	logger *logger.Slog
}

func NewSQLChatLinkRepository(db *sql.DB, log *logger.Slog) repo.LinkRepository {
	return &SqlChatLinkRepository{
		db:     db,
		logger: log,
	}
}

func (r *SqlChatLinkRepository) Add(ctx context.Context, chatID int64, link domain.Link) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Error("failed to rollback tx", "error", err)
		}
	}()

	var linkID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO links (url, last_updated)
		VALUES ($1, $2)
		ON CONFLICT (url) DO UPDATE SET last_updated = EXCLUDED.last_updated
		RETURNING id
	`, link.URL, link.LastUpdatedAt).Scan(&linkID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO subscriptions (chat_id, link_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, chatID, linkID)
	if err != nil {
		return err
	}

	for _, tag := range link.Tags {
		var tagID int64

		err = tx.QueryRowContext(ctx, `
			INSERT INTO tags (name)
			VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tag).Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO subscription_tags (chat_id, link_id, tag_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, chatID, linkID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SqlChatLinkRepository) Remove(ctx context.Context, chatID int64, url string) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM subscriptions
		WHERE chat_id = $1
		  AND link_id = (SELECT id FROM links WHERE url = $2)
	`, chatID, url)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return pkg.ErrLinkNotFound
	}

	return nil
}

func (r *SqlChatLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.url, l.last_updated
		FROM links l
		JOIN subscriptions s ON l.id = s.link_id
		WHERE s.chat_id = $1
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	type linkRow struct {
		ID          int64
		URL         string
		LastUpdated time.Time
	}

	var rawLinks []linkRow
	var linkIDs []int64

	for rows.Next() {
		var row linkRow
		if err := rows.Scan(&row.ID, &row.URL, &row.LastUpdated); err != nil {
			return nil, err
		}
		rawLinks = append(rawLinks, row)
		linkIDs = append(linkIDs, row.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagsByLinkID, err := r.getTagsByLinkIDs(ctx, chatID, linkIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(rawLinks))
	for _, row := range rawLinks {
		result = append(result, domain.Link{
			URL:           row.URL,
			LastUpdatedAt: row.LastUpdated,
			Tags:          tagsByLinkID[row.ID],
		})
	}

	return result, nil
}

func (r *SqlChatLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT l.id, l.url, l.last_updated
		FROM links l
		JOIN subscriptions s ON l.id = s.link_id
		JOIN subscription_tags st ON st.link_id = l.id AND st.chat_id = s.chat_id
		JOIN tags t ON t.id = st.tag_id
		WHERE s.chat_id = $1 AND t.name = $2
	`, chatID, tag)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	type linkRow struct {
		ID          int64
		URL         string
		LastUpdated time.Time
	}

	var rawLinks []linkRow
	var linkIDs []int64

	for rows.Next() {
		var row linkRow
		if err := rows.Scan(&row.ID, &row.URL, &row.LastUpdated); err != nil {
			return nil, err
		}
		rawLinks = append(rawLinks, row)
		linkIDs = append(linkIDs, row.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagsByLinkID, err := r.getTagsByLinkIDs(ctx, chatID, linkIDs)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Link, 0, len(rawLinks))
	for _, row := range rawLinks {
		result = append(result, domain.Link{
			URL:           row.URL,
			LastUpdatedAt: row.LastUpdated,
			Tags:          tagsByLinkID[row.ID],
		})
	}

	return result, nil
}

func (r *SqlChatLinkRepository) getTagsByLinkIDs(ctx context.Context, chatID int64, linkIDs []int64) (map[int64][]string, error) {
	if len(linkIDs) == 0 {
		return map[int64][]string{}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT st.link_id, t.name
		FROM subscription_tags st
		JOIN tags t ON t.id = st.tag_id
		WHERE st.chat_id = $1
		  AND st.link_id = ANY($2)
	`, chatID, pq.Array(linkIDs))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	result := make(map[int64][]string)
	for rows.Next() {
		var linkID int64
		var tag string
		if err := rows.Scan(&linkID, &tag); err != nil {
			return nil, err
		}
		result[linkID] = append(result[linkID], tag)
	}

	return result, rows.Err()
}

func (r *SqlChatLinkRepository) RemoveByTag(ctx context.Context, chatID int64, tag string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM subscriptions s
		WHERE s.chat_id = $1
		  AND s.link_id IN (
			  SELECT st.link_id
			  FROM subscription_tags st
			  JOIN tags t ON t.id = st.tag_id
			  WHERE st.chat_id = $1
			    AND t.name = $2
		  )
	`, chatID, tag)
	if err != nil {
		return 0, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rows, nil
}
