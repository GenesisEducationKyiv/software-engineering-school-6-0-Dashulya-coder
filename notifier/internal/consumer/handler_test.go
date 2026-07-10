package consumer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/contract"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/consumer"
)

var errSend = errors.New("smtp down")

type fakeSender struct {
	called    bool
	gotEmail  string
	gotTag    string
	delivered bool
	err       error
}

func (f *fakeSender) SendRelease(
	_ context.Context,
	email, _, tag, _, _ string,
) (bool, error) {
	f.called = true
	f.gotEmail = email
	f.gotTag = tag
	return f.delivered, f.err
}

func validBody(t *testing.T) []byte {
	t.Helper()
	body, err := contract.ReleaseCommand{
		Email:          "a@b.com",
		RepoFullName:   "owner/repo",
		Tag:            "v1.2.3",
		ReleaseURL:     "https://example.com/r",
		UnsubscribeURL: "https://example.com/u",
	}.Marshal()
	require.NoError(t, err)
	return body
}

func TestHandle(t *testing.T) {
	t.Run("valid command is delivered and acked", func(t *testing.T) {
		sender := &fakeSender{delivered: true}
		h := consumer.NewHandler(sender)

		err := h.Handle(context.Background(), validBody(t))

		require.NoError(t, err)
		assert.True(t, sender.called)
		assert.Equal(t, "a@b.com", sender.gotEmail)
		assert.Equal(t, "v1.2.3", sender.gotTag)
	})

	t.Run("duplicate (not delivered) is acked without error", func(t *testing.T) {
		sender := &fakeSender{delivered: false}
		h := consumer.NewHandler(sender)

		err := h.Handle(context.Background(), validBody(t))

		require.NoError(t, err)
		assert.True(t, sender.called)
	})

	t.Run("malformed json is dropped and never sent", func(t *testing.T) {
		sender := &fakeSender{}
		h := consumer.NewHandler(sender)

		err := h.Handle(context.Background(), []byte("{not json"))

		require.ErrorIs(t, err, consumer.ErrDrop)
		assert.False(t, sender.called)
	})

	t.Run("missing required fields is dropped and never sent", func(t *testing.T) {
		sender := &fakeSender{}
		h := consumer.NewHandler(sender)

		body, err := contract.ReleaseCommand{Email: "", RepoFullName: "owner/repo", Tag: "v1"}.Marshal()
		require.NoError(t, err)

		err = h.Handle(context.Background(), body)

		require.ErrorIs(t, err, consumer.ErrDrop)
		assert.False(t, sender.called)
	})

	t.Run("sender error is requeued (not dropped)", func(t *testing.T) {
		sender := &fakeSender{err: errSend}
		h := consumer.NewHandler(sender)

		err := h.Handle(context.Background(), validBody(t))

		require.Error(t, err)
		assert.NotErrorIs(t, err, consumer.ErrDrop)
		assert.ErrorIs(t, err, errSend)
		assert.True(t, sender.called)
	})
}
