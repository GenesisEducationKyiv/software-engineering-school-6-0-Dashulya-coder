package publisher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/contract"
)

const publishTimeout = 5 * time.Second

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.Mutex
}

func New(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open channel: %w", err), conn.Close())
	}

	if err := ch.Confirm(false); err != nil {
		return nil, errors.Join(fmt.Errorf("enable publisher confirms: %w", err), ch.Close(), conn.Close())
	}

	if err := contract.DeclareReleaseTopology(ch); err != nil {
		return nil, errors.Join(err, ch.Close(), conn.Close())
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	body, err := contract.ReleaseCommand{
		Email:          email,
		RepoFullName:   repo,
		Tag:            tag,
		ReleaseURL:     releaseURL,
		UnsubscribeURL: unsubscribeLink,
	}.Marshal()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	conf, err := p.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		contract.ReleaseExchange,
		contract.ReleaseRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish release command: %w", err)
	}

	if !conf.Wait() {
		return errors.New("release command not acknowledged by broker")
	}

	return nil
}

func (p *Publisher) Close() error {
	return errors.Join(p.channel.Close(), p.conn.Close())
}

var _ mailer.ReleaseSender = (*Publisher)(nil)
