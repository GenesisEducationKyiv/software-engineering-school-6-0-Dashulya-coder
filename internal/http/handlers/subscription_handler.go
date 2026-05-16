package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/dto"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

type subscriptionService interface {
	Subscribe(ctx context.Context, email, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	GetSubscriptionsByEmail(ctx context.Context, email string) ([]subscription.SubscriptionView, error)
}

type SubscriptionHandler struct {
	service subscriptionService
}

func NewSubscriptionHandler(svc subscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: svc}
}

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

func (h *SubscriptionHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	err := h.service.Confirm(r.Context(), token)
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

	writeJSON(w, http.StatusOK, map[string]string{"message": "Subscription confirmed successfully"})
}

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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
