package mailer

import (
	"fmt"
	"net/smtp"
)

type SMTPMailer struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewSMTPMailer(host string, port int, user, pass string) *SMTPMailer {
	from := user
	if from == "" {
		from = "no-reply@example.com"
	}

	return &SMTPMailer{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

func (m *SMTPMailer) SendConfirmation(email, confirmLink string) error {
	subject := "Confirm your subscription"
	body := fmt.Sprintf(
		"Subject: %s\r\n\r\nPlease confirm your subscription:\n%s",
		subject,
		confirmLink,
	)

	return m.send(email, body)
}

func (m *SMTPMailer) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	subject := fmt.Sprintf("New release for %s", repo)
	body := fmt.Sprintf(
		"Subject: %s\r\n\r\nNew release detected for %s\nTag: %s\nRelease: %s\nUnsubscribe: %s",
		subject,
		repo,
		tag,
		releaseURL,
		unsubscribeLink,
	)

	return m.send(email, body)
}

func (m *SMTPMailer) send(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var auth smtp.Auth
	if m.user != "" && m.pass != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
