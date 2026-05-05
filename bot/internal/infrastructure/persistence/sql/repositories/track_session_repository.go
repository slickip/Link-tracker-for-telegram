package repositories

import (
	"context"
	"database/sql"
	"errors"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"
	repo "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain/repositories"
)

type SQLTrackSessionRepository struct {
	db *sql.DB
}

func NewSQLTrackSessionRepository(db *sql.DB) repo.TrackSessionRepository {
	return &SQLTrackSessionRepository{db: db}
}

func (r *SQLTrackSessionRepository) Get(ctx context.Context, chatID int64) (domain.TrackSession, bool, error) {
	var session domain.TrackSession

	err := r.db.QueryRowContext(ctx, `
		SELECT chat_id, state, url
		FROM track_sessions
		WHERE chat_id = $1
	`, chatID).Scan(&session.ChatID, &session.State, &session.URL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TrackSession{}, false, nil
		}
		return domain.TrackSession{}, false, err
	}

	return session, true, nil
}

func (r *SQLTrackSessionRepository) Set(ctx context.Context, chatID int64, session domain.TrackSession) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO track_sessions (chat_id, state, url)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id)
		DO UPDATE SET
			state = EXCLUDED.state,
			url = EXCLUDED.url
	`, chatID, session.State, session.URL)

	return err
}

func (r *SQLTrackSessionRepository) Reset(ctx context.Context, chatID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM track_sessions
		WHERE chat_id = $1
	`, chatID)

	return err
}
