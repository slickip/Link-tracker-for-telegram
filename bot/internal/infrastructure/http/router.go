package http

import (
	"net/http"

	"github.com/slickip/link-tracker/bot/internal/infrastructure/http/handlers"
)

func NewBotRouter(
	updatesHandler *handlers.UpdatesHandler,
) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("/updates", updatesHandler.Handle)

	return mux
}
