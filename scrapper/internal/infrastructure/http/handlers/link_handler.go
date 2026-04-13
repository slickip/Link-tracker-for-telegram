package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/internal/domain"
)

type AddLinkRequest struct {
	ChatID int64    `json:"chatId"`
	URL    string   `json:"url"`
	Tags   []string `json:"tags"`
}

type LinkHandler struct {
	service *services.LinkService
}

func NewLinkHandler(service *services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	var req AddLinkRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	link := domain.Link{
		URL:  req.URL,
		Tags: req.Tags,
	}

	err = h.service.AddLink(req.ChatID, link)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *LinkHandler) RemoveLink(w http.ResponseWriter, r *http.Request) {
	var req AddLinkRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err = h.service.RemoveLink(req.ChatID, req.URL)
	if err != nil {
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

	links, err := h.service.ListLinks(chatID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(links); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
