package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/slickip/link-tracker/pkg"
	"github.com/slickip/link-tracker/scrapper/internal/application/services"
)

type CreateTagRequest struct {
	ChatID int64  `json:"chatId"`
	Name   string `json:"name"`
}

type RenameTagRequest struct {
	ChatID  int64  `json:"chatId"`
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type DeleteTagRequest struct {
	ChatID int64  `json:"chatId"`
	Name   string `json:"name"`
}

type TagHandler struct {
	service *services.TagService
}

func NewTagHandler(service *services.TagService) *TagHandler {
	return &TagHandler{service: service}
}

func (h *TagHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err := h.service.CreateTag(r.Context(), req.ChatID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, pkg.ErrChatNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, pkg.ErrTagExists):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, pkg.ErrInvalidRequest):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *TagHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("chatId")

	chatID, err := strconv.ParseInt(chatIDStr, base10, bitSize64)
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}

	tags, err := h.service.ListTags(r.Context(), chatID)
	if err != nil {
		if errors.Is(err, pkg.ErrChatNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tags); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *TagHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
	var req RenameTagRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err := h.service.RenameTag(r.Context(), req.ChatID, req.OldName, req.NewName)
	if err != nil {
		switch {
		case errors.Is(err, pkg.ErrChatNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, pkg.ErrTagNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, pkg.ErrTagExists):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, pkg.ErrInvalidRequest):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TagHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	var req DeleteTagRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteTag(r.Context(), req.ChatID, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, pkg.ErrChatNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, pkg.ErrTagNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, pkg.ErrInvalidRequest):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
