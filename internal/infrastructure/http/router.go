package http

import (
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure/http/handlers"
)

func NewRouter(
	chatHandler *handlers.ChatHandler,
	linkHandler *handlers.LinkHandler,
	updatesHandler *handlers.UpdatesHandler,
) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("/tg-chat/", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodPost {
			chatHandler.RegisterChat(w, r)
			return
		}

		if r.Method == http.MethodDelete {
			chatHandler.DeleteChat(w, r)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case http.MethodGet:
			linkHandler.ListLinks(w, r)

		case http.MethodPost:
			linkHandler.AddLink(w, r)

		case http.MethodDelete:
			linkHandler.RemoveLink(w, r)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/updates", updatesHandler.Handle)
	return mux
}
