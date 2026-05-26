package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	err := h.service.Unsubscribe(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, subscription.ErrInvalidToken):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, subscription.ErrTokenNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Unsubscribed successfully"})
}
