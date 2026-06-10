package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/slickip/link-tracker/pkg"
	"github.com/slickip/link-tracker/scrapper/internal/application/services"
	"github.com/slickip/link-tracker/scrapper/internal/domain"
)

const tgChatIDHeader = "Tg-Chat-Id"

type AddLinkRequest struct {
	URL  string   `json:"url"`
	Tags []string `json:"tags"`
}

type RemoveLinkRequest struct {
	URL string `json:"url"`
}

type RemoveLinksByTagRequest struct {
	Tag string `json:"tag"`
}

type RemoveLinksByTagResponse struct {
	RemovedCount int64 `json:"removedCount"`
}

type LinkHandler struct {
	service *services.LinkService
}

func NewLinkHandler(service *services.LinkService) *LinkHandler {
	return &LinkHandler{service: service}
}

func (h *LinkHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	chatID, err := getChatIDFromRequest(r)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	var req AddLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	link := domain.Link{
		URL:  req.URL,
		Tags: req.Tags,
	}

	err = h.service.AddLink(r.Context(), chatID, link)
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
	chatID, err := getChatIDFromRequest(r)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	var req RemoveLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err = h.service.RemoveLink(r.Context(), chatID, req.URL)
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

func (h *LinkHandler) RemoveLinksByTag(w http.ResponseWriter, r *http.Request) {
	chatID, err := getChatIDFromRequest(r)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	var req RemoveLinksByTagRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	removedCount, err := h.service.RemoveLinksByTag(r.Context(), chatID, req.Tag)
	if err != nil {
		if errors.Is(err, pkg.ErrChatNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(RemoveLinksByTagResponse{
		RemovedCount: removedCount,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *LinkHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	chatID, err := getChatIDFromRequest(r)
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

func getChatIDFromRequest(r *http.Request) (int64, error) {
	chatIDStr := r.Header.Get(tgChatIDHeader)

	return strconv.ParseInt(chatIDStr, base10, bitSize64)
}
