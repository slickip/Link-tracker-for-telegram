package repositories

import (
	"context"
	"database/sql"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain/repositories"
)

type SqlChatLinkRepository struct {
	db *sql.DB
}

func NewSQLChatLinkRepository(db *sql.DB) repo.LinkRepository {
	return &SqlChatLinkRepository{db: db}
}

func (r *SqlChatLinkRepository) Add(ctx context.Context, chatID int64, link domain.Link) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
			INSERT INTO link_tags (link_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, linkID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SqlChatLinkRepository) Remove(ctx context.Context, chatID int64, url string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM subscriptions
		WHERE chat_id = $1
		  AND link_id = (SELECT id FROM links WHERE url = $2)
	`, chatID, url)

	return err
}

func (r *SqlChatLinkRepository) List(ctx context.Context, chatID int64) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.url, l.last_updated
		FROM links l
		JOIN subscriptions s ON l.id = s.link_id
		WHERE s.chat_id = $1
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (r *SqlChatLinkRepository) ListByTag(ctx context.Context, chatID int64, tag string) ([]domain.Link, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT l.url, l.last_updated
		FROM links l
		JOIN subscriptions s ON l.id = s.link_id
		JOIN link_tags lt ON l.id = lt.link_id
		JOIN tags t ON t.id = lt.tag_id
		WHERE s.chat_id = $1 AND t.name = $2
	`, chatID, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
