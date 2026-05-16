package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
