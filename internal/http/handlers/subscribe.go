package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/dto"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req dto.SubscribeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err := h.service.Subscribe(r.Context(), req.Email, req.Repo)
	if err != nil {
		switch {
		case errors.Is(err, subscription.ErrInvalidEmail),
			errors.Is(err, subscription.ErrInvalidRepo):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, subscription.ErrRepoNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		case errors.Is(err, subscription.ErrAlreadySubscribed):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription successful. Confirmation email sent.",
	})
}
