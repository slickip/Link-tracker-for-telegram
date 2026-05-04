package http

import (
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
)

func NewRouter(
	chatHandler *handlers.ChatHandler,
	linkHandler *handlers.LinkHandler,
	tagHandler *handlers.TagHandler,
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

	mux.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tagHandler.ListTags(w, r)
		case http.MethodPost:
			tagHandler.CreateTag(w, r)
		case http.MethodPut:
			tagHandler.RenameTag(w, r)
		case http.MethodDelete:
			tagHandler.DeleteTag(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return mux
}
