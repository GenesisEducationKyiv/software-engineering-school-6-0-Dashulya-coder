package saga

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	compensationTimeout = 30 * time.Second
	pivotMaxAttempts    = 3
	pivotRetryDelay     = 200 * time.Millisecond
)

type Step struct {
	Name         string
	Action       func(ctx context.Context) error
	Compensation func(ctx context.Context) error
	Pivot        bool
}

func Run(ctx context.Context, steps ...Step) error {
	completed := make([]Step, 0, len(steps))

	for _, step := range steps {
		if step.Pivot {
			if err := runPivot(ctx, step); err != nil {
				return fmt.Errorf("saga pivot %q failed, not compensating: %w", step.Name, err)
			}
			completed = append(completed, step)
			continue
		}

		if err := step.Action(ctx); err != nil {
			slog.Error("saga step failed", "step", step.Name, "error", err)
			compensate(completed)
			return fmt.Errorf("saga step %q: %w", step.Name, err)
		}

		slog.Debug("saga step completed", "step", step.Name)
		completed = append(completed, step)
	}

	return nil
}

func runPivot(ctx context.Context, step Step) error {
	var err error

	for attempt := 1; attempt <= pivotMaxAttempts; attempt++ {
		if err = step.Action(ctx); err == nil {
			slog.Debug("saga pivot completed", "step", step.Name, "attempt", attempt)
			return nil
		}

		slog.Warn("saga pivot attempt failed", "step", step.Name, "attempt", attempt, "error", err)

		if attempt == pivotMaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pivotRetryDelay):
		}
	}

	return err
}

func compensate(completed []Step) {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
	defer cancel()

	for i := len(completed) - 1; i >= 0; i-- {
		step := completed[i]
		if step.Compensation == nil {
			continue
		}
		if err := step.Compensation(ctx); err != nil {
			slog.Error("saga compensation failed", "step", step.Name, "error", err)
			continue
		}
		slog.Info("saga step compensated", "step", step.Name)
	}
}
