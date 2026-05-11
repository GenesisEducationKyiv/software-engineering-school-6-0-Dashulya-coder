package scanner

import (
	"context"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/service"
)

type Scanner struct {
	service  service.ReleaseScanner
	interval time.Duration
}

func New(svc service.ReleaseScanner, interval time.Duration) *Scanner {
	return &Scanner{
		service:  svc,
		interval: interval,
	}
}

func (s *Scanner) Start(ctx context.Context) {
	go func() {
		s.service.CheckReleases(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.service.CheckReleases(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}
