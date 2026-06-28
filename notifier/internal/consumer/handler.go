package consumer

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/contract"
)

// ErrDrop marks a message that must not be redelivered (malformed or invalid).
var ErrDrop = errors.New("drop message")

type ReleaseSender interface {
	SendRelease(
		ctx context.Context,
		email, repoFullName, tag, releaseURL, unsubscribeURL string,
	) (bool, error)
}

type Handler struct {
	svc ReleaseSender
}

func NewHandler(svc ReleaseSender) *Handler {
	return &Handler{svc: svc}
}

// Handle decodes and dispatches one message. A nil return means ack; an error
// wrapping ErrDrop means reject without requeue; any other error means the
// message should be requeued for retry.
func (h *Handler) Handle(ctx context.Context, body []byte) error {
	cmd, err := contract.UnmarshalReleaseCommand(body)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDrop, err)
	}

	if cmd.Email == "" || cmd.RepoFullName == "" || cmd.Tag == "" {
		return fmt.Errorf("%w: missing required fields", ErrDrop)
	}

	if _, err := h.svc.SendRelease(
		ctx, cmd.Email, cmd.RepoFullName, cmd.Tag, cmd.ReleaseURL, cmd.UnsubscribeURL,
	); err != nil {
		return fmt.Errorf("send release: %w", err)
	}

	return nil
}
