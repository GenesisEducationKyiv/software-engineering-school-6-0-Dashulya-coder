package release

import (
	"context"
	"errors"
	"testing"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

type mockSubscriptionRepository struct {
	getAllConfirmedActiveFn     func(ctx context.Context) ([]subscription.Subscription, error)
	getConfirmedActiveByRepoFn func(ctx context.Context, repoID int64) ([]subscription.Subscription, error)
}

func (m *mockSubscriptionRepository) Create(_ context.Context, _ *subscription.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepository) FindByConfirmToken(_ context.Context, _ string) (*subscription.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(_ context.Context, _ string) (*subscription.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) GetByEmail(_ context.Context, _ string) ([]subscription.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) ExistsByEmailAndRepo(_ context.Context, _ string, _ int64) (bool, error) {
	return false, nil
}

func (m *mockSubscriptionRepository) ConfirmByToken(_ context.Context, _ string) error {
	return nil
}

func (m *mockSubscriptionRepository) DeactivateByToken(_ context.Context, _ string) error {
	return nil
}

func (m *mockSubscriptionRepository) GetAllConfirmedActive(ctx context.Context) ([]subscription.Subscription, error) {
	if m.getAllConfirmedActiveFn != nil {
		return m.getAllConfirmedActiveFn(ctx)
	}
	return nil, nil
}

func (m *mockSubscriptionRepository) GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]subscription.Subscription, error) {
	if m.getConfirmedActiveByRepoFn != nil {
		return m.getConfirmedActiveByRepoFn(ctx, repoID)
	}
	return nil, nil
}

type mockGitHubRepository struct {
	getByIDFn           func(ctx context.Context, id int64) (*repo.Repository, error)
	updateLastSeenTagFn func(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

func (m *mockGitHubRepository) FindOrCreate(_ context.Context, _, _, _ string) (*repo.Repository, error) {
	return nil, nil
}

func (m *mockGitHubRepository) FindByFullName(_ context.Context, _ string) (*repo.Repository, error) {
	return nil, nil
}

func (m *mockGitHubRepository) GetByID(ctx context.Context, id int64) (*repo.Repository, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockGitHubRepository) UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error {
	if m.updateLastSeenTagFn != nil {
		return m.updateLastSeenTagFn(ctx, repoID, tag, releaseURL)
	}
	return nil
}

type mockGitHubClient struct {
	getLatestReleaseFn func(ctx context.Context, owner, repo string) (string, string, error)
}

func (m *mockGitHubClient) RepositoryExists(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (string, string, error) {
	if m.getLatestReleaseFn != nil {
		return m.getLatestReleaseFn(ctx, owner, repo)
	}
	return "", "", nil
}

type mockMailer struct {
	sendNewReleaseCalls int
	sendNewReleaseFn    func(email, repo, tag, releaseURL, unsubscribeLink string) error
}

func (m *mockMailer) SendConfirmation(_, _ string) error {
	return nil
}

func (m *mockMailer) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	m.sendNewReleaseCalls++
	if m.sendNewReleaseFn != nil {
		return m.sendNewReleaseFn(email, repo, tag, releaseURL, unsubscribeLink)
	}
	return nil
}

func newTestURLs() *urlbuilder.Builder {
	return urlbuilder.New("http://localhost:8080")
}

var _ gh.Client = (*mockGitHubClient)(nil)

func TestPoller_NoConfirmedActiveSubscriptions(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{}, nil
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

	p := NewPoller(subRepo, &mockGitHubRepository{}, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if ghClientCalled {
		t.Fatal("github client should not be called when there are no subscriptions")
	}
	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_SameTag_NoEmailSent(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 10, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 10, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil
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

	p := NewPoller(subRepo, repoRepo, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_NewTag_SendsEmailAndUpdatesTag(t *testing.T) {
	lastSeen := "old-tag"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "a@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok1"},
				{ID: 2, Email: "b@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok2"},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 20, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil
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

	p := NewPoller(subRepo, repoRepo, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 2 {
		t.Fatalf("expected 2 emails sent, got %d", m.sendNewReleaseCalls)
	}
	if !updated {
		t.Fatal("expected last seen tag to be updated")
	}
}

func TestPoller_FirstSeenRelease_SetsBaselineWithoutEmail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 30, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 30, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: nil}, nil
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

	p := NewPoller(subRepo, repoRepo, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if !updated {
		t.Fatal("expected baseline tag to be set")
	}
	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_NoReleases_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 40, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 40, FullName: "golang/go", Owner: "golang", Name: "go", LastSeenTag: &lastSeen}, nil
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

	p := NewPoller(subRepo, repoRepo, ghClient, &mockMailer{}, newTestURLs())
	p.Poll(context.Background())
}

func TestPoller_GitHubError_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 50, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 50, FullName: "cli/cli", Owner: "cli", Name: "cli"}, nil
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
	p := NewPoller(subRepo, repoRepo, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_GetAllConfirmedActiveError_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return nil, errors.New("db error")
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			t.Fatal("github client should not be called on db error")
			return "", "", nil
		},
	}

	m := &mockMailer{}
	p := NewPoller(subRepo, &mockGitHubRepository{}, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_GetByIDError_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 60, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return nil, errors.New("db error")
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _, _ string) error {
			t.Fatal("should not update tag on repo lookup error")
			return nil
		},
	}

	m := &mockMailer{}
	p := NewPoller(subRepo, repoRepo, &mockGitHubClient{}, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_RepoNotFound_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 70, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return nil, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _, _ string) error {
			t.Fatal("should not update tag when repo not found")
			return nil
		},
	}

	m := &mockMailer{}
	p := NewPoller(subRepo, repoRepo, &mockGitHubClient{}, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}

func TestPoller_RateLimited_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(_ context.Context) ([]subscription.Subscription, error) {
			return []subscription.Subscription{
				{ID: 1, Email: "test@example.com", RepositoryID: 80, Confirmed: true, Active: true},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(_ context.Context, _ int64) (*repo.Repository, error) {
			return &repo.Repository{ID: 80, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil
		},
		updateLastSeenTagFn: func(_ context.Context, _ int64, _, _ string) error {
			t.Fatal("should not update tag when rate limited")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", gh.ErrRateLimited
		},
	}

	m := &mockMailer{}
	p := NewPoller(subRepo, repoRepo, ghClient, m, newTestURLs())
	p.Poll(context.Background())

	if m.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", m.sendNewReleaseCalls)
	}
}
