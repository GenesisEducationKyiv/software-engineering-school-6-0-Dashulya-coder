package scanner

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
)

type Scanner struct {
	subRepo  repository.SubscriptionRepository
	repoRepo repository.GitHubRepository
	ghClient github.Client
	mailer   mailer.Mailer
	interval time.Duration
	baseURL  string
}

func New(
	subRepo repository.SubscriptionRepository,
	repoRepo repository.GitHubRepository,
	ghClient github.Client,
	m mailer.Mailer,
	interval time.Duration,
	baseURL string,
) *Scanner {
	return &Scanner{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   m,
		interval: interval,
		baseURL:  baseURL,
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
		log.Println("scanner: get active subscriptions error:", err)
		return
	}

	if len(subs) == 0 {
		log.Println("scanner: no confirmed active subscriptions found")
		return
	}

	grouped := groupByRepoID(subs)

	for repoID, repoSubs := range grouped {
		repo, err := s.repoRepo.GetByID(ctx, repoID)
		if err != nil {
			log.Printf("scanner: get repo by id %d error: %v\n", repoID, err)
			continue
		}
		if repo == nil {
			log.Printf("scanner: repo with id %d not found\n", repoID)
			continue
		}

		tag, releaseURL, err := s.ghClient.GetLatestRelease(ctx, repo.Owner, repo.Name)
		if err != nil {
			switch {
			case errors.Is(err, github.ErrNoReleases):
				log.Printf("scanner: repo %s has no releases\n", repo.FullName)
			case errors.Is(err, github.ErrRateLimited):
				log.Printf("scanner: rate limited while checking repo %s\n", repo.FullName)
			default:
				log.Printf("scanner: get latest release for %s error: %v\n", repo.FullName, err)
			}
			continue
		}

		if repo.LastSeenTag == nil {
			if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
				log.Printf("scanner: update baseline tag for %s error: %v\n", repo.FullName, err)
			} else {
				log.Printf("scanner: baseline tag set for %s -> %s\n", repo.FullName, tag)
			}
			continue
		}

		if *repo.LastSeenTag == tag {
			log.Printf("scanner: no new release for %s\n", repo.FullName)
			continue
		}

		for _, sub := range repoSubs {
			unsubscribeLink := s.baseURL + "/api/unsubscribe/" + sub.UnsubscribeToken

			if err := s.mailer.SendNewRelease(
				sub.Email,
				repo.FullName,
				tag,
				releaseURL,
				unsubscribeLink,
			); err != nil {
				log.Printf("scanner: send release email to %s failed: %v\n", sub.Email, err)
				continue
			}
		}

		if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
			log.Printf("scanner: update last_seen_tag for %s error: %v\n", repo.FullName, err)
			continue
		}

		log.Printf("scanner: new release processed for %s -> %s\n", repo.FullName, tag)
	}
}

func groupByRepoID(subs []model.Subscription) map[int64][]model.Subscription {
	grouped := make(map[int64][]model.Subscription)

	for _, sub := range subs {
		grouped[sub.RepositoryID] = append(grouped[sub.RepositoryID], sub)
	}

	return grouped
}
