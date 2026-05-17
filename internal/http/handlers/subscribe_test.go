package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

type mockService struct {
	subscribeFn               func(ctx context.Context, email, repo string) error
	confirmFn                 func(ctx context.Context, token string) error
	unsubscribeFn             func(ctx context.Context, token string) error
	getSubscriptionsByEmailFn func(ctx context.Context, email string) ([]subscription.SubscriptionView, error)
}

func (m *mockService) Subscribe(ctx context.Context, email, repo string) error {
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, email, repo)
	}
	return nil
}

func (m *mockService) Confirm(ctx context.Context, token string) error {
	if m.confirmFn != nil {
		return m.confirmFn(ctx, token)
	}
	return nil
}

func (m *mockService) Unsubscribe(ctx context.Context, token string) error {
	if m.unsubscribeFn != nil {
		return m.unsubscribeFn(ctx, token)
	}
	return nil
}

func (m *mockService) GetSubscriptionsByEmail(
	ctx context.Context,
	email string,
) ([]subscription.SubscriptionView, error) {
	if m.getSubscriptionsByEmailFn != nil {
		return m.getSubscriptionsByEmailFn(ctx, email)
	}
	return nil, nil
}

func TestSubscribe(t *testing.T) {
	cases := []struct {
		name           string
		body           string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "success",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid json",
			body:           `{bad json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid email",
			body:           `{"email":"bad","repo":"golang/go"}`,
			serviceErr:     subscription.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid repo",
			body:           `{"email":"test@example.com","repo":"bad"}`,
			serviceErr:     subscription.ErrInvalidRepo,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "repo not found",
			body:           `{"email":"test@example.com","repo":"owner/repo"}`,
			serviceErr:     subscription.ErrRepoNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "already subscribed",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			serviceErr:     subscription.ErrAlreadySubscribed,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "internal error",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{
				subscribeFn: func(_ context.Context, _, _ string) error { return tc.serviceErr },
			}
			h := handlers.NewSubscriptionHandler(svc)
			r := httptest.NewRequest(http.MethodPost, "/api/subscribe", strings.NewReader(tc.body))
			w := httptest.NewRecorder()

			h.Subscribe(w, r)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
