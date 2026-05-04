package scanner

import (
	"context"
	"errors"
	"log/slog"
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
	slog.Info("scanner: checking for new releases")
	subs, err := s.subRepo.GetAllConfirmedActive(ctx)
	if err != nil {
		slog.Error("scanner: get active subscriptions error", "error", err)
		return
	}

	if len(subs) == 0 {
		slog.Info("scanner: no confirmed active subscriptions found")
		return
	}

	grouped := groupByRepoID(subs)

	for repoID, repoSubs := range grouped {
		repo, err := s.repoRepo.GetByID(ctx, repoID)
		if err != nil {
			slog.Error("scanner: get repo by id error", "repo_id", repoID, "error", err)
			continue
		}
		if repo == nil {
			slog.Warn("scanner: repo not found", "repo_id", repoID)
			continue
		}

		tag, releaseURL, err := s.ghClient.GetLatestRelease(ctx, repo.Owner, repo.Name)
		if err != nil {
			switch {
			case errors.Is(err, github.ErrNoReleases):
				slog.Info("scanner: repo has no releases", "repo", repo.FullName)
			case errors.Is(err, github.ErrRateLimited):
				slog.Warn("scanner: rate limited while checking repo", "repo", repo.FullName)
			default:
				slog.Error("scanner: get latest release error", "repo", repo.FullName, "error", err)
			}
			continue
		}

		if repo.LastSeenTag == nil {
			if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
				slog.Error("scanner: update baseline tag error", "repo", repo.FullName, "error", err)
			} else {
				slog.Info("scanner: baseline tag set", "repo", repo.FullName, "tag", tag)
			}
			continue
		}

		if *repo.LastSeenTag == tag {
			slog.Info("scanner: no new release", "repo", repo.FullName)
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
				slog.Error("scanner: send release email failed", "email", sub.Email, "error", err)
				continue
			}
		}

		if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
			slog.Error("scanner: update last_seen_tag error", "repo", repo.FullName, "error", err)
			continue
		}

		slog.Info("scanner: new release processed", "repo", repo.FullName, "tag", tag)
	}
}

func groupByRepoID(subs []model.Subscription) map[int64][]model.Subscription {
	grouped := make(map[int64][]model.Subscription)

	for _, sub := range subs {
		grouped[sub.RepositoryID] = append(grouped[sub.RepositoryID], sub)
	}

	return grouped
}
