package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

func TestGetSubscriptions(t *testing.T) {
	tag := "v1.0.0"

	cases := []struct {
		name           string
		email          string
		serviceResult  []subscription.SubscriptionView
		serviceErr     error
		expectedStatus int
		expectedLen    int
	}{
		{
			name:  "OK_WithResults",
			email: "test@example.com",
			serviceResult: []subscription.SubscriptionView{
				{Email: "test@example.com", Repo: "golang/go", Confirmed: true, LastSeenTag: &tag},
			},
			expectedStatus: http.StatusOK,
			expectedLen:    1,
		},
		{
			name:           "OK_EmptyResults",
			email:          "test@example.com",
			serviceResult:  []subscription.SubscriptionView{},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name:           "Error_InvalidEmail",
			email:          "bad-email",
			serviceResult:  []subscription.SubscriptionView(nil),
			serviceErr:     subscription.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error_Internal",
			email:          "test@example.com",
			serviceResult:  []subscription.SubscriptionView(nil),
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(mockService)
			svc.On("GetSubscriptionsByEmail", mock.Anything, mock.Anything).
				Return(tc.serviceResult, tc.serviceErr)

			h := handlers.NewSubscriptionHandler(svc)

			r := httptest.NewRequest(http.MethodGet, "/api/subscriptions?email="+tc.email, nil)
			w := httptest.NewRecorder()

			h.GetSubscriptions(w, r)

			assert.Equal(t, tc.expectedStatus, w.Code)
			svc.AssertExpectations(t)

			if tc.expectedStatus == http.StatusOK {
				var result []map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Len(t, result, tc.expectedLen)
			}
		})
	}
}
