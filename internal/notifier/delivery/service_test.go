package delivery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/notifier/delivery"
)

var errBoom = errors.New("boom")

type mockLedger struct {
	mock.Mock
}

func (m *mockLedger) Reserve(ctx context.Context, dedupKey string) (bool, error) {
	args := m.Called(ctx, dedupKey)
	return args.Bool(0), args.Error(1)
}

type mockSender struct {
	mock.Mock
}

func (m *mockSender) SendConfirm(email, confirmURL string) error {
	return m.Called(email, confirmURL).Error(0)
}

func (m *mockSender) SendRelease(email, repoFullName, tag, releaseURL, unsubscribeURL string) error {
	return m.Called(email, repoFullName, tag, releaseURL, unsubscribeURL).Error(0)
}

func anyArgs(n int) []any {
	args := make([]any, n)
	for i := range args {
		args[i] = mock.Anything
	}
	return args
}

func TestSendRelease(t *testing.T) {
	t.Run("first delivery sends and reports delivered", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(true, nil).Once()
		sender.On("SendRelease", "a@b.com", "owner/repo", "v1", "url", "unsub").Return(nil).Once()

		delivered, err := delivery.New(ledger, sender).SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)

		require.NoError(t, err)
		assert.True(t, delivered)
		ledger.AssertExpectations(t)
		sender.AssertExpectations(t)
	})

	t.Run("duplicate is deduplicated and not resent", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(false, nil).Once()

		delivered, err := delivery.New(ledger, sender).SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)

		require.NoError(t, err)
		assert.False(t, delivered)
		sender.AssertNotCalled(t, "SendRelease")
		ledger.AssertExpectations(t)
	})

	t.Run("reserve ok then smtp fail does not resend on retry", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(true, nil).Once()
		ledger.On("Reserve", anyArgs(2)...).Return(false, nil).Once()
		sender.On("SendRelease", anyArgs(5)...).Return(errBoom).Once()

		svc := delivery.New(ledger, sender)

		delivered, err := svc.SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)
		require.ErrorIs(t, err, errBoom)
		assert.False(t, delivered)

		delivered, err = svc.SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)
		require.NoError(t, err)
		assert.False(t, delivered)

		sender.AssertNumberOfCalls(t, "SendRelease", 1)
		ledger.AssertExpectations(t)
	})

	t.Run("ledger error is propagated", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(false, errBoom).Once()

		_, err := delivery.New(ledger, sender).SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)

		require.ErrorIs(t, err, errBoom)
		sender.AssertNotCalled(t, "SendRelease")
	})
}

func TestSendConfirm(t *testing.T) {
	t.Run("first delivery sends", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(true, nil).Once()
		sender.On("SendConfirm", "a@b.com", "confirm-url").Return(nil).Once()

		delivered, err := delivery.New(ledger, sender).SendConfirm(
			context.Background(), "a@b.com", "confirm-url",
		)

		require.NoError(t, err)
		assert.True(t, delivered)
		sender.AssertExpectations(t)
	})

	t.Run("duplicate confirm is deduplicated", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(false, nil).Once()

		delivered, err := delivery.New(ledger, sender).SendConfirm(
			context.Background(), "a@b.com", "confirm-url",
		)

		require.NoError(t, err)
		assert.False(t, delivered)
		sender.AssertNotCalled(t, "SendConfirm")
	})
}
