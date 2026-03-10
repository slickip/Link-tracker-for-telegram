package domain

type TrackState string

const (
	StateIdle TrackState = "idle"
	StateWaitingForURL  TrackState = "waiting_for_url"
	StateWaitingForTags TrackState = "waiting_for_tags"
)

type TrackSession struct {
	ChatID int64
	State TrackState
	URL string
}
