package subscription

import (
	"context"
	"errors"
	"testing"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

type mockSubscriptionStore struct {
	createFn                   func(ctx context.Context, sub *Subscription) error
	findByConfirmTokenFn       func(ctx context.Context, token string) (*Subscription, error)
	findByUnsubscribeTokenFn   func(ctx context.Context, token string) (*Subscription, error)
	getByEmailFn               func(ctx context.Context, email string) ([]Subscription, error)
	existsByEmailAndRepoFn     func(ctx context.Context, email string, repoID int64) (bool, error)
	confirmByTokenFn           func(ctx context.Context, token string) error
	deactivateByTokenFn        func(ctx context.Context, token string) error
}

func (m *mockSubscriptionStore) Create(ctx context.Context, sub *Subscription) error {
	return m.createFn(ctx, sub)
}

func (m *mockSubscriptionStore) FindByConfirmToken(ctx context.Context, token string) (*Subscription, error) {
	return m.findByConfirmTokenFn(ctx, token)
}

func (m *mockSubscriptionStore) FindByUnsubscribeToken(ctx context.Context, token string) (*Subscription, error) {
	return m.findByUnsubscribeTokenFn(ctx, token)
}

func (m *mockSubscriptionStore) GetByEmail(ctx context.Context, email string) ([]Subscription, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockSubscriptionStore) ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error) {
	return m.existsByEmailAndRepoFn(ctx, email, repoID)
}

func (m *mockSubscriptionStore) ConfirmByToken(ctx context.Context, token string) error {
	return m.confirmByTokenFn(ctx, token)
}

func (m *mockSubscriptionStore) DeactivateByToken(ctx context.Context, token string) error {
	return m.deactivateByTokenFn(ctx, token)
}

type mockRepoStore struct {
	findOrCreateFn func(ctx context.Context, owner, name, fullName string) (*repo.Repository, error)
	getByIDFn      func(ctx context.Context, id int64) (*repo.Repository, error)
}

func (m *mockRepoStore) FindOrCreate(ctx context.Context, owner, name, fullName string) (*repo.Repository, error) {
	if m.findOrCreateFn != nil {
		return m.findOrCreateFn(ctx, owner, name, fullName)
	}
	return nil, nil
}

func (m *mockRepoStore) GetByID(ctx context.Context, id int64) (*repo.Repository, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
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
	sendConfirmationFn func(email, confirmLink string) error
	sendNewReleaseFn   func(email, repo, tag, releaseURL, unsubscribeLink string) error
}

func (m *mockMailer) SendConfirmation(email, confirmLink string) error {
	if m.sendConfirmationFn != nil {
		return m.sendConfirmationFn(email, confirmLink)
	}
	return nil
}

func (m *mockMailer) SendNewRelease(email, repo, tag, releaseURL, unsubscribeLink string) error {
	if m.sendNewReleaseFn != nil {
		return m.sendNewReleaseFn(email, repo, tag, releaseURL, unsubscribeLink)
	}
	return nil
}

func newTestURLs() *urlbuilder.Builder {
	return urlbuilder.New("http://localhost:8080")
}

type mockTokenGenerator struct {
	generateFn func() (string, error)
}

func (m *mockTokenGenerator) Generate() (string, error) {
	if m.generateFn != nil {
		return m.generateFn()
	}
	return "test-token", nil
}

var _ SubscriptionStore = (*mockSubscriptionStore)(nil)
var _ RepoStore = (*mockRepoStore)(nil)
var _ gh.Client = (*mockGitHubClient)(nil)
var _ TokenGenerator = (*mockTokenGenerator)(nil)

func TestSubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionStore{
		createFn: func(ctx context.Context, sub *Subscription) error {
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
		existsByEmailAndRepoFn:   func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		findByConfirmTokenFn:     func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn: func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		getByEmailFn:             func(_ context.Context, _ string) ([]Subscription, error) { return nil, nil },
		confirmByTokenFn:         func(_ context.Context, _ string) error { return nil },
		deactivateByTokenFn:      func(_ context.Context, _ string) error { return nil },
	}

	repoRepo := &mockRepoStore{
		findOrCreateFn: func(_ context.Context, _, _, _ string) (*repo.Repository, error) {
			return &repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil
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

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, m, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Subscribe(context.Background(), "test@example.com", "golang/go"); err != nil {
		t.Fatalf("Subscribe() unexpected error: %v", err)
	}
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	svc := NewSubscriptionService(&mockSubscriptionStore{}, &mockRepoStore{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Subscribe(context.Background(), "bad-email", "golang/go"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestSubscribe_InvalidRepo(t *testing.T) {
	svc := NewSubscriptionService(&mockSubscriptionStore{}, &mockRepoStore{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Subscribe(context.Background(), "test@example.com", "wrongformat"); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}
}

func TestSubscribe_RepoNotFound(t *testing.T) {
	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := NewSubscriptionService(&mockSubscriptionStore{}, &mockRepoStore{}, ghClient, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Subscribe(context.Background(), "test@example.com", "owner/repo"); !errors.Is(err, ErrRepoNotFound) {
		t.Fatalf("expected ErrRepoNotFound, got %v", err)
	}
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	subRepo := &mockSubscriptionStore{
		existsByEmailAndRepoFn:   func(_ context.Context, _ string, _ int64) (bool, error) { return true, nil },
		findByConfirmTokenFn:     func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		findByUnsubscribeTokenFn: func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		getByEmailFn:             func(_ context.Context, _ string) ([]Subscription, error) { return nil, nil },
		createFn:                 func(_ context.Context, _ *Subscription) error { return nil },
		confirmByTokenFn:         func(_ context.Context, _ string) error { return nil },
		deactivateByTokenFn:      func(_ context.Context, _ string) error { return nil },
	}

	repoRepo := &mockRepoStore{
		findOrCreateFn: func(_ context.Context, _, _, _ string) (*repo.Repository, error) {
			return &repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil
		},
	}

	ghClient := &mockGitHubClient{
		repositoryExistsFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}

	svc := NewSubscriptionService(subRepo, repoRepo, ghClient, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Subscribe(context.Background(), "test@example.com", "golang/go"); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("expected ErrAlreadySubscribed, got %v", err)
	}
}

func TestConfirm_Success(t *testing.T) {
	subRepo := &mockSubscriptionStore{
		findByConfirmTokenFn: func(_ context.Context, token string) (*Subscription, error) {
			return &Subscription{ID: 1, ConfirmToken: token, Confirmed: false}, nil
		},
		confirmByTokenFn:         func(_ context.Context, _ string) error { return nil },
		findByUnsubscribeTokenFn: func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		getByEmailFn:             func(_ context.Context, _ string) ([]Subscription, error) { return nil, nil },
		createFn:                 func(_ context.Context, _ *Subscription) error { return nil },
		existsByEmailAndRepoFn:   func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		deactivateByTokenFn:      func(_ context.Context, _ string) error { return nil },
	}

	svc := NewSubscriptionService(subRepo, &mockRepoStore{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Confirm(context.Background(), "confirm-token"); err != nil {
		t.Fatalf("Confirm() unexpected error: %v", err)
	}
}

func TestUnsubscribe_Success(t *testing.T) {
	subRepo := &mockSubscriptionStore{
		findByUnsubscribeTokenFn: func(_ context.Context, token string) (*Subscription, error) {
			return &Subscription{ID: 1, UnsubscribeToken: token, Active: true}, nil
		},
		deactivateByTokenFn:    func(_ context.Context, _ string) error { return nil },
		findByConfirmTokenFn:   func(_ context.Context, _ string) (*Subscription, error) { return nil, nil },
		getByEmailFn:           func(_ context.Context, _ string) ([]Subscription, error) { return nil, nil },
		createFn:               func(_ context.Context, _ *Subscription) error { return nil },
		existsByEmailAndRepoFn: func(_ context.Context, _ string, _ int64) (bool, error) { return false, nil },
		confirmByTokenFn:       func(_ context.Context, _ string) error { return nil },
	}

	svc := NewSubscriptionService(subRepo, &mockRepoStore{}, &mockGitHubClient{}, &mockMailer{}, newTestURLs(), &mockTokenGenerator{})
	if err := svc.Unsubscribe(context.Background(), "unsubscribe-token"); err != nil {
		t.Fatalf("Unsubscribe() unexpected error: %v", err)
	}
}
