package contract

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ReleaseExchange   = "notifications"
	ReleaseRoutingKey = "release"
	ReleaseQueue      = "notifications.release"
)

type ReleaseCommand struct {
	Email          string `json:"email"`
	RepoFullName   string `json:"repo_full_name"`
	Tag            string `json:"tag"`
	ReleaseURL     string `json:"release_url"`
	UnsubscribeURL string `json:"unsubscribe_url"`
}

func (c ReleaseCommand) Marshal() ([]byte, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal release command: %w", err)
	}
	return body, nil
}

func UnmarshalReleaseCommand(data []byte) (ReleaseCommand, error) {
	var c ReleaseCommand
	if err := json.Unmarshal(data, &c); err != nil {
		return ReleaseCommand{}, fmt.Errorf("unmarshal release command: %w", err)
	}
	return c, nil
}

func DeclareReleaseTopology(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(ReleaseExchange, amqp.ExchangeDirect, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(ReleaseQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(ReleaseQueue, ReleaseRoutingKey, ReleaseExchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	return nil
}
