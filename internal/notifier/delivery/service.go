package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Ledger interface {
	Reserve(ctx context.Context, dedupKey string) (bool, error)
}

type Sender interface {
	SendConfirm(email, confirmURL string) error
	SendRelease(email, repoFullName, tag, releaseURL, unsubscribeURL string) error
}

type Service struct {
	ledger Ledger
	sender Sender
}

func New(ledger Ledger, sender Sender) *Service {
	return &Service{ledger: ledger, sender: sender}
}

func (s *Service) SendConfirm(ctx context.Context, email, confirmURL string) (bool, error) {
	key := dedupKey("confirm", email, confirmURL)

	reserved, err := s.ledger.Reserve(ctx, key)
	if err != nil {
		return false, err
	}
	if !reserved {
		return false, nil
	}

	if err := s.sender.SendConfirm(email, confirmURL); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) SendRelease(
	ctx context.Context,
	email, repoFullName, tag, releaseURL, unsubscribeURL string,
) (bool, error) {
	key := dedupKey("release", email, repoFullName, tag)

	reserved, err := s.ledger.Reserve(ctx, key)
	if err != nil {
		return false, err
	}
	if !reserved {
		return false, nil
	}

	if err := s.sender.SendRelease(email, repoFullName, tag, releaseURL, unsubscribeURL); err != nil {
		return false, err
	}

	return true, nil
}

func dedupKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
