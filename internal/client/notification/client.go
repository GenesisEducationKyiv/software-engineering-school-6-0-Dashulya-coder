package notification

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
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

func (c *Client) SendConfirmation(email, confirmLink string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	_, err := c.client.SendConfirm(ctx, &notificationv1.SendConfirmRequest{
		Email:      email,
		ConfirmUrl: confirmLink,
	})
	if err != nil {
		return fmt.Errorf("send confirmation: %w", err)
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

var _ mailer.Mailer = (*Client)(nil)
