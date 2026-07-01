package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

type Ledger interface {
	Reserve(ctx context.Context, dedupKey string) (bool, error)
}

type Sender interface {
	SendConfirm(ctx context.Context, email, confirmURL string) error
	SendRelease(ctx context.Context, email, repoFullName, tag, releaseURL, unsubscribeURL string) error
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

	if err := s.sender.SendConfirm(ctx, email, confirmURL); err != nil {
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

	if err := s.sender.SendRelease(ctx, email, repoFullName, tag, releaseURL, unsubscribeURL); err != nil {
		return false, err
	}

	return true, nil
}

func dedupKey(parts ...string) string {
	h := sha256.New()
	var lenBuf [8]byte
	for _, p := range parts {
		binary.LittleEndian.PutUint64(lenBuf[:], uint64(len(p)))
		h.Write(lenBuf[:])
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}
