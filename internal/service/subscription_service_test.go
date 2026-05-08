package service

import (
	"context"
	"errors"
	"testing"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
)

type mockSubscriptionRepository struct {
	createFn                   func(ctx context.Context, sub *model.Subscription) error
	findByConfirmTokenFn       func(ctx context.Context, token string) (*model.Subscription, error)
	findByUnsubscribeTokenFn   func(ctx context.Context, token string) (*model.Subscription, error)
	getByEmailFn               func(ctx context.Context, email string) ([]model.Subscription, error)
	existsByEmailAndRepoFn     func(ctx context.Context, email string, repoID int64) (bool, error)
	confirmByTokenFn           func(ctx context.Context, token string) error
	deactivateByTokenFn        func(ctx context.Context, token string) error
	getAllConfirmedActiveFn    func(ctx context.Context) ([]model.Subscription, error)
	getConfirmedActiveByRepoFn func(ctx context.Context, repoID int64) ([]model.Subscription, error)
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	return m.createFn(ctx, sub)
}

func (m *mockSubscriptionRepository) FindByConfirmToken(ctx context.Context, token string) (*model.Subscription, error) {
	return m.findByConfirmTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*model.Subscription, error) {
	return m.findByUnsubscribeTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) GetByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockSubscriptionRepository) ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error) {
	return m.existsByEmailAndRepoFn(ctx, email, repoID)
}

func (m *mockSubscriptionRepository) ConfirmByToken(ctx context.Context, token string) error {
	return m.confirmByTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) DeactivateByToken(ctx context.Context, token string) error {
	return m.deactivateByTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) GetAllConfirmedActive(ctx context.Context) ([]model.Subscription, error) {
	return m.getAllConfirmedActiveFn(ctx)
}

func (m *mockSubscriptionRepository) GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]model.Subscription, error) {
	return m.getConfirmedActiveByRepoFn(ctx, repoID)
}

type mockGitHubRepository struct {
	createFn            func(ctx context.Context, repo *model.GitHubRepository) error
	findByFullNameFn    func(ctx context.Context, fullName string) (*model.GitHubRepository, error)
	getByIDFn           func(ctx context.Context, id int64) (*model.GitHubRepository, error)
	updateLastSeenTagFn func(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

func (m *mockGitHubRepository) Create(ctx context.Context, repo *model.GitHubRepository) error {
	return m.createFn(ctx, repo)
}

func (m *mockGitHubRepository) FindByFullName(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
	return m.findByFullNameFn(ctx, fullName)
}

func (m *mockGitHubRepository) GetByID(ctx context.Context, id int64) (*model.GitHubRepository, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockGitHubRepository) UpdateLastSeenTag(ctx context.Context, repoID int64, tag string, releaseURL string) error {
	return m.updateLastSeenTagFn(ctx, repoID, tag, releaseURL)
}

type mockGitHubClient struct {
	repositoryExistsFn func(ctx context.Context, owner, repo string) (bool, error)
	getLatestReleaseFn func(ctx context.Context, owner, repo string) (string, string, error)
}

func (m *mockGitHubClient) RepositoryExists(ctx context.Context, owner, repo string) (bool, error) {
	return m.repositoryExistsFn(ctx, owner, repo)
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (string, string, error) {
	return m.getLatestReleaseFn(ctx, owner, repo)
}

type mockMailer struct {
	sendConfirmationFn func(email, confirmLink string) error
	sendNewReleaseFn   func(email, repo, tag, releaseURL, unsubscribeLink string) error
}

func (m *mockMailer) SendConfirmation(email, confirmLink string) error {
	return m.sendConfirmationFn(email, confirmLink)
}

func (m *mockMailer) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	return m.sendNewReleaseFn(email, repo, tag, releaseURL, unsubscribeLink)
}

func TestSubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		createFn: func(ctx context.Context, sub *model.Subscription) error {
			if sub.Email != "test@example.com" {
				t.Fatalf("unexpected email: %s", sub.Email)
			}
			if sub.RepositoryID != 1 {
				t.Fatalf("unexpected repository id: %d", sub.RepositoryID)
			}
			if sub.ConfirmToken == "" {
				t.Fatal("confirm token should not be empty")
			}
			if sub.UnsubscribeToken == "" {
				t.Fatal("unsubscribe token should not be empty")
			}
			return nil
		},
		existsByEmailAndRepoFn: func(ctx context.Context, email string, repoID int64) (bool, error) {
			return false, nil
		},
		findByConfirmTokenFn:       func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn:   func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		getByEmailFn:               func(ctx context.Context, email string) ([]model.Subscription, error) { return nil, nil },
		confirmByTokenFn:           func(ctx context.Context, token string) error { return nil },
		deactivateByTokenFn:        func(ctx context.Context, token string) error { return nil },
		getAllConfirmedActiveFn:    func(ctx context.Context) ([]model.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(ctx context.Context, repoID int64) ([]model.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{
		findByFullNameFn: func(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:       1,
				FullName: "golang/go",
				Owner:    "golang",
				Name:     "go",
			}, nil
		},
		createFn:            func(ctx context.Context, repo *model.GitHubRepository) error { return nil },
		getByIDFn:           func(ctx context.Context, id int64) (*model.GitHubRepository, error) { return nil, nil },
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error { return nil },
	}

	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(ctx context.Context, owner, repo string) (bool, error) {
			return true, nil
		},
		getLatestReleaseFn: func(ctx context.Context, owner, repo string) (string, string, error) {
			return "", "", nil
		},
	}

	mailer := &mockMailer{
		sendConfirmationFn: func(email, confirmLink string) error {
			if email != "test@example.com" {
				t.Fatalf("unexpected recipient: %s", email)
			}
			if confirmLink == "" {
				t.Fatal("confirm link should not be empty")
			}
			return nil
		},
		sendNewReleaseFn: func(email, repo, tag, releaseURL, unsubscribeLink string) error {
			return nil
		},
	}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Subscribe(context.Background(), "test@example.com", "golang/go")
	if err != nil {
		t.Fatalf("Subscribe() unexpected error: %v", err)
	}
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	subRepo := &mockSubscriptionRepository{}
	repoRepo := &mockGitHubRepository{}
	ghClient := &mockGitHubClient{}
	mailer := &mockMailer{}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Subscribe(context.Background(), "bad-email", "golang/go")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestSubscribe_InvalidRepo(t *testing.T) {
	subRepo := &mockSubscriptionRepository{}
	repoRepo := &mockGitHubRepository{}
	ghClient := &mockGitHubClient{}
	mailer := &mockMailer{}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Subscribe(context.Background(), "test@example.com", "wrongformat")
	if !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}
}

func TestSubscribe_RepoNotFound(t *testing.T) {
	subRepo := &mockSubscriptionRepository{}
	repoRepo := &mockGitHubRepository{}
	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(ctx context.Context, owner, repo string) (bool, error) {
			return false, nil
		},
	}
	mailer := &mockMailer{}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Subscribe(context.Background(), "test@example.com", "owner/repo")
	if !errors.Is(err, ErrRepoNotFound) {
		t.Fatalf("expected ErrRepoNotFound, got %v", err)
	}
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		existsByEmailAndRepoFn: func(ctx context.Context, email string, repoID int64) (bool, error) {
			return true, nil
		},
		findByConfirmTokenFn:       func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn:   func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		getByEmailFn:               func(ctx context.Context, email string) ([]model.Subscription, error) { return nil, nil },
		createFn:                   func(ctx context.Context, sub *model.Subscription) error { return nil },
		confirmByTokenFn:           func(ctx context.Context, token string) error { return nil },
		deactivateByTokenFn:        func(ctx context.Context, token string) error { return nil },
		getAllConfirmedActiveFn:    func(ctx context.Context) ([]model.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(ctx context.Context, repoID int64) ([]model.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{
		findByFullNameFn: func(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{
				ID:       1,
				FullName: "golang/go",
				Owner:    "golang",
				Name:     "go",
			}, nil
		},
		createFn:            func(ctx context.Context, repo *model.GitHubRepository) error { return nil },
		getByIDFn:           func(ctx context.Context, id int64) (*model.GitHubRepository, error) { return nil, nil },
		updateLastSeenTagFn: func(ctx context.Context, repoID int64, tag string, releaseURL string) error { return nil },
	}

	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(ctx context.Context, owner, repo string) (bool, error) {
			return true, nil
		},
	}

	mailer := &mockMailer{
		sendConfirmationFn: func(email, confirmLink string) error { return nil },
		sendNewReleaseFn:   func(email, repo, tag, releaseURL, unsubscribeLink string) error { return nil },
	}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Subscribe(context.Background(), "test@example.com", "golang/go")
	if !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("expected ErrAlreadySubscribed, got %v", err)
	}
}

func TestConfirm_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		findByConfirmTokenFn: func(ctx context.Context, token string) (*model.Subscription, error) {
			return &model.Subscription{
				ID:           1,
				ConfirmToken: token,
				Confirmed:    false,
			}, nil
		},
		confirmByTokenFn: func(ctx context.Context, token string) error {
			return nil
		},
		findByUnsubscribeTokenFn:   func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		getByEmailFn:               func(ctx context.Context, email string) ([]model.Subscription, error) { return nil, nil },
		createFn:                   func(ctx context.Context, sub *model.Subscription) error { return nil },
		existsByEmailAndRepoFn:     func(ctx context.Context, email string, repoID int64) (bool, error) { return false, nil },
		deactivateByTokenFn:        func(ctx context.Context, token string) error { return nil },
		getAllConfirmedActiveFn:    func(ctx context.Context) ([]model.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(ctx context.Context, repoID int64) ([]model.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{}
	ghClient := &mockGitHubClient{}
	mailer := &mockMailer{}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Confirm(context.Background(), "confirm-token")
	if err != nil {
		t.Fatalf("Confirm() unexpected error: %v", err)
	}
}

func TestUnsubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		findByUnsubscribeTokenFn: func(ctx context.Context, token string) (*model.Subscription, error) {
			return &model.Subscription{
				ID:               1,
				UnsubscribeToken: token,
				Active:           true,
			}, nil
		},
		deactivateByTokenFn: func(ctx context.Context, token string) error {
			return nil
		},
		findByConfirmTokenFn:       func(ctx context.Context, token string) (*model.Subscription, error) { return nil, nil },
		getByEmailFn:               func(ctx context.Context, email string) ([]model.Subscription, error) { return nil, nil },
		createFn:                   func(ctx context.Context, sub *model.Subscription) error { return nil },
		existsByEmailAndRepoFn:     func(ctx context.Context, email string, repoID int64) (bool, error) { return false, nil },
		confirmByTokenFn:           func(ctx context.Context, token string) error { return nil },
		getAllConfirmedActiveFn:    func(ctx context.Context) ([]model.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(ctx context.Context, repoID int64) ([]model.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{}
	ghClient := &mockGitHubClient{}
	mailer := &mockMailer{}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, mailer, "http://localhost:8080")

	err := svc.Unsubscribe(context.Background(), "unsubscribe-token")
	if err != nil {
		t.Fatalf("Unsubscribe() unexpected error: %v", err)
	}
}

var _ gh.Client = (*mockGitHubClient)(nil)
