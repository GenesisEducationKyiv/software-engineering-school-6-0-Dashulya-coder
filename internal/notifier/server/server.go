package server

import (
	"context"
	"log/slog"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	notificationv1 "github.com/Dashulya-coder/CaseTaskNotifier/gen/notification/v1"
)

type Service interface {
	SendConfirm(ctx context.Context, email, confirmURL string) (bool, error)
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

func (s *Server) SendConfirm(
	ctx context.Context,
	req *notificationv1.SendConfirmRequest,
) (*notificationv1.SendResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	delivered, err := s.svc.SendConfirm(ctx, req.GetEmail(), req.GetConfirmUrl())
	if err != nil {
		slog.Error("send confirm failed", "trace_id", traceID(ctx), "error", err)
		return nil, status.Error(codes.Internal, "failed to send notification")
	}

	return &notificationv1.SendResponse{Delivered: delivered}, nil
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
