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

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/notifier/server"
)

const bufSize = 1024 * 1024

type mockService struct {
	mock.Mock
}

func (m *mockService) SendConfirm(ctx context.Context, email, confirmURL string) (bool, error) {
	args := m.Called(ctx, email, confirmURL)
	return args.Bool(0), args.Error(1)
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

func TestSendConfirm(t *testing.T) {
	t.Run("valid request is delivered", func(t *testing.T) {
		svc := new(mockService)
		svc.On("SendConfirm", mock.Anything, "a@b.com", "http://localhost:8080/api/confirm/tok").
			Return(true, nil).Once()
		client := newTestClient(t, svc)

		resp, err := client.SendConfirm(context.Background(), &notificationv1.SendConfirmRequest{
			Email:      "a@b.com",
			ConfirmUrl: "http://localhost:8080/api/confirm/tok",
		})

		require.NoError(t, err)
		assert.True(t, resp.GetDelivered())
		svc.AssertExpectations(t)
	})

	t.Run("invalid email is rejected before reaching service", func(t *testing.T) {
		svc := new(mockService)
		client := newTestClient(t, svc)

		_, err := client.SendConfirm(context.Background(), &notificationv1.SendConfirmRequest{
			Email:      "not-an-email",
			ConfirmUrl: "http://localhost:8080/api/confirm/tok",
		})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		svc.AssertNotCalled(t, "SendConfirm")
	})

	t.Run("service error maps to Internal without leaking details", func(t *testing.T) {
		svc := new(mockService)
		svc.On("SendConfirm", mock.Anything, mock.Anything, mock.Anything).
			Return(false, errors.New("smtp exploded with secret token abc123")).Once()
		client := newTestClient(t, svc)

		_, err := client.SendConfirm(context.Background(), &notificationv1.SendConfirmRequest{
			Email:      "a@b.com",
			ConfirmUrl: "http://localhost:8080/api/confirm/tok",
		})

		require.Equal(t, codes.Internal, status.Code(err))
		st, _ := status.FromError(err)
		assert.Equal(t, "failed to send notification", st.Message())
	})
}
