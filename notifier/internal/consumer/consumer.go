package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/contract"
)

const prefetchCount = 10

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	handler *Handler
}

func New(url string, svc ReleaseSender) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open channel: %w", err), conn.Close())
	}

	if err := contract.DeclareReleaseTopology(ch); err != nil {
		return nil, errors.Join(err, ch.Close(), conn.Close())
	}

	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		return nil, errors.Join(fmt.Errorf("set qos: %w", err), ch.Close(), conn.Close())
	}

	return &Consumer{conn: conn, channel: ch, handler: NewHandler(svc)}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.Consume(contract.ReleaseQueue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}

	slog.Info("notifier consumer started", "queue", contract.ReleaseQueue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			c.process(ctx, d)
		}
	}
}

func (c *Consumer) process(ctx context.Context, d amqp.Delivery) {
	err := c.handler.Handle(ctx, d.Body)

	switch {
	case err == nil:
		if ackErr := d.Ack(false); ackErr != nil {
			slog.Error("consumer ack failed", "error", ackErr)
		}
	case errors.Is(err, ErrDrop):
		slog.Warn("consumer dropping message", "error", err)
		if rejErr := d.Reject(false); rejErr != nil {
			slog.Error("consumer reject failed", "error", rejErr)
		}
	default:
		slog.Error("consumer requeueing message", "error", err)
		if nackErr := d.Nack(false, true); nackErr != nil {
			slog.Error("consumer nack failed", "error", nackErr)
		}
	}
}

func (c *Consumer) Close() error {
	return errors.Join(c.channel.Close(), c.conn.Close())
}
