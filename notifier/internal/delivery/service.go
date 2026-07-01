package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"
)

const (
	markSentMaxAttempts = 3
	markSentRetryDelay  = 100 * time.Millisecond
)

var (
	ErrNotReserved = errors.New("confirmation not reserved")
	ErrCanceled    = errors.New("confirmation already canceled")
)

type DeliveryStatus string

const (
	StatusPending  DeliveryStatus = "PENDING"
	StatusSent     DeliveryStatus = "SENT"
	StatusCanceled DeliveryStatus = "CANCELED"
)

type Delivery struct {
	SagaID     string
	Email      string
	ConfirmURL string
	Status     DeliveryStatus
}

type Ledger interface {
	Reserve(ctx context.Context, dedupKey string) (bool, error)
}

type Deliveries interface {
	Reserve(ctx context.Context, d Delivery) (bool, error)
	Get(ctx context.Context, sagaID string) (*Delivery, error)
	MarkSent(ctx context.Context, sagaID string) error
	Cancel(ctx context.Context, sagaID string) error
}

type Sender interface {
	SendConfirm(ctx context.Context, email, confirmURL string) error
	SendRelease(ctx context.Context, email, repoFullName, tag, releaseURL, unsubscribeURL string) error
}

type Service struct {
	ledger     Ledger
	deliveries Deliveries
	sender     Sender
}

func New(ledger Ledger, deliveries Deliveries, sender Sender) *Service {
	return &Service{ledger: ledger, deliveries: deliveries, sender: sender}
}

func (s *Service) ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) (bool, error) {
	return s.deliveries.Reserve(ctx, Delivery{
		SagaID:     sagaID,
		Email:      email,
		ConfirmURL: confirmURL,
		Status:     StatusPending,
	})
}

func (s *Service) CommitConfirmation(ctx context.Context, sagaID string) (bool, error) {
	d, err := s.deliveries.Get(ctx, sagaID)
	if err != nil {
		return false, err
	}
	if d == nil {
		return false, ErrNotReserved
	}

	switch d.Status {
	case StatusSent:
		return true, nil
	case StatusCanceled:
		return false, ErrCanceled
	}

	if err := s.sender.SendConfirm(ctx, d.Email, d.ConfirmURL); err != nil {
		return false, err
	}

	if err := s.markSent(ctx, sagaID); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) markSent(ctx context.Context, sagaID string) error {
	var err error

	for attempt := 1; attempt <= markSentMaxAttempts; attempt++ {
		if err = s.deliveries.MarkSent(ctx, sagaID); err == nil {
			return nil
		}

		if attempt == markSentMaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(markSentRetryDelay):
		}
	}

	return err
}

func (s *Service) CancelConfirmation(ctx context.Context, sagaID string) error {
	return s.deliveries.Cancel(ctx, sagaID)
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
