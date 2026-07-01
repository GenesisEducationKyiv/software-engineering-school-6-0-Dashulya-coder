package server_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/notifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/server"
)

const (
	bufSize = 1024 * 1024
	testSaga = "11111111-1111-1111-1111-111111111111"
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

func (m *mockService) SendRelease(
	ctx context.Context,
	email, repoFullName, tag, releaseURL, unsubscribeURL string,
) (bool, error) {
	args := m.Called(ctx, email, repoFullName, tag, releaseURL, unsubscribeURL)
	return args.Bool(0), args.Error(1)
}

func newTestClient(t *testing.T, svc server.Service) notificationv1.NotificationServiceClient {
	t.Helper()

	validator, err := protovalidate.New()
	require.NoError(t, err)

	lis := bufconn.Listen(bufSize)
	gs := grpc.NewServer(
		grpc.ChainUnaryInterceptor(server.RecoveryInterceptor, server.TraceInterceptor),
	)
	notificationv1.RegisterNotificationServiceServer(gs, server.New(svc, validator))

	go func() {
		_ = gs.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()
		gs.Stop()
	})

	return notificationv1.NewNotificationServiceClient(conn)
}

func TestReserveConfirmation(t *testing.T) {
	t.Run("valid request is reserved", func(t *testing.T) {
		svc := new(mockService)
		svc.On("ReserveConfirmation", mock.Anything, testSaga, "a@b.com",
			"http://localhost:8080/api/confirm/tok").Return(true, nil).Once()
		client := newTestClient(t, svc)

		resp, err := client.ReserveConfirmation(context.Background(), &notificationv1.ReserveConfirmationRequest{
			SagaId:     testSaga,
			Email:      "a@b.com",
			ConfirmUrl: "http://localhost:8080/api/confirm/tok",
		})

		require.NoError(t, err)
		assert.True(t, resp.GetOk())
		svc.AssertExpectations(t)
	})

	t.Run("invalid saga id is rejected before reaching service", func(t *testing.T) {
		svc := new(mockService)
		client := newTestClient(t, svc)

		_, err := client.ReserveConfirmation(context.Background(), &notificationv1.ReserveConfirmationRequest{
			SagaId:     "not-a-uuid",
			Email:      "a@b.com",
			ConfirmUrl: "http://localhost:8080/api/confirm/tok",
		})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		svc.AssertNotCalled(t, "ReserveConfirmation")
	})
}

func TestCommitConfirmation(t *testing.T) {
	t.Run("valid request commits", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, testSaga).Return(true, nil).Once()
		client := newTestClient(t, svc)

		resp, err := client.CommitConfirmation(context.Background(), &notificationv1.CommitConfirmationRequest{
			SagaId: testSaga,
		})

		require.NoError(t, err)
		assert.True(t, resp.GetOk())
		svc.AssertExpectations(t)
	})

	t.Run("service error maps to Internal without leaking details", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, mock.Anything).
			Return(false, errors.New("smtp exploded with secret token abc123")).Once()
		client := newTestClient(t, svc)

		_, err := client.CommitConfirmation(context.Background(), &notificationv1.CommitConfirmationRequest{
			SagaId: testSaga,
		})

		require.Equal(t, codes.Internal, status.Code(err))
		st, _ := status.FromError(err)
		assert.Equal(t, "failed to commit confirmation", st.Message())
	})

	t.Run("missing reservation maps to FailedPrecondition", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, mock.Anything).
			Return(false, delivery.ErrNotReserved).Once()
		client := newTestClient(t, svc)

		_, err := client.CommitConfirmation(context.Background(), &notificationv1.CommitConfirmationRequest{
			SagaId: testSaga,
		})

		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	})

	t.Run("canceled reservation maps to Aborted", func(t *testing.T) {
		svc := new(mockService)
		svc.On("CommitConfirmation", mock.Anything, mock.Anything).
			Return(false, delivery.ErrCanceled).Once()
		client := newTestClient(t, svc)

		_, err := client.CommitConfirmation(context.Background(), &notificationv1.CommitConfirmationRequest{
			SagaId: testSaga,
		})

		assert.Equal(t, codes.Aborted, status.Code(err))
	})
}

func TestCancelConfirmation(t *testing.T) {
	svc := new(mockService)
	svc.On("CancelConfirmation", mock.Anything, testSaga).Return(nil).Once()
	client := newTestClient(t, svc)

	resp, err := client.CancelConfirmation(context.Background(), &notificationv1.CancelConfirmationRequest{
		SagaId: testSaga,
	})

	require.NoError(t, err)
	assert.True(t, resp.GetOk())
	svc.AssertExpectations(t)
}
