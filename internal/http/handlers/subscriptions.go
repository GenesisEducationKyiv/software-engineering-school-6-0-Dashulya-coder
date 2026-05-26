package handlers

import (
	"errors"
	"net/http"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/dto"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func (h *SubscriptionHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	subs, err := h.service.GetSubscriptionsByEmail(r.Context(), email)
	if err != nil {
		switch {
		case errors.Is(err, subscription.ErrInvalidEmail):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		return
	}

	response := make([]dto.SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		response = append(response, dto.SubscriptionResponse{
			Email:       sub.Email,
			Repo:        sub.Repo,
			Confirmed:   sub.Confirmed,
			LastSeenTag: sub.LastSeenTag,
		})
	}

	writeJSON(w, http.StatusOK, response)
}
