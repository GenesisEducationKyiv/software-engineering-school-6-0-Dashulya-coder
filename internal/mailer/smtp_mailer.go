package mailer

import (
	"fmt"
	"net/smtp"

	appmetrics "github.com/Dashulya-coder/CaseTaskNotifier/internal/metrics"
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

	err := m.send(email, body)
	recordEmail("confirmation", err)
	return err
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

	err := m.send(email, body)
	recordEmail("release", err)
	return err
}

func recordEmail(kind string, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	appmetrics.EmailsSentTotal.WithLabelValues(kind, status).Inc()
}

func (m *SMTPMailer) send(to, msg string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	var auth smtp.Auth
	if m.user != "" && m.pass != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
