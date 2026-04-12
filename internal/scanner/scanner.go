package scanner

import (
	"context"
	"log"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
)

type Scanner struct {
	subRepo  repository.SubscriptionRepository
	repoRepo repository.GitHubRepository
	ghClient github.Client
	mailer   mailer.Mailer
	interval time.Duration
}

func New(
	subRepo repository.SubscriptionRepository,
	repoRepo repository.GitHubRepository,
	ghClient github.Client,
	mailer mailer.Mailer,
	interval time.Duration,
) *Scanner {
	return &Scanner{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   mailer,
		interval: interval,
	}
}

func (s *Scanner) Start(ctx context.Context) {
	go func() {
		s.run(ctx)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.run(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scanner) run(ctx context.Context) {
	log.Println("scanner: checking for new releases...")

	subs, err := s.subRepo.GetAllConfirmedActive(ctx)
	if err != nil {
		log.Println("scanner error:", err)
		return
	}

	log.Printf("scanner: found %d active subscriptions\n", len(subs))

}
