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

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/rest"
)

const testSaga = "11111111-1111-1111-1111-111111111111"

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
		svc.On("ReserveConfirmation", mock.Anything, testSaga, "a@b.com", "http://x/c").
			Return(true, nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve",
			`{"saga_id":"`+testSaga+`","email":"a@b.com","confirm_url":"http://x/c"}`)

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

	t.Run("malformed fields are rejected before the service", func(t *testing.T) {
		cases := map[string]string{
			"bad saga_id": `{"saga_id":"not-a-uuid","email":"a@b.com","confirm_url":"http://x/c"}`,
			"bad email":   `{"saga_id":"` + testSaga + `","email":"nope","confirm_url":"http://x/c"}`,
			"bad url":     `{"saga_id":"` + testSaga + `","email":"a@b.com","confirm_url":"/relative"}`,
			"missing":     `{"saga_id":"` + testSaga + `"}`,
		}
		for name, body := range cases {
			t.Run(name, func(t *testing.T) {
				svc := new(mockService)
				rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve", body)

				assert.Equal(t, http.StatusBadRequest, rec.Code)
				svc.AssertNotCalled(t, "ReserveConfirmation")
			})
		}
	})

	t.Run("service error maps to 500", func(t *testing.T) {
		svc := new(mockService)
		svc.On("ReserveConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(false, errors.New("boom")).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/reserve",
			`{"saga_id":"`+testSaga+`","email":"a@b.com","confirm_url":"http://x/c"}`)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestCommitAndCancel(t *testing.T) {
	t.Run("commit ok", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, testSaga).Return(true, nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/commit",
			`{"saga_id":"`+testSaga+`"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		svc.AssertExpectations(t)
	})

	t.Run("commit on missing reservation maps to 409", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, testSaga).
			Return(false, delivery.ErrNotReserved).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/commit",
			`{"saga_id":"`+testSaga+`"}`)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("commit on canceled reservation maps to 409", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, testSaga).
			Return(false, delivery.ErrCanceled).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/commit",
			`{"saga_id":"`+testSaga+`"}`)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("cancel ok", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CancelConfirmation", mock.Anything, testSaga).Return(nil).Once()

		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/cancel",
			`{"saga_id":"`+testSaga+`"}`)

		assert.Equal(t, http.StatusOK, rec.Code)
		svc.AssertExpectations(t)
	})

	t.Run("bad saga_id is rejected", func(t *testing.T) {
		svc := new(mockService)
		rec := do(t, rest.NewHandler(svc), http.MethodPost, "/v1/confirmations/commit",
			`{"saga_id":"nope"}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		svc.AssertNotCalled(t, "CommitConfirmation")
	})

	t.Run("wrong method is rejected", func(t *testing.T) {
		svc := new(mockService)
		rec := do(t, rest.NewHandler(svc), http.MethodGet, "/v1/confirmations/commit", "")

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}
