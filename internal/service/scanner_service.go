package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

type ReleaseScanner interface {
	CheckReleases(ctx context.Context)
}

type ReleaseScannerImpl struct {
	subRepo  repository.SubscriptionRepository
	repoRepo repository.GitHubRepository
	ghClient github.Client
	mailer   mailer.Mailer
	urls     urlbuilder.URLBuilder
}

func NewReleaseScanner(
	subRepo repository.SubscriptionRepository,
	repoRepo repository.GitHubRepository,
	ghClient github.Client,
	m mailer.Mailer,
	urls urlbuilder.URLBuilder,
) *ReleaseScannerImpl {
	return &ReleaseScannerImpl{
		subRepo:  subRepo,
		repoRepo: repoRepo,
		ghClient: ghClient,
		mailer:   m,
		urls:     urls,
	}
}

func (s *ReleaseScannerImpl) CheckReleases(ctx context.Context) {
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

	for repoID, repoSubs := range groupByRepoID(subs) {
		s.processRepo(ctx, repoID, repoSubs)
	}
}

func (s *ReleaseScannerImpl) processRepo(ctx context.Context, repoID int64, subs []model.Subscription) {
	repo, err := s.repoRepo.GetByID(ctx, repoID)
	if err != nil {
		slog.Error("scanner: get repo by id error", "repo_id", repoID, "error", err)
		return
	}
	if repo == nil {
		slog.Warn("scanner: repo not found", "repo_id", repoID)
		return
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
		return
	}

	if repo.LastSeenTag == nil {
		if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
			slog.Error("scanner: update baseline tag error", "repo", repo.FullName, "error", err)
		} else {
			slog.Info("scanner: baseline tag set", "repo", repo.FullName, "tag", tag)
		}
		return
	}

	if *repo.LastSeenTag == tag {
		slog.Info("scanner: no new release", "repo", repo.FullName)
		return
	}

	s.notifySubscribers(repo, subs, tag, releaseURL)

	if err := s.repoRepo.UpdateLastSeenTag(ctx, repo.ID, tag, releaseURL); err != nil {
		slog.Error("scanner: update last_seen_tag error", "repo", repo.FullName, "error", err)
		return
	}

	slog.Info("scanner: new release processed", "repo", repo.FullName, "tag", tag)
}

func (s *ReleaseScannerImpl) notifySubscribers(
	repo *model.GitHubRepository,
	subs []model.Subscription,
	tag, releaseURL string,
) {
	for _, sub := range subs {
		err := s.mailer.SendNewRelease(
			sub.Email,
			repo.FullName,
			tag,
			releaseURL,
			s.urls.UnsubscribeURL(sub.UnsubscribeToken),
		)
		if err != nil {
			slog.Error("scanner: send release email failed", "email", sub.Email, "error", err)
		}
	}
}

func groupByRepoID(subs []model.Subscription) map[int64][]model.Subscription {
	grouped := make(map[int64][]model.Subscription)
	for _, sub := range subs {
		grouped[sub.RepositoryID] = append(grouped[sub.RepositoryID], sub)
	}
	return grouped
}

var _ ReleaseScanner = (*ReleaseScannerImpl)(nil)
