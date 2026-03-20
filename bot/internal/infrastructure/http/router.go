package http

import (
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/infrastructure/http/handlers"
)

func NewBotRouter(
	updatesHandler *handlers.UpdatesHandler,
) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("/updates", updatesHandler.Handle)

	return mux
}
