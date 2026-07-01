package server

import (
	"context"
	"errors"
	"log/slog"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/notifier/gen/notification/v1"
	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
)

type Service interface {
	ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) (bool, error)
	CommitConfirmation(ctx context.Context, sagaID string) (bool, error)
	CancelConfirmation(ctx context.Context, sagaID string) error
	SendRelease(
		ctx context.Context,
		email, repoFullName, tag, releaseURL, unsubscribeURL string,
	) (bool, error)
}

type Server struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc       Service
	validator protovalidate.Validator
}

func New(svc Service, validator protovalidate.Validator) *Server {
	return &Server{svc: svc, validator: validator}
}

func (s *Server) ReserveConfirmation(
	ctx context.Context,
	req *notificationv1.ReserveConfirmationRequest,
) (*notificationv1.ConfirmationResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ok, err := s.svc.ReserveConfirmation(ctx, req.GetSagaId(), req.GetEmail(), req.GetConfirmUrl())
	if err != nil {
		slog.Error("reserve confirmation failed", "trace_id", traceID(ctx), "error", err)
		return nil, status.Error(codes.Internal, "failed to reserve confirmation")
	}

	return &notificationv1.ConfirmationResponse{Ok: ok}, nil
}

func (s *Server) CommitConfirmation(
	ctx context.Context,
	req *notificationv1.CommitConfirmationRequest,
) (*notificationv1.ConfirmationResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ok, err := s.svc.CommitConfirmation(ctx, req.GetSagaId())
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrNotReserved):
			return nil, status.Error(codes.FailedPrecondition, "confirmation not reserved")
		case errors.Is(err, delivery.ErrCanceled):
			return nil, status.Error(codes.Aborted, "confirmation already canceled")
		default:
			slog.Error("commit confirmation failed", "trace_id", traceID(ctx), "error", err)
			return nil, status.Error(codes.Internal, "failed to commit confirmation")
		}
	}

	return &notificationv1.ConfirmationResponse{Ok: ok}, nil
}

func (s *Server) CancelConfirmation(
	ctx context.Context,
	req *notificationv1.CancelConfirmationRequest,
) (*notificationv1.ConfirmationResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if err := s.svc.CancelConfirmation(ctx, req.GetSagaId()); err != nil {
		slog.Error("cancel confirmation failed", "trace_id", traceID(ctx), "error", err)
		return nil, status.Error(codes.Internal, "failed to cancel confirmation")
	}

	return &notificationv1.ConfirmationResponse{Ok: true}, nil
}

func (s *Server) SendRelease(
	ctx context.Context,
	req *notificationv1.SendReleaseRequest,
) (*notificationv1.SendResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delivered, err := s.svc.SendRelease(
		ctx,
		req.GetEmail(),
		req.GetRepoFullName(),
		req.GetTag(),
		req.GetReleaseUrl(),
		req.GetUnsubscribeUrl(),
	)
	if err != nil {
		slog.Error("send release failed", "trace_id", traceID(ctx), "error", err)
		return nil, status.Error(codes.Internal, "failed to send notification")
	}

	return &notificationv1.SendResponse{Delivered: delivered}, nil
}
