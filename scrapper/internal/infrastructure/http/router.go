package http

import (
	"net/http"
	"time"

	scrappermetrics "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/metrics"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/infrastructure/http/handlers"
)

const (
	httpAPIScope = "http_api"
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

	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			linkHandler.ListLinks(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/links/by-tag", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			linkHandler.RemoveLinksByTag(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/links", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
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

	return metricsMiddleware(mux)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		scrappermetrics.APIRequestsTotal.
			WithLabelValues(r.URL.Path).
			Inc()

		next.ServeHTTP(w, r)

		scrappermetrics.RequestDurationMs.
			WithLabelValues(httpAPIScope, r.URL.Path).
			Observe(float64(time.Since(started).Milliseconds()))
	})
}
