package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func TestSubscribe(t *testing.T) {
	cases := []struct {
		name            string
		body            string
		serviceErr      error
		skipServiceCall bool
		expectedStatus  int
	}{
		{
			name:           "OK",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:            "Error_InvalidJSON",
			body:            `{bad json}`,
			skipServiceCall: true,
			expectedStatus:  http.StatusBadRequest,
		},
		{
			name:           "Error_InvalidEmail",
			body:           `{"email":"bad","repo":"golang/go"}`,
			serviceErr:     subscription.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error_InvalidRepo",
			body:           `{"email":"test@example.com","repo":"bad"}`,
			serviceErr:     subscription.ErrInvalidRepo,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error_RepoNotFound",
			body:           `{"email":"test@example.com","repo":"owner/repo"}`,
			serviceErr:     subscription.ErrRepoNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Error_AlreadySubscribed",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			serviceErr:     subscription.ErrAlreadySubscribed,
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Error_Internal",
			body:           `{"email":"test@example.com","repo":"golang/go"}`,
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(mockService)
			if !tc.skipServiceCall {
				svc.On("Subscribe", mock.Anything, mock.Anything, mock.Anything).Return(tc.serviceErr)
			}

			h := handlers.NewSubscriptionHandler(svc)
			r := httptest.NewRequest(http.MethodPost, "/api/subscribe", strings.NewReader(tc.body))
			w := httptest.NewRecorder()

			h.Subscribe(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			svc.AssertExpectations(t)
		})
	}
}
