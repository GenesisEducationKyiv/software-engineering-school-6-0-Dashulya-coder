package scanner

import (
	"context"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/release"
)

type Scanner struct {
	poller   release.Poller
	interval time.Duration
}

func New(p release.Poller, interval time.Duration) *Scanner {
	return &Scanner{
		poller:   p,
		interval: interval,
	}
}

func (s *Scanner) Start(ctx context.Context) {
	go func() {
		s.poller.Poll(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.poller.Poll(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}
