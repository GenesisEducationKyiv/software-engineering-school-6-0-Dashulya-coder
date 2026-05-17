package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
			name:  "success with results",
			email: "test@example.com",
			serviceResult: []subscription.SubscriptionView{
				{Email: "test@example.com", Repo: "golang/go", Confirmed: true, LastSeenTag: &tag},
			},
			expectedStatus: http.StatusOK,
			expectedLen:    1,
		},
		{
			name:           "success empty results",
			email:          "test@example.com",
			serviceResult:  []subscription.SubscriptionView{},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
		{
			name:           "invalid email",
			email:          "bad-email",
			serviceErr:     subscription.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal error",
			email:          "test@example.com",
			serviceErr:     errors.New("unexpected"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{
				getSubscriptionsByEmailFn: func(_ context.Context, _ string) ([]subscription.SubscriptionView, error) {
					return tc.serviceResult, tc.serviceErr
				},
			}
			h := handlers.NewSubscriptionHandler(svc)

			r := httptest.NewRequest(http.MethodGet, "/api/subscriptions?email="+tc.email, nil)
			w := httptest.NewRecorder()

			h.GetSubscriptions(w, r)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedStatus == http.StatusOK {
				var result []map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(result) != tc.expectedLen {
					t.Fatalf("expected %d items, got %d", tc.expectedLen, len(result))
				}
			}
		})
	}
}
