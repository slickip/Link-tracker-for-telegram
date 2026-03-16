package handlers

import (
	"encoding/json"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/adapters"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/dto"
)

type UpdatesHandler struct {
	bot *adapters.Bot
}

func NewUpdatesHandler(bot *adapters.Bot) *UpdatesHandler {
	return &UpdatesHandler{
		bot: bot,
	}
}

func (h *UpdatesHandler) Handle(w http.ResponseWriter, r *http.Request) {

	var update dto.LinkUpdate

	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, chatID := range update.TgChatIDs {

		_ = h.bot.SendMessage(
			chatID,
			"Обнаружено обновление по ссылке: "+update.URL,
		)
	}

	w.WriteHeader(http.StatusOK)
}
