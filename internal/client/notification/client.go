package notification

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/notifier/gen/notification/v1"
)

const rpcTimeout = 30 * time.Second

type Client struct {
	conn   *grpc.ClientConn
	client notificationv1.NotificationServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial notifier: %w", err)
	}

	return &Client{
		conn:   conn,
		client: notificationv1.NewNotificationServiceClient(conn),
	}, nil
}

func (c *Client) ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	_, err := c.client.ReserveConfirmation(ctx, &notificationv1.ReserveConfirmationRequest{
		SagaId:     sagaID,
		Email:      email,
		ConfirmUrl: confirmURL,
	})
	if err != nil {
		return fmt.Errorf("reserve confirmation: %w", err)
	}

	return nil
}

func (c *Client) CommitConfirmation(ctx context.Context, sagaID string) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	_, err := c.client.CommitConfirmation(ctx, &notificationv1.CommitConfirmationRequest{
		SagaId: sagaID,
	})
	if err != nil {
		return fmt.Errorf("commit confirmation: %w", err)
	}

	return nil
}

func (c *Client) CancelConfirmation(ctx context.Context, sagaID string) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	_, err := c.client.CancelConfirmation(ctx, &notificationv1.CancelConfirmationRequest{
		SagaId: sagaID,
	})
	if err != nil {
		return fmt.Errorf("cancel confirmation: %w", err)
	}

	return nil
}

func (c *Client) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	_, err := c.client.SendRelease(ctx, &notificationv1.SendReleaseRequest{
		Email:          email,
		RepoFullName:   repo,
		Tag:            tag,
		ReleaseUrl:     releaseURL,
		UnsubscribeUrl: unsubscribeLink,
	})
	if err != nil {
		return fmt.Errorf("send new release: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close notifier conn: %w", err)
	}
	return nil
}

var _ mailer.ReleaseSender = (*Client)(nil)
