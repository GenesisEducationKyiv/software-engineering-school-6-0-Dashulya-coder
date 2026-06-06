package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	chi "github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func TestUnsubscribeHandler(t *testing.T) {
	cases := []struct {
		name           string
		token          string
		serviceErr     error
		expectedStatus int
	}{
		{
			name:           "OK",
			token:          "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Error_InvalidToken",
			token:          "",
			serviceErr:     subscription.ErrInvalidToken,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error_TokenNotFound",
			token:          "unknown-token",
			serviceErr:     subscription.ErrTokenNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Error_Internal",
			token:          "some-token",
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(mockService)
			svc.On("Unsubscribe", mock.Anything, mock.Anything).Return(tc.serviceErr)

			h := handlers.NewSubscriptionHandler(svc)

			r := httptest.NewRequest(http.MethodGet, "/api/unsubscribe/"+tc.token, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("token", tc.token)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			h.Unsubscribe(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			svc.AssertExpectations(t)
		})
	}
}
