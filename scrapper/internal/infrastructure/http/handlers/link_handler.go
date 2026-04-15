package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type AddLinkRequest struct {
	ChatID int64    `json:"chatId"`
	URL    string   `json:"url"`
	Tags   []string `json:"tags"`
}

type RemoveLinkRequest struct {
	ChatID int64  `json:"chatId"`
	URL    string `json:"url"`
}

type LinkHandler struct {
	service *services.LinkService
}

func NewLinkHandler(service *services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	var req AddLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	link := domain.Link{
		URL:  req.URL,
		Tags: req.Tags,
	}

	err := h.service.AddLink(r.Context(), req.ChatID, link)
	if err != nil {
		if errors.Is(err, pkg.ErrChatNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *LinkHandler) RemoveLink(w http.ResponseWriter, r *http.Request) {
	var req RemoveLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err := h.service.RemoveLink(r.Context(), req.ChatID, req.URL)
	if err != nil {
		if errors.Is(err, pkg.ErrChatNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *LinkHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("chatId")

	chatID, err := strconv.ParseInt(chatIDStr, base10, bitSize64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	links, err := h.service.ListLinks(r.Context(), chatID)
	if err != nil {
		if errors.Is(err, pkg.ErrChatNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(links); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
