package api

import (
	"strings"
	"time"
)

const DefaultLinkUpdateNoticePrefix = "Обнаружено обновление по ссылке: "

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

func (u LinkUpdate) SubscriberNotificationBody() string {
	if strings.TrimSpace(u.Description) != "" {
		return u.Description
	}

	return DefaultLinkUpdateNoticePrefix + u.URL
}
