package delivery_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
)

var errBoom = errors.New("boom")

type mockLedger struct {
	mock.Mock
}

func (m *mockLedger) Reserve(ctx context.Context, dedupKey string) (bool, error) {
	args := m.Called(ctx, dedupKey)
	return args.Bool(0), args.Error(1)
}

type mockDeliveries struct {
	mock.Mock
}

func (m *mockDeliveries) Reserve(ctx context.Context, d delivery.Delivery) (bool, error) {
	args := m.Called(ctx, d)
	return args.Bool(0), args.Error(1)
}

func (m *mockDeliveries) Get(ctx context.Context, sagaID string) (*delivery.Delivery, error) {
	args := m.Called(ctx, sagaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*delivery.Delivery), args.Error(1)
}

func (m *mockDeliveries) MarkSent(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

func (m *mockDeliveries) Cancel(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

type mockSender struct {
	mock.Mock
}

func (m *mockSender) SendConfirm(ctx context.Context, email, confirmURL string) error {
	return m.Called(ctx, email, confirmURL).Error(0)
}

func (m *mockSender) SendRelease(
	ctx context.Context,
	email, repoFullName, tag, releaseURL, unsubscribeURL string,
) error {
	return m.Called(ctx, email, repoFullName, tag, releaseURL, unsubscribeURL).Error(0)
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
		sender.On("SendRelease", mock.Anything, "a@b.com", "owner/repo", "v1", "url", "unsub").
			Return(nil).Once()

		delivered, err := delivery.New(ledger, new(mockDeliveries), sender).SendRelease(
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

		delivered, err := delivery.New(ledger, new(mockDeliveries), sender).SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)

		require.NoError(t, err)
		assert.False(t, delivered)
		sender.AssertNotCalled(t, "SendRelease")
		ledger.AssertExpectations(t)
	})

	t.Run("ledger error is propagated", func(t *testing.T) {
		ledger := new(mockLedger)
		sender := new(mockSender)
		ledger.On("Reserve", anyArgs(2)...).Return(false, errBoom).Once()

		_, err := delivery.New(ledger, new(mockDeliveries), sender).SendRelease(
			context.Background(), "a@b.com", "owner/repo", "v1", "url", "unsub",
		)

		require.ErrorIs(t, err, errBoom)
		sender.AssertNotCalled(t, "SendRelease")
	})
}

func TestReserveConfirmation(t *testing.T) {
	deliveries := new(mockDeliveries)
	deliveries.On("Reserve", mock.Anything, mock.MatchedBy(func(d delivery.Delivery) bool {
		return d.SagaID == "saga-1" && d.Email == "a@b.com" && d.ConfirmURL == "url" &&
			d.Status == delivery.StatusPending
	})).Return(true, nil).Once()

	ok, err := delivery.New(new(mockLedger), deliveries, new(mockSender)).
		ReserveConfirmation(context.Background(), "saga-1", "a@b.com", "url")

	require.NoError(t, err)
	assert.True(t, ok)
	deliveries.AssertExpectations(t)
}

func TestCommitConfirmation(t *testing.T) {
	t.Run("pending reservation sends and marks sent", func(t *testing.T) {
		deliveries := new(mockDeliveries)
		sender := new(mockSender)
		deliveries.On("Get", mock.Anything, "saga-1").Return(&delivery.Delivery{
			SagaID: "saga-1", Email: "a@b.com", ConfirmURL: "url", Status: delivery.StatusPending,
		}, nil).Once()
		sender.On("SendConfirm", mock.Anything, "a@b.com", "url").Return(nil).Once()
		deliveries.On("MarkSent", mock.Anything, "saga-1").Return(nil).Once()

		ok, err := delivery.New(new(mockLedger), deliveries, sender).
			CommitConfirmation(context.Background(), "saga-1")

		require.NoError(t, err)
		assert.True(t, ok)
		deliveries.AssertExpectations(t)
		sender.AssertExpectations(t)
	})

	t.Run("already sent is idempotent and does not resend", func(t *testing.T) {
		deliveries := new(mockDeliveries)
		sender := new(mockSender)
		deliveries.On("Get", mock.Anything, "saga-1").Return(&delivery.Delivery{
			SagaID: "saga-1", Status: delivery.StatusSent,
		}, nil).Once()

		ok, err := delivery.New(new(mockLedger), deliveries, sender).
			CommitConfirmation(context.Background(), "saga-1")

		require.NoError(t, err)
		assert.True(t, ok)
		sender.AssertNotCalled(t, "SendConfirm")
		deliveries.AssertNotCalled(t, "MarkSent")
	})

	t.Run("missing reservation errors", func(t *testing.T) {
		deliveries := new(mockDeliveries)
		deliveries.On("Get", mock.Anything, "saga-1").Return(nil, nil).Once()

		_, err := delivery.New(new(mockLedger), deliveries, new(mockSender)).
			CommitConfirmation(context.Background(), "saga-1")

		require.ErrorIs(t, err, delivery.ErrNotReserved)
	})

	t.Run("send failure leaves reservation unmarked", func(t *testing.T) {
		deliveries := new(mockDeliveries)
		sender := new(mockSender)
		deliveries.On("Get", mock.Anything, "saga-1").Return(&delivery.Delivery{
			SagaID: "saga-1", Email: "a@b.com", ConfirmURL: "url", Status: delivery.StatusPending,
		}, nil).Once()
		sender.On("SendConfirm", mock.Anything, "a@b.com", "url").Return(errBoom).Once()

		ok, err := delivery.New(new(mockLedger), deliveries, sender).
			CommitConfirmation(context.Background(), "saga-1")

		require.ErrorIs(t, err, errBoom)
		assert.False(t, ok)
		deliveries.AssertNotCalled(t, "MarkSent")
	})

	t.Run("transient mark failure is retried after a successful send", func(t *testing.T) {
		deliveries := new(mockDeliveries)
		sender := new(mockSender)
		deliveries.On("Get", mock.Anything, "saga-1").Return(&delivery.Delivery{
			SagaID: "saga-1", Email: "a@b.com", ConfirmURL: "url", Status: delivery.StatusPending,
		}, nil).Once()
		sender.On("SendConfirm", mock.Anything, "a@b.com", "url").Return(nil).Once()
		deliveries.On("MarkSent", mock.Anything, "saga-1").Return(errBoom).Once()
		deliveries.On("MarkSent", mock.Anything, "saga-1").Return(nil).Once()

		ok, err := delivery.New(new(mockLedger), deliveries, sender).
			CommitConfirmation(context.Background(), "saga-1")

		require.NoError(t, err)
		assert.True(t, ok)
		sender.AssertNumberOfCalls(t, "SendConfirm", 1)
		deliveries.AssertNumberOfCalls(t, "MarkSent", 2)
	})
}

func TestCancelConfirmation(t *testing.T) {
	deliveries := new(mockDeliveries)
	deliveries.On("Cancel", mock.Anything, "saga-1").Return(nil).Once()

	err := delivery.New(new(mockLedger), deliveries, new(mockSender)).
		CancelConfirmation(context.Background(), "saga-1")

	require.NoError(t, err)
	deliveries.AssertExpectations(t)
}
