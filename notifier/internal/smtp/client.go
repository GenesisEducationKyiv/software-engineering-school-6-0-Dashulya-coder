package smtp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/metrics"
)

type Client struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewClient(host string, port int, user, pass, from string) *Client {
	return &Client{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

func (c *Client) SendConfirm(ctx context.Context, email, confirmURL string) error {
	body := fmt.Sprintf(
		"Subject: %s\r\n\r\nPlease confirm your subscription:\n%s",
		"Confirm your subscription",
		confirmURL,
	)

	err := c.send(ctx, email, body)
	metrics.RecordEmail("confirmation", err)
	return err
}

func (c *Client) SendRelease(
	ctx context.Context,
	email, repoFullName, tag, releaseURL, unsubscribeURL string,
) error {
	const format = "Subject: New release for %s\r\n\r\n" +
		"New release detected for %s\nTag: %s\nRelease: %s\nUnsubscribe: %s"

	body := fmt.Sprintf(format, repoFullName, repoFullName, tag, releaseURL, unsubscribeURL)

	err := c.send(ctx, email, body)
	metrics.RecordEmail("release", err)
	return err
}

func (c *Client) send(ctx context.Context, to, msg string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return errors.Join(fmt.Errorf("set smtp deadline: %w", err), conn.Close())
		}
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return errors.Join(fmt.Errorf("create smtp client: %w", err), conn.Close())
	}

	if err := c.deliver(client, to, msg); err != nil {
		return errors.Join(err, client.Close())
	}

	return client.Quit()
}

func (c *Client) deliver(client *smtp.Client, to, msg string) error {
	if c.user != "" && c.pass != "" {
		if err := client.Auth(smtp.PlainAuth("", c.user, c.pass, c.host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(c.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return nil
}
