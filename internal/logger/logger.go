package logger

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
)

func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type SamplingHandler struct {
	next    slog.Handler
	levels  map[slog.Level]struct{}
	rateN   uint64
	counter *atomic.Uint64
}

func NewSamplingHandler(next slog.Handler, rateN uint64, sampled ...slog.Level) *SamplingHandler {
	set := make(map[slog.Level]struct{}, len(sampled))
	for _, lvl := range sampled {
		set[lvl] = struct{}{}
	}
	return &SamplingHandler{
		next:    next,
		levels:  set,
		rateN:   rateN,
		counter: &atomic.Uint64{},
	}
}

func (h *SamplingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *SamplingHandler) Handle(ctx context.Context, r slog.Record) error {
	if _, ok := h.levels[r.Level]; ok && h.rateN > 1 {
		n := h.counter.Add(1)
		if n%h.rateN != 0 {
			return nil
		}
	}
	return h.next.Handle(ctx, r)
}

func (h *SamplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SamplingHandler{
		next:    h.next.WithAttrs(attrs),
		levels:  h.levels,
		rateN:   h.rateN,
		counter: h.counter,
	}
}

func (h *SamplingHandler) WithGroup(name string) slog.Handler {
	return &SamplingHandler{
		next:    h.next.WithGroup(name),
		levels:  h.levels,
		rateN:   h.rateN,
		counter: h.counter,
	}
}

var _ slog.Handler = (*SamplingHandler)(nil)
