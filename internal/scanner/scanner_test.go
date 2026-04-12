package scanner

import (
	"context"
	"errors"
	"testing"
	"time"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
)

type mockSubscriptionRepository struct {
	getAllConfirmedActiveFn    func(ctx context.Context) ([]model.Subscription, error)
	getConfirmedActiveByRepoFn func(ctx context.Context, repoID int64) ([]model.Subscription, error)
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepository) FindByConfirmToken(ctx context.Context, token string) (*model.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*model.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) GetByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepository) ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error) {
	return false, nil
}

func (m *mockSubscriptionRepository) ConfirmByToken(ctx context.Context, token string) error {
	return nil
}

func (m *mockSubscriptionRepository) DeactivateByToken(ctx context.Context, token string) error {
	return nil
}

func (m *mockSubscriptionRepository) GetAllConfirmedActive(ctx context.Context) ([]model.Subscription, error) {
	if m.getAllConfirmedActiveFn != nil {
		return m.getAllConfirmedActiveFn(ctx)
	}
	return nil, nil
}

func (m *mockSubscriptionRepository) GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]model.Subscription, error) {
	if m.getConfirmedActiveByRepoFn != nil {
		return m.getConfirmedActiveByRepoFn(ctx, repoID)
	}
	return nil, nil
}

type mockGitHubRepository struct {
	getByIDFn           func(ctx context.Context, id int64) (*model.GitHubRepository, error)
	updateLastSeenTagFn func(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

func (m *mockGitHubRepository) Create(ctx context.Context, repo *model.GitHubRepository) error {
	return nil
}

func (m *mockGitHubRepository) FindByFullName(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
	return nil, nil
}

func (m *mockGitHubRepository) GetByID(ctx context.Context, id int64) (*model.GitHubRepository, error) {
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

func (m *mockGitHubClient) RepositoryExists(ctx context.Context, owner, repo string) (bool, error) {
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

func (m *mockMailer) SendConfirmation(email, confirmLink string) error {
	return nil
}

func (m *mockMailer) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	m.sendNewReleaseCalls++
	if m.sendNewReleaseFn != nil {
		return m.sendNewReleaseFn(email, repo, tag, releaseURL, unsubscribeLink)
	}
	return nil
}

func TestScanner_Run_NoConfirmedActiveSubscriptions(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{}, nil
		},
	}

	repoRepo := &mockGitHubRepository{}
	ghClientCalled := false
	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			ghClientCalled = true
			return "", "", nil
		},
	}
	mailer := &mockMailer{}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if ghClientCalled {
		t.Fatal("github client should not be called when there are no subscriptions")
	}
	if mailer.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
}

func TestScanner_Run_SameTag_NoEmailSent(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{
					ID:               1,
					Email:            "test@example.com",
					RepositoryID:     10,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token",
				},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(ctx context.Context, id int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:          10,
				FullName:    "cli/cli",
				Owner:       "cli",
				Name:        "cli",
				LastSeenTag: &lastSeen,
			}, nil
		},
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error {
			t.Fatal("last seen tag should not be updated when tag did not change")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "v1.0.0", "https://example.com/release", nil
		},
	}

	mailer := &mockMailer{
		sendNewReleaseFn: func(email, repo, tag, releaseURL, unsubscribeLink string) error {
			t.Fatal("email should not be sent when tag did not change")
			return nil
		},
	}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if mailer.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
}

func TestScanner_Run_NewTag_SendsEmailAndUpdatesLastSeenTag(t *testing.T) {
	lastSeen := "old-tag"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{
					ID:               1,
					Email:            "test@example.com",
					RepositoryID:     20,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token-1",
				},
				{
					ID:               2,
					Email:            "test2@example.com",
					RepositoryID:     20,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token-2",
				},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(ctx context.Context, id int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:          20,
				FullName:    "cli/cli",
				Owner:       "cli",
				Name:        "cli",
				LastSeenTag: &lastSeen,
			}, nil
		},
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error {
			updated = true
			if repoID != 20 {
				t.Fatalf("unexpected repo id: %d", repoID)
			}
			if tag != "v2.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			if releaseURL != "https://example.com/release/v2.0.0" {
				t.Fatalf("unexpected release URL: %s", releaseURL)
			}
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "v2.0.0", "https://example.com/release/v2.0.0", nil
		},
	}

	mailer := &mockMailer{
		sendNewReleaseFn: func(email, repo, tag, releaseURL, unsubscribeLink string) error {
			if repo != "cli/cli" {
				t.Fatalf("unexpected repo: %s", repo)
			}
			if tag != "v2.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			if releaseURL != "https://example.com/release/v2.0.0" {
				t.Fatalf("unexpected release URL: %s", releaseURL)
			}
			if unsubscribeLink == "" {
				t.Fatal("unsubscribe link should not be empty")
			}
			return nil
		},
	}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if mailer.sendNewReleaseCalls != 2 {
		t.Fatalf("expected 2 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
	if !updated {
		t.Fatal("expected last seen tag to be updated")
	}
}

func TestScanner_Run_FirstSeenRelease_SetsBaselineWithoutSendingEmail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{
					ID:               1,
					Email:            "test@example.com",
					RepositoryID:     30,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token",
				},
			}, nil
		},
	}

	updated := false
	repoRepo := &mockGitHubRepository{
		getByIDFn: func(ctx context.Context, id int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:          30,
				FullName:    "cli/cli",
				Owner:       "cli",
				Name:        "cli",
				LastSeenTag: nil,
			}, nil
		},
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error {
			updated = true
			if tag != "v3.0.0" {
				t.Fatalf("unexpected tag: %s", tag)
			}
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "v3.0.0", "https://example.com/release/v3.0.0", nil
		},
	}

	mailer := &mockMailer{
		sendNewReleaseFn: func(email, repo, tag, releaseURL, unsubscribeLink string) error {
			t.Fatal("email should not be sent when setting baseline")
			return nil
		},
	}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if !updated {
		t.Fatal("expected baseline tag to be set")
	}
	if mailer.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
}

func TestScanner_Run_NoReleases_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{
					ID:               1,
					Email:            "test@example.com",
					RepositoryID:     40,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token",
				},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(ctx context.Context, id int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:          40,
				FullName:    "golang/go",
				Owner:       "golang",
				Name:        "go",
				LastSeenTag: &lastSeen,
			}, nil
		},
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error {
			t.Fatal("last seen tag should not be updated when repo has no releases")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "", "", gh.ErrNoReleases
		},
	}

	mailer := &mockMailer{
		sendNewReleaseFn: func(email, repo, tag, releaseURL, unsubscribeLink string) error {
			t.Fatal("email should not be sent when repo has no releases")
			return nil
		},
	}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if mailer.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
}

func TestScanner_Run_GitHubError_DoesNotFail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		getAllConfirmedActiveFn: func(ctx context.Context) ([]model.Subscription, error) {
			return []model.Subscription{
				{
					ID:               1,
					Email:            "test@example.com",
					RepositoryID:     50,
					Confirmed:        true,
					Active:           true,
					UnsubscribeToken: "unsubscribe-token",
				},
			}, nil
		},
	}

	repoRepo := &mockGitHubRepository{
		getByIDFn: func(ctx context.Context, id int64) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:       50,
				FullName: "cli/cli",
				Owner:    "cli",
				Name:     "cli",
			}, nil
		},
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error {
			t.Fatal("last seen tag should not be updated on github error")
			return nil
		},
	}

	ghClient := &mockGitHubClient{
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "", "", errors.New("network error")
		},
	}

	mailer := &mockMailer{}

	sc := New(subRepo, repoRepo, ghClient, mailer, 5*time.Second, "http://localhost:8080")
	sc.run(context.Background())

	if mailer.sendNewReleaseCalls != 0 {
		t.Fatalf("expected 0 emails sent, got %d", mailer.sendNewReleaseCalls)
	}
}
