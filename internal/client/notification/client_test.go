package notification_test

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/client/notification"
)

type mockServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	mock.Mock
}

func (m *mockServer) SendConfirm(
	ctx context.Context,
	req *notificationv1.SendConfirmRequest,
) (*notificationv1.SendResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notificationv1.SendResponse), args.Error(1)
}

func (m *mockServer) SendRelease(
	ctx context.Context,
	req *notificationv1.SendReleaseRequest,
) (*notificationv1.SendResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notificationv1.SendResponse), args.Error(1)
}

func newTestClient(t *testing.T, srv *mockServer) *notification.Client {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	gs := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(gs, srv)
	go func() {
		_ = gs.Serve(lis)
	}()

	client, err := notification.New(lis.Addr().String())
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
		gs.Stop()
	})

	return client
}

func TestSendConfirmation(t *testing.T) {
	srv := new(mockServer)
	var got *notificationv1.SendConfirmRequest
	srv.On("SendConfirm", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			got = args.Get(1).(*notificationv1.SendConfirmRequest)
		}).
		Return(&notificationv1.SendResponse{Delivered: true}, nil).Once()

	client := newTestClient(t, srv)

	require.NoError(t, client.SendConfirmation("a@b.com", "http://x/confirm/tok"))
	assert.Equal(t, "a@b.com", got.GetEmail())
	assert.Equal(t, "http://x/confirm/tok", got.GetConfirmUrl())
	srv.AssertExpectations(t)
}

func TestSendNewRelease(t *testing.T) {
	srv := new(mockServer)
	var got *notificationv1.SendReleaseRequest
	srv.On("SendRelease", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			got = args.Get(1).(*notificationv1.SendReleaseRequest)
		}).
		Return(&notificationv1.SendResponse{Delivered: true}, nil).Once()

	client := newTestClient(t, srv)

	err := client.SendNewRelease("a@b.com", "owner/repo", "v1.2", "http://x/rel", "http://x/unsub")
	require.NoError(t, err)
	assert.Equal(t, "a@b.com", got.GetEmail())
	assert.Equal(t, "owner/repo", got.GetRepoFullName())
	assert.Equal(t, "v1.2", got.GetTag())
	assert.Equal(t, "http://x/rel", got.GetReleaseUrl())
	assert.Equal(t, "http://x/unsub", got.GetUnsubscribeUrl())
	srv.AssertExpectations(t)
}

func TestSendConfirmation_ServerError(t *testing.T) {
	srv := new(mockServer)
	srv.On("SendConfirm", mock.Anything, mock.Anything).
		Return(nil, errors.New("backend failure")).Once()

	client := newTestClient(t, srv)

	require.Error(t, client.SendConfirmation("a@b.com", "http://x/confirm/tok"))
}
