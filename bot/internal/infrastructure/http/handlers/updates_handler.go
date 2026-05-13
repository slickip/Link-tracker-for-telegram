package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/bot/internal/application/services"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api"
)

type LinkUpdateHandler interface {
	HandleLinkUpdate(ctx context.Context, update api.LinkUpdate) error
}

type UpdatesHandler struct {
	handler LinkUpdateHandler
}

func NewUpdatesHandler(handler LinkUpdateHandler) *UpdatesHandler {
	return &UpdatesHandler{
		handler: handler,
	}
}

func (h *UpdatesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var update api.LinkUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.handler.HandleLinkUpdate(r.Context(), update); err != nil {
		if errors.Is(err, services.ErrInvalidLinkUpdate) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
