package handlers

import (
	"encoding/json"
	"net/http"

	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type MessageSender interface {
	SendMessage(chatID int64, text string) error
}

type UpdatesHandler struct {
	bot MessageSender
}

func NewUpdatesHandler(bot MessageSender) *UpdatesHandler {
	return &UpdatesHandler{
		bot: bot,
	}
}

func (h *UpdatesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var update api.LinkUpdate
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if update.URL == "" || len(update.TgChatIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	text := update.Description
	if text == "" {
		text = "Обнаружено обновление по ссылке: " + update.URL
	}

	for _, chatID := range update.TgChatIDs {
		_ = h.bot.SendMessage(
			chatID,
			text,
		)
	}

	w.WriteHeader(http.StatusOK)
}
