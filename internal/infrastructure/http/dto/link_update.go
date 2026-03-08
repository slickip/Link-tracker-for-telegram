package dto

type LinkUpdate struct {
	URL       string  `json:"url"`
	TgChatIDs []int64 `json:"tgChatIds"`
}
