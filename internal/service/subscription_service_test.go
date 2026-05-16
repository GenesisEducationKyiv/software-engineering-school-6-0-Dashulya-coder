package service

import (
	"context"
	"errors"
	"testing"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

type mockSubscriptionRepository struct {
	createFn                   func(ctx context.Context, sub *subscription.Subscription) error
	findByConfirmTokenFn       func(ctx context.Context, token string) (*subscription.Subscription, error)
	findByUnsubscribeTokenFn   func(ctx context.Context, token string) (*subscription.Subscription, error)
	getByEmailFn               func(ctx context.Context, email string) ([]subscription.Subscription, error)
	existsByEmailAndRepoFn     func(ctx context.Context, email string, repoID int64) (bool, error)
	confirmByTokenFn           func(ctx context.Context, token string) error
	deactivateByTokenFn        func(ctx context.Context, token string) error
	getAllConfirmedActiveFn     func(ctx context.Context) ([]subscription.Subscription, error)
	getConfirmedActiveByRepoFn func(ctx context.Context, repoID int64) ([]subscription.Subscription, error)
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, sub *subscription.Subscription) error {
	return m.createFn(ctx, sub)
}

func (m *mockSubscriptionRepository) FindByConfirmToken(ctx context.Context, token string) (*subscription.Subscription, error) {
	return m.findByConfirmTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*subscription.Subscription, error) {
	return m.findByUnsubscribeTokenFn(ctx, token)
}

func (m *mockSubscriptionRepository) GetByEmail(ctx context.Context, email string) ([]subscription.Subscription, error) {
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
	findOrCreateFn      func(ctx context.Context, owner, name, fullName string) (*model.GitHubRepository, error)
	findByFullNameFn    func(ctx context.Context, fullName string) (*model.GitHubRepository, error)
	getByIDFn           func(ctx context.Context, id int64) (*model.GitHubRepository, error)
	updateLastSeenTagFn func(ctx context.Context, repoID int64, tag string, releaseURL string) error
}

func (m *mockGitHubRepository) FindOrCreate(ctx context.Context, owner, name, fullName string) (*model.GitHubRepository, error) {
	if m.findOrCreateFn != nil {
		return m.findOrCreateFn(ctx, owner, name, fullName)
	}
	if m.findByFullNameFn != nil {
		return m.findByFullNameFn(ctx, fullName)
	}
	return nil, nil
}

func (m *mockGitHubRepository) FindByFullName(ctx context.Context, fullName string) (*model.GitHubRepository, error) {
	if m.findByFullNameFn != nil {
		return m.findByFullNameFn(ctx, fullName)
	}
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
	repositoryExistsFn func(ctx context.Context, owner, repo string) (bool, error)
	getLatestReleaseFn func(ctx context.Context, owner, repo string) (string, string, error)
}

func (m *mockGitHubClient) RepositoryExists(ctx context.Context, owner, repo string) (bool, error) {
	if m.repositoryExistsFn != nil {
		return m.repositoryExistsFn(ctx, owner, repo)
	}
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
	sendConfirmationFn  func(email, confirmLink string) error
	sendNewReleaseFn    func(email, repo, tag, releaseURL, unsubscribeLink string) error
}

func (m *mockMailer) SendConfirmation(email, confirmLink string) error {
	if m.sendConfirmationFn != nil {
		return m.sendConfirmationFn(email, confirmLink)
	}
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

func TestSubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		createFn: func(ctx context.Context, sub *subscription.Subscription) error {
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
		existsByEmailAndRepoFn:     func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		findByConfirmTokenFn:       func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn:   func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		getByEmailFn:               func(_ context.Context, _ string) ([]subscription.Subscription, error) { return nil, nil },
		confirmByTokenFn:           func(_ context.Context, _ string) error { return nil },
		deactivateByTokenFn:        func(_ context.Context, _ string) error { return nil },
		getAllConfirmedActiveFn:     func(_ context.Context) ([]subscription.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(_ context.Context, _ int64) ([]subscription.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{
		findOrCreateFn: func(_ context.Context, _, _, _ string) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil
		},
	}

	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}

	m := &mockMailer{
		sendConfirmationFn: func(email, confirmLink string) error {
			if email != "test@example.com" {
				t.Fatalf("unexpected recipient: %s", email)
			}
			if confirmLink == "" {
				t.Fatal("confirm link should not be empty")
			}
			return nil
		},
	}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, m, newTestURLs())
	if err := svc.Subscribe(context.Background(), "test@example.com", "golang/go"); err != nil {
		t.Fatalf("Subscribe() unexpected error: %v", err)
	}
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	svc := NewSubscriptionService(&mockSubscriptionRepository{}, &mockGitHubRepository{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs())
	if err := svc.Subscribe(context.Background(), "bad-email", "golang/go"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestSubscribe_InvalidRepo(t *testing.T) {
	svc := NewSubscriptionService(&mockSubscriptionRepository{}, &mockGitHubRepository{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs())
	if err := svc.Subscribe(context.Background(), "test@example.com", "wrongformat"); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}
}

func TestSubscribe_RepoNotFound(t *testing.T) {
	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := NewSubscriptionService(&mockSubscriptionRepository{}, &mockGitHubRepository{}, ghClient, &mockMailer{}, newTestURLs())
	if err := svc.Subscribe(context.Background(), "test@example.com", "owner/repo"); !errors.Is(err, ErrRepoNotFound) {
		t.Fatalf("expected ErrRepoNotFound, got %v", err)
	}
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		existsByEmailAndRepoFn:     func(_ context.Context, _ string, _ int64) (bool, error) { return true, nil },
		findByConfirmTokenFn:       func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn:   func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		getByEmailFn:               func(_ context.Context, _ string) ([]subscription.Subscription, error) { return nil, nil },
		createFn:                   func(_ context.Context, _ *subscription.Subscription) error { return nil },
		confirmByTokenFn:           func(_ context.Context, _ string) error { return nil },
		deactivateByTokenFn:        func(_ context.Context, _ string) error { return nil },
		getAllConfirmedActiveFn:     func(_ context.Context) ([]subscription.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(_ context.Context, _ int64) ([]subscription.Subscription, error) { return nil, nil },
	}

	repoRepo := &mockGitHubRepository{
		findOrCreateFn: func(_ context.Context, _, _, _ string) (*model.GitHubRepository, error) {
			return &model.GitHubRepository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil
		},
	}

	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, &mockMailer{}, newTestURLs())
	if err := svc.Subscribe(context.Background(), "test@example.com", "golang/go"); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("expected ErrAlreadySubscribed, got %v", err)
	}
}

func TestConfirm_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		findByConfirmTokenFn: func(_ context.Context, token string) (*subscription.Subscription, error) {
			return &subscription.Subscription{ID: 1, ConfirmToken: token, Confirmed: false}, nil
		},
		confirmByTokenFn:           func(_ context.Context, _ string) error { return nil },
		findByUnsubscribeTokenFn:   func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		getByEmailFn:               func(_ context.Context, _ string) ([]subscription.Subscription, error) { return nil, nil },
		createFn:                   func(_ context.Context, _ *subscription.Subscription) error { return nil },
		existsByEmailAndRepoFn:     func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		deactivateByTokenFn:        func(_ context.Context, _ string) error { return nil },
		getAllConfirmedActiveFn:     func(_ context.Context) ([]subscription.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(_ context.Context, _ int64) ([]subscription.Subscription, error) { return nil, nil },
	}

	svc := NewSubscriptionService(subRepo, &mockGitHubRepository{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs())
	if err := svc.Confirm(context.Background(), "confirm-token"); err != nil {
		t.Fatalf("Confirm() unexpected error: %v", err)
	}
}

func TestUnsubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionRepository{
		findByUnsubscribeTokenFn: func(_ context.Context, token string) (*subscription.Subscription, error) {
			return &subscription.Subscription{ID: 1, UnsubscribeToken: token, Active: true}, nil
		},
		deactivateByTokenFn:        func(_ context.Context, _ string) error { return nil },
		findByConfirmTokenFn:       func(_ context.Context, _ string) (*subscription.Subscription, error) { return nil, nil },
		getByEmailFn:               func(_ context.Context, _ string) ([]subscription.Subscription, error) { return nil, nil },
		createFn:                   func(_ context.Context, _ *subscription.Subscription) error { return nil },
		existsByEmailAndRepoFn:     func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		confirmByTokenFn:           func(_ context.Context, _ string) error { return nil },
		getAllConfirmedActiveFn:     func(_ context.Context) ([]subscription.Subscription, error) { return nil, nil },
		getConfirmedActiveByRepoFn: func(_ context.Context, _ int64) ([]subscription.Subscription, error) { return nil, nil },
	}

	svc := NewSubscriptionService(subRepo, &mockGitHubRepository{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs())
	if err := svc.Unsubscribe(context.Background(), "unsubscribe-token"); err != nil {
		t.Fatalf("Unsubscribe() unexpected error: %v", err)
	}
}

var _ gh.Client = (*mockGitHubClient)(nil)
