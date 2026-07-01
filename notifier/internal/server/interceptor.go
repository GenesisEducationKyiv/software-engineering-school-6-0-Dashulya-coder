package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"regexp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey string

const (
	traceIDKey   ctxKey = "trace_id"
	traceMetaKey string = "x-trace-id"
	traceIDBytes int    = 16
)

var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TraceInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	id := traceIDFromMetadata(ctx)
	if id == "" {
		id = newTraceID()
	}

	ctx = context.WithValue(ctx, traceIDKey, id)
	slog.Info("grpc request", "method", info.FullMethod, "trace_id", id)

	return handler(ctx, req)
}

func RecoveryInterceptor(
	ctx context.Context,
	req any,
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("grpc panic recovered", "trace_id", traceID(ctx), "panic", r)
			err = status.Error(codes.Internal, "internal error")
		}
	}()

	return handler(ctx, req)
}

func traceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

func traceIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(traceMetaKey)
	if len(values) == 0 {
		return ""
	}

	if !traceIDPattern.MatchString(values[0]) {
		return ""
	}

	return values[0]
}

func newTraceID() string {
	buf := make([]byte, traceIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}
