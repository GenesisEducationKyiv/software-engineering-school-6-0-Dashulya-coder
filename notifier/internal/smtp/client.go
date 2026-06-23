package smtp

import (
	"fmt"
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

func (c *Client) SendConfirm(email, confirmURL string) error {
	body := fmt.Sprintf(
		"Subject: %s\r\n\r\nPlease confirm your subscription:\n%s",
		"Confirm your subscription",
		confirmURL,
	)

	err := c.send(email, body)
	metrics.RecordEmail("confirmation", err)
	return err
}

func (c *Client) SendRelease(email, repoFullName, tag, releaseURL, unsubscribeURL string) error {
	const format = "Subject: New release for %s\r\n\r\n" +
		"New release detected for %s\nTag: %s\nRelease: %s\nUnsubscribe: %s"

	body := fmt.Sprintf(format, repoFullName, repoFullName, tag, releaseURL, unsubscribeURL)

	err := c.send(email, body)
	metrics.RecordEmail("release", err)
	return err
}

func (c *Client) send(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var auth smtp.Auth
	if c.user != "" && c.pass != "" {
		auth = smtp.PlainAuth("", c.user, c.pass, c.host)
	}

	if err := smtp.SendMail(addr, auth, c.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	return nil
}
