package rest_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/rest"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) (bool, error) {
	args := m.Called(ctx, sagaID, email, confirmURL)
	return args.Bool(0), args.Error(1)
}

func (m *mockService) CommitConfirmation(ctx context.Context, sagaID string) (bool, error) {
	args := m.Called(ctx, sagaID)
	return args.Bool(0), args.Error(1)
}

func (m *mockService) CancelConfirmation(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReserve(t *testing.T) {
	t.Run("valid request reserves", func(t *testing.T) {
		svc := new(mockService)
		svc.On("ReserveConfirmation", mock.Anything, "s1", "a@b.com", "http://x/c").
			Return(true, nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve",
			`{"saga_id":"s1","email":"a@b.com","confirm_url":"http://x/c"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"ok":true}`, rec.Body.String())
		svc.AssertExpectations(t)
	})

	t.Run("invalid json is rejected", func(t *testing.T) {
		svc := new(mockService)
		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve", `{`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		svc.AssertNotCalled(t, "ReserveConfirmation")
	})

	t.Run("missing fields are rejected", func(t *testing.T) {
		svc := new(mockService)
		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve",
			`{"saga_id":"s1"}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		svc.AssertNotCalled(t, "ReserveConfirmation")
	})

	t.Run("service error maps to 500", func(t *testing.T) {
		svc := new(mockService)
		svc.On("ReserveConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(false, errors.New("boom")).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve",
			`{"saga_id":"s1","email":"a@b.com","confirm_url":"http://x/c"}`)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestCommitAndCancel(t *testing.T) {
	t.Run("commit ok", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, "s1").Return(true, nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/commit",
			`{"saga_id":"s1"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		svc.AssertExpectations(t)
	})

	t.Run("cancel ok", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CancelConfirmation", mock.Anything, "s1").Return(nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/cancel",
			`{"saga_id":"s1"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		svc.AssertExpectations(t)
	})

	t.Run("wrong method is rejected", func(t *testing.T) {
		svc := new(mockService)
		rec := do(t, rest.NewHandler(svc), http.MethodGet, "/v1/confirmations/commit", "")

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}
