package service

import (
	"context"
	"errors"
	"testing"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
)

func TestReleaseScanner_NoConfirmedActiveSubscriptions(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{}, nil
		},
	}

	ghClientCalled := false
	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			ghClientCalled = true
			return "", "", nil
		},
	}
	m := &mockMailer{}

	svc := NewReleaseScanner(subRepo, &mockGitHubRepository{}, ghClient, m, newTestURLs())
	svc.CheckReleases(context.Background())

	if ghClientCalled {
		t.Fatal("github client should not be called when there are no subscriptions")
	}
	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestReleaseScanner_SameTag_NoEmailSent(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 10, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 10, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _ string, _ string) error {
			t.Fatal("last seen tag should not be updated when tag did not change")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "v1.0.0", "https://example.com/release", nil
		},
	}

	m := &mockMailer{
		sendNewReleaseFn: func(_, _, _, _, _ string) error {
			t.Fatal("email should not be sent when tag did not change")
			return nil
		},
	}

	svc := NewReleaseScanner(subRepo, repoRepo, ghClient, m, newTestURLs())
	svc.CheckReleases(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestReleaseScanner_NewTag_SendsEmailAndUpdatesTag(t *testing.T) {
	lastSeen := "old-tag"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{ID: 1, Email: "a@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok1"},
				{ID: 2, Email: "b@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok2"},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 20, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, repoID int64, tag, releaseURL string) error {
			updated = true
			if repoID != 20 {
				t.Fatalf("unexpected repo id: %d", repoID)
			}
			if tag != "v2.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "v2.0.0", "https://example.com/v2.0.0", nil
		},
	}

	m := &mockMailer{
		sendNewReleaseFn: func(_, repo, tag, _, unsubscribeLink string) error {
			if repo != "cli/cli" {
				t.Fatalf("unexpected repo: %s", repo)
			}
			if tag != "v2.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			if unsubscribeLink == "" {
				t.Fatal("unsubscribe link should not be empty")
			}
			return nil
		},
	}

	svc := NewReleaseScanner(subRepo, repoRepo, ghClient, m, newTestURLs())
	svc.CheckReleases(context.Background())

	if m.sendNewReleaseCalls != 2 {
		t.Fatalf("expected 2 emails sent, got %d", m.sendNewReleaseCalls)
	}
	if !updated {
		t.Fatal("expected last seen tag to be updated")
	}
}

func TestReleaseScanner_FirstSeenRelease_SetsBaselineWithoutEmail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 30, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 30, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: nil}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, tag, _ string) error {
			updated = true
			if tag != "v3.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "v3.0.0", "https://example.com/v3.0.0", nil
		},
	}

	m := &mockMailer{
		sendNewReleaseFn: func(_, _, _, _, _ string) error {
			t.Fatal("email should not be sent when setting baseline")
			return nil
		},
	}

	svc := NewReleaseScanner(subRepo, repoRepo, ghClient, m, newTestURLs())
	svc.CheckReleases(context.Background())

	if !updated {
		t.Fatal("expected baseline tag to be set")
	}
	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestReleaseScanner_NoReleases_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 40, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 40, FullName: "golang/go", Owner: "golang", Name: "go", LastSeenTag: &lastSeen}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _, _ string) error {
			t.Fatal("should not update tag when no releases")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", gh.ErrNoReleases
		},
	}

	svc := NewReleaseScanner(subRepo, repoRepo, ghClient, &mockMailer{}, newTestURLs())
	svc.CheckReleases(context.Background())
}

func TestReleaseScanner_GitHubError_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 50, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 50, FullName: "cli/cli", Owner: "cli", Name: "cli"}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _, _ string) error {
			t.Fatal("should not update tag on github error")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("network error")
		},
	}

	m := &mockMailer{}
	svc := NewReleaseScanner(subRepo, repoRepo, ghClient, m, newTestURLs())
	svc.CheckReleases(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}
