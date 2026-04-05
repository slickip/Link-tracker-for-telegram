package repositories

import "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/domain"

type TrackSessionRepository interface {
	Get(chatID int64) (domain.TrackSession, bool)
	Set(chatID int64, session domain.TrackSession)
	Reset(chatID int64)
}

type InMemoryTrackSessionRepository struct {
	sessions map[int64]domain.TrackSession
}

func NewInMemoryTrackSessionRepository() *InMemoryTrackSessionRepository {
	return &InMemoryTrackSessionRepository{
		sessions: make(map[int64]domain.TrackSession),
	}
}

func (r *InMemoryTrackSessionRepository) Get(chatID int64) (domain.TrackSession, bool) {
	if r == nil {
		return domain.TrackSession{}, false
	}

	session, ok := r.sessions[chatID]
	return session, ok
}

func (r *InMemoryTrackSessionRepository) Set(chatID int64, session domain.TrackSession) {
	if r == nil {
		return
	}
	r.sessions[chatID] = session
}

func (r *InMemoryTrackSessionRepository) Reset(chatID int64) {
	if r == nil {
		return
	}
	delete(r.sessions, chatID)
}
