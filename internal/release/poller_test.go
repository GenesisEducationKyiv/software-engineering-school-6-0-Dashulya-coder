package release

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	gh "github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

type mockSubscriptionRepository struct {
	mock.Mock
}

func (m *mockSubscriptionRepository) CreateForSaga(
	ctx context.Context, sub *subscription.Subscription, sagaID string,
) (bool, error) {
	args := m.Called(ctx, sub, sagaID)
	return args.Bool(0), args.Error(1)
}

func (m *mockSubscriptionRepository) CancelBySaga(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

func (m *mockSubscriptionRepository) FindByConfirmToken(
	ctx context.Context, token string,
) (*subscription.Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *mockSubscriptionRepository) FindByUnsubscribeToken(
	ctx context.Context, token string,
) (*subscription.Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.Subscription), args.Error(1)
}

func (m *mockSubscriptionRepository) GetByEmail(
	ctx context.Context, email string,
) ([]subscription.Subscription, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.Subscription), args.Error(1)
}

func (m *mockSubscriptionRepository) ConfirmByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockSubscriptionRepository) DeactivateByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockSubscriptionRepository) GetAllConfirmedActive(
	ctx context.Context,
) ([]subscription.Subscription, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.Subscription), args.Error(1)
}

func (m *mockSubscriptionRepository) GetConfirmedActiveByRepo(
	ctx context.Context, repoID int64,
) ([]subscription.Subscription, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]subscription.Subscription), args.Error(1)
}

type mockGitHubRepository struct {
	mock.Mock
}

func (m *mockGitHubRepository) FindOrCreate(
	ctx context.Context, owner, name, fullName string,
) (*repo.Repository, error) {
	args := m.Called(ctx, owner, name, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Repository), args.Error(1)
}

func (m *mockGitHubRepository) FindByFullName(
	ctx context.Context, fullName string,
) (*repo.Repository, error) {
	args := m.Called(ctx, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Repository), args.Error(1)
}

func (m *mockGitHubRepository) GetByID(ctx context.Context, id int64) (*repo.Repository, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Repository), args.Error(1)
}

func (m *mockGitHubRepository) UpdateLastSeenTag(
	ctx context.Context, repoID int64, tag string, releaseURL string,
) error {
	return m.Called(ctx, repoID, tag, releaseURL).Error(0)
}

type mockGitHubClient struct {
	mock.Mock
}

func (m *mockGitHubClient) RepositoryExists(ctx context.Context, owner, repoName string) (bool, error) {
	args := m.Called(ctx, owner, repoName)
	return args.Bool(0), args.Error(1)
}

func (m *mockGitHubClient) GetLatestRelease(
	ctx context.Context, owner, repoName string,
) (string, string, error) {
	args := m.Called(ctx, owner, repoName)
	return args.String(0), args.String(1), args.Error(2)
}

type mockMailer struct {
	mock.Mock
}

func (m *mockMailer) SendConfirmation(email, link string) error {
	return m.Called(email, link).Error(0)
}

func (m *mockMailer) SendNewRelease(email, repoName, tag, releaseURL, unsubscribeLink string) error {
	return m.Called(email, repoName, tag, releaseURL, unsubscribeLink).Error(0)
}

func newTestURLs() *urlbuilder.Builder {
	return urlbuilder.New("http://localhost:8080")
}

func TestPoller_NoConfirmedActiveSubscriptions(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).
		Return([]subscription.Subscription{}, nil).Once()

	ghClient := new(mockGitHubClient)
	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, new(mockGitHubRepository), ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	subRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_SameTag_NoEmailSent(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 10, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(10)).
		Return(&repo.Repository{ID: 10, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "cli", "cli").
		Return("v1.0.0", "https://example.com/release", nil).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_NewTag_SendsEmailAndUpdatesTag(t *testing.T) {
	lastSeen := "old-tag"

	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "a@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok1"},
		{ID: 2, Email: "b@example.com", RepositoryID: 20, Confirmed: true, Active: true, UnsubscribeToken: "tok2"},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(20)).
		Return(&repo.Repository{ID: 20, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil).Once()
	repoRepo.On("UpdateLastSeenTag", mock.Anything, int64(20), "v2.0.0", "https://example.com/v2.0.0").
		Return(nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "cli", "cli").
		Return("v2.0.0", "https://example.com/v2.0.0", nil).Once()

	mailerMock := new(mockMailer)
	mailerMock.On("SendNewRelease", "a@example.com", "cli/cli", "v2.0.0", "https://example.com/v2.0.0",
		mock.MatchedBy(func(link string) bool { return link != "" })).Return(nil).Once()
	mailerMock.On("SendNewRelease", "b@example.com", "cli/cli", "v2.0.0", "https://example.com/v2.0.0",
		mock.MatchedBy(func(link string) bool { return link != "" })).Return(nil).Once()

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 2)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_FirstSeenRelease_SetsBaselineWithoutEmail(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 30, Confirmed: true, Active: true, UnsubscribeToken: "tok"},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(30)).
		Return(&repo.Repository{ID: 30, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: nil}, nil).Once()
	repoRepo.On("UpdateLastSeenTag", mock.Anything, int64(30), "v3.0.0", "https://example.com/v3.0.0").
		Return(nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "cli", "cli").
		Return("v3.0.0", "https://example.com/v3.0.0", nil).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_NoReleases_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 40, Confirmed: true, Active: true},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(40)).
		Return(&repo.Repository{ID: 40, FullName: "golang/go", Owner: "golang", Name: "go", LastSeenTag: &lastSeen}, nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "golang", "go").
		Return("", "", gh.ErrNoReleases).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_GitHubError_DoesNotFail(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 50, Confirmed: true, Active: true},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(50)).
		Return(&repo.Repository{ID: 50, FullName: "cli/cli", Owner: "cli", Name: "cli"}, nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "cli", "cli").
		Return("", "", errors.New("network error")).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_GetAllConfirmedActiveError_DoesNotFail(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return(nil, errors.New("db error")).Once()

	ghClient := new(mockGitHubClient)
	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, new(mockGitHubRepository), ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_GetByIDError_DoesNotFail(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 60, Confirmed: true, Active: true},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(60)).
		Return(nil, errors.New("db error")).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, new(mockGitHubClient), mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_RepoNotFound_DoesNotFail(t *testing.T) {
	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 70, Confirmed: true, Active: true},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(70)).Return(nil, nil).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, new(mockGitHubClient), mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestPoller_RateLimited_DoesNotFail(t *testing.T) {
	lastSeen := "v1.0.0"

	subRepo := new(mockSubscriptionRepository)
	subRepo.On("GetAllConfirmedActive", mock.Anything).Return([]subscription.Subscription{
		{ID: 1, Email: "test@example.com", RepositoryID: 80, Confirmed: true, Active: true},
	}, nil).Once()

	repoRepo := new(mockGitHubRepository)
	repoRepo.On("GetByID", mock.Anything, int64(80)).
		Return(&repo.Repository{ID: 80, FullName: "cli/cli", Owner: "cli", Name: "cli", LastSeenTag: &lastSeen}, nil).Once()

	ghClient := new(mockGitHubClient)
	ghClient.On("GetLatestRelease", mock.Anything, "cli", "cli").
		Return("", "", gh.ErrRateLimited).Once()

	mailerMock := new(mockMailer)

	p := NewPoller(subRepo, repoRepo, ghClient, mailerMock, newTestURLs())
	p.Poll(context.Background())

	mailerMock.AssertNumberOfCalls(t, "SendNewRelease", 0)
	subRepo.AssertExpectations(t)
	repoRepo.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}
