package api

import "time"

type LinkUpdate struct {
	ID        int64   `json:"id"`
	URL       string  `json:"url"`
	TgChatIDs []int64 `json:"tgChatIds"`

	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	Preview   string    `json:"preview"`

	Description string `json:"description"`
}
