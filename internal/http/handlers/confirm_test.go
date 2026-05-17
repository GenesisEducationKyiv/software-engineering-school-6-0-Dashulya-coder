package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	chi "github.com/go-chi/chi/v5"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func TestConfirmHandler(t *testing.T) {
	cases := []struct {
		name           string
		token          string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "success",
			token:          "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid token",
			token:          "",
			serviceErr:     subscription.ErrInvalidToken,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "token not found",
			token:          "unknown-token",
			serviceErr:     subscription.ErrTokenNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal error",
			token:          "some-token",
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{
				confirmFn: func(_ context.Context, _ string) error { return tc.serviceErr },
			}
			h := handlers.NewSubscriptionHandler(svc)

			r := httptest.NewRequest(http.MethodGet, "/api/confirm/"+tc.token, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("token", tc.token)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			h.Confirm(w, r)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
