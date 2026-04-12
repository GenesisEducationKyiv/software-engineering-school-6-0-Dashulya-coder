package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/dto"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/service"
)

type SubscriptionService interface {
	Subscribe(ctx context.Context, email, repo string) error
}

type SubscriptionHandler struct {
	service service.SubscriptionService
}

func NewSubscriptionHandler(service service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		service: service,
	}
}

func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var req dto.SubscribeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	err := h.service.Subscribe(r.Context(), req.Email, req.Repo)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail),
			errors.Is(err, service.ErrInvalidRepo):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return

		case errors.Is(err, service.ErrRepoNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
			return

		case errors.Is(err, service.ErrAlreadySubscribed):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
			return

		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Subscription successful. Confirmation email sent.",
	})
}

func (h *SubscriptionHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	_ = token

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "not implemented yet",
	})
}

func (h *SubscriptionHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	_ = token

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "not implemented yet",
	})
}

func (h *SubscriptionHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	_ = email

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"message": "not implemented yet",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
