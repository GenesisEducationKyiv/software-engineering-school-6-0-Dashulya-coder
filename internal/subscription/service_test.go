package subscription

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repo"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

var errBoom = errors.New("boom")

type mockSubscriptionStore struct {
	mock.Mock
}

func (m *mockSubscriptionStore) CreateForSaga(ctx context.Context, sub *Subscription, sagaID string) (bool, error) {
	args := m.Called(ctx, sub, sagaID)
	return args.Bool(0), args.Error(1)
}

func (m *mockSubscriptionStore) CancelBySaga(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

func (m *mockSubscriptionStore) FindByConfirmToken(ctx context.Context, token string) (*Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Subscription), args.Error(1)
}

func (m *mockSubscriptionStore) FindByUnsubscribeToken(ctx context.Context, token string) (*Subscription, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Subscription), args.Error(1)
}

func (m *mockSubscriptionStore) GetByEmail(ctx context.Context, email string) ([]Subscription, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Subscription), args.Error(1)
}

func (m *mockSubscriptionStore) ConfirmByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockSubscriptionStore) DeactivateByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

type mockRepoStore struct {
	mock.Mock
}

func (m *mockRepoStore) FindOrCreate(ctx context.Context, owner, name, fullName string) (*repo.Repository, error) {
	args := m.Called(ctx, owner, name, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Repository), args.Error(1)
}

func (m *mockRepoStore) GetByID(ctx context.Context, id int64) (*repo.Repository, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo.Repository), args.Error(1)
}

type mockGitHubClient struct {
	mock.Mock
}

func (m *mockGitHubClient) RepositoryExists(ctx context.Context, owner, repo string) (bool, error) {
	args := m.Called(ctx, owner, repo)
	return args.Bool(0), args.Error(1)
}

func (m *mockGitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (string, string, error) {
	args := m.Called(ctx, owner, repo)
	return args.String(0), args.String(1), args.Error(2)
}

type mockNotifier struct {
	mock.Mock
}

func (m *mockNotifier) ReserveConfirmation(ctx context.Context, sagaID, email, confirmURL string) error {
	return m.Called(ctx, sagaID, email, confirmURL).Error(0)
}

func (m *mockNotifier) CommitConfirmation(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

func (m *mockNotifier) CancelConfirmation(ctx context.Context, sagaID string) error {
	return m.Called(ctx, sagaID).Error(0)
}

type mockTokenGenerator struct {
	mock.Mock
}

func (m *mockTokenGenerator) Generate() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func newTestURLs() *urlbuilder.Builder {
	return urlbuilder.New("http://localhost:8080")
}

func TestSubscribe_Success(t *testing.T) {
	subStore := new(mockSubscriptionStore)
	repoStore := new(mockRepoStore)
	ghClient := new(mockGitHubClient)
	notifier := new(mockNotifier)
	tokenGen := new(mockTokenGenerator)

	ghClient.On("RepositoryExists", mock.Anything, "golang", "go").Return(true, nil).Once()
	repoStore.On("FindOrCreate", mock.Anything, "golang", "go", "golang/go").
		Return(&repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil).Once()
	tokenGen.On("Generate").Return("confirm-token", nil).Once()
	tokenGen.On("Generate").Return("unsub-token", nil).Once()
	subStore.On("CreateForSaga", mock.Anything, mock.MatchedBy(func(sub *Subscription) bool {
		return sub.Email == "test@example.com" &&
			sub.RepositoryID == 1 &&
			sub.ConfirmToken != "" &&
			sub.UnsubscribeToken != ""
	}), mock.Anything).Return(false, nil).Once()
	notifier.On("ReserveConfirmation", mock.Anything, mock.Anything, "test@example.com",
		mock.MatchedBy(func(link string) bool { return link != "" })).Return(nil).Once()
	notifier.On("CommitConfirmation", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSubscriptionService(subStore, repoStore, ghClient, notifier, newTestURLs(), tokenGen)
	require.NoError(t, svc.Subscribe(context.Background(), "test@example.com", "golang/go"))

	subStore.AssertNumberOfCalls(t, "CreateForSaga", 1)
	notifier.AssertNumberOfCalls(t, "CommitConfirmation", 1)
	subStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	notifier.AssertExpectations(t)
	tokenGen.AssertExpectations(t)
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	svc := NewSubscriptionService(
		new(mockSubscriptionStore), new(mockRepoStore), new(mockGitHubClient),
		new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
	)
	require.ErrorIs(t, svc.Subscribe(context.Background(), "bad-email", "golang/go"), ErrInvalidEmail)
}

func TestSubscribe_InvalidRepo(t *testing.T) {
	svc := NewSubscriptionService(
		new(mockSubscriptionStore), new(mockRepoStore), new(mockGitHubClient),
		new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
	)
	require.ErrorIs(t, svc.Subscribe(context.Background(), "test@example.com", "wrongformat"), ErrInvalidRepo)
}

func TestSubscribe_RepoNotFound(t *testing.T) {
	ghClient := new(mockGitHubClient)
	ghClient.On("RepositoryExists", mock.Anything, "owner", "repo").Return(false, nil).Once()

	svc := NewSubscriptionService(
		new(mockSubscriptionStore), new(mockRepoStore), ghClient,
		new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
	)
	require.ErrorIs(t, svc.Subscribe(context.Background(), "test@example.com", "owner/repo"), ErrRepoNotFound)
	ghClient.AssertExpectations(t)
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	subStore := new(mockSubscriptionStore)
	repoStore := new(mockRepoStore)
	ghClient := new(mockGitHubClient)
	tokenGen := new(mockTokenGenerator)

	ghClient.On("RepositoryExists", mock.Anything, "golang", "go").Return(true, nil).Once()
	repoStore.On("FindOrCreate", mock.Anything, "golang", "go", "golang/go").
		Return(&repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil).Once()
	tokenGen.On("Generate").Return("confirm-token", nil).Once()
	tokenGen.On("Generate").Return("unsub-token", nil).Once()
	subStore.On("CreateForSaga", mock.Anything, mock.Anything, mock.Anything).Return(true, nil).Once()

	notifier := new(mockNotifier)
	svc := NewSubscriptionService(subStore, repoStore, ghClient, notifier, newTestURLs(), tokenGen)
	require.ErrorIs(t, svc.Subscribe(context.Background(), "test@example.com", "golang/go"), ErrAlreadySubscribed)

	subStore.AssertNumberOfCalls(t, "CreateForSaga", 1)
	notifier.AssertNotCalled(t, "ReserveConfirmation")
	subStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
	ghClient.AssertExpectations(t)
	tokenGen.AssertExpectations(t)
}

func TestSubscribe_ReSubscribeAfterUnsubscribe(t *testing.T) {
	subStore := new(mockSubscriptionStore)
	repoStore := new(mockRepoStore)
	ghClient := new(mockGitHubClient)
	notifier := new(mockNotifier)
	tokenGen := new(mockTokenGenerator)

	ghClient.On("RepositoryExists", mock.Anything, "golang", "go").Return(true, nil).Once()
	repoStore.On("FindOrCreate", mock.Anything, "golang", "go", "golang/go").
		Return(&repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil).Once()
	tokenGen.On("Generate").Return("new-confirm-token", nil).Once()
	tokenGen.On("Generate").Return("new-unsub-token", nil).Once()
	subStore.On("CreateForSaga", mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Once()
	notifier.On("ReserveConfirmation", mock.Anything, mock.Anything, "test@example.com",
		mock.MatchedBy(func(link string) bool { return link != "" })).Return(nil).Once()
	notifier.On("CommitConfirmation", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSubscriptionService(subStore, repoStore, ghClient, notifier, newTestURLs(), tokenGen)
	require.NoError(t, svc.Subscribe(context.Background(), "test@example.com", "golang/go"))

	subStore.AssertExpectations(t)
	notifier.AssertExpectations(t)
	tokenGen.AssertExpectations(t)
}

func TestSubscribe_CommitPivotFailureDoesNotCompensate(t *testing.T) {
	subStore := new(mockSubscriptionStore)
	repoStore := new(mockRepoStore)
	ghClient := new(mockGitHubClient)
	notifier := new(mockNotifier)
	tokenGen := new(mockTokenGenerator)

	ghClient.On("RepositoryExists", mock.Anything, "golang", "go").Return(true, nil).Once()
	repoStore.On("FindOrCreate", mock.Anything, "golang", "go", "golang/go").
		Return(&repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil).Once()
	tokenGen.On("Generate").Return("confirm-token", nil).Once()
	tokenGen.On("Generate").Return("unsub-token", nil).Once()
	subStore.On("CreateForSaga", mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Once()
	notifier.On("ReserveConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	notifier.On("CommitConfirmation", mock.Anything, mock.Anything).Return(errBoom)

	svc := NewSubscriptionService(subStore, repoStore, ghClient, notifier, newTestURLs(), tokenGen)
	require.Error(t, svc.Subscribe(context.Background(), "test@example.com", "golang/go"))

	notifier.AssertNotCalled(t, "CancelConfirmation")
	subStore.AssertNotCalled(t, "CancelBySaga")
	subStore.AssertExpectations(t)
}

func TestSubscribe_ReserveFailureCompensatesSubscriptionOnly(t *testing.T) {
	subStore := new(mockSubscriptionStore)
	repoStore := new(mockRepoStore)
	ghClient := new(mockGitHubClient)
	notifier := new(mockNotifier)
	tokenGen := new(mockTokenGenerator)

	ghClient.On("RepositoryExists", mock.Anything, "golang", "go").Return(true, nil).Once()
	repoStore.On("FindOrCreate", mock.Anything, "golang", "go", "golang/go").
		Return(&repo.Repository{ID: 1, FullName: "golang/go", Owner: "golang", Name: "go"}, nil).Once()
	tokenGen.On("Generate").Return("confirm-token", nil).Once()
	tokenGen.On("Generate").Return("unsub-token", nil).Once()
	subStore.On("CreateForSaga", mock.Anything, mock.Anything, mock.Anything).Return(false, nil).Once()
	notifier.On("ReserveConfirmation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errBoom).Once()
	subStore.On("CancelBySaga", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewSubscriptionService(subStore, repoStore, ghClient, notifier, newTestURLs(), tokenGen)
	require.Error(t, svc.Subscribe(context.Background(), "test@example.com", "golang/go"))

	subStore.AssertCalled(t, "CancelBySaga", mock.Anything, mock.Anything)
	notifier.AssertNotCalled(t, "CancelConfirmation")
	notifier.AssertNotCalled(t, "CommitConfirmation")
	subStore.AssertExpectations(t)
	notifier.AssertExpectations(t)
}

func TestConfirm(t *testing.T) {
	cases := []struct {
		name              string
		token             string
		findResult        *Subscription
		findErr           error
		expectedErr       error
		expectConfirmCall bool
	}{
		{
			name:              "success",
			token:             "valid-token",
			findResult:        &Subscription{ID: 1, ConfirmToken: "valid-token", Confirmed: false},
			expectedErr:       nil,
			expectConfirmCall: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectedErr: ErrInvalidToken,
		},
		{
			name:        "token not found",
			token:       "unknown-token",
			findResult:  nil,
			expectedErr: ErrTokenNotFound,
		},
		{
			name:              "already confirmed",
			token:             "already-confirmed-token",
			findResult:        &Subscription{ID: 1, ConfirmToken: "already-confirmed-token", Confirmed: true},
			expectedErr:       nil,
			expectConfirmCall: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subStore := new(mockSubscriptionStore)
			if tc.token != "" {
				subStore.On("FindByConfirmToken", mock.Anything, tc.token).
					Return(tc.findResult, tc.findErr).Once()
			}
			if tc.expectConfirmCall {
				subStore.On("ConfirmByToken", mock.Anything, tc.token).Return(nil).Once()
			}

			svc := NewSubscriptionService(
				subStore, new(mockRepoStore), new(mockGitHubClient),
				new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
			)
			err := svc.Confirm(context.Background(), tc.token)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
			subStore.AssertExpectations(t)
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	cases := []struct {
		name        string
		token       string
		findResult  *Subscription
		findErr     error
		expectedErr error
	}{
		{
			name:        "success",
			token:       "valid-token",
			findResult:  &Subscription{ID: 1, UnsubscribeToken: "valid-token", Active: true},
			expectedErr: nil,
		},
		{
			name:        "empty token",
			token:       "",
			expectedErr: ErrInvalidToken,
		},
		{
			name:        "token not found",
			token:       "unknown-token",
			findResult:  nil,
			expectedErr: ErrTokenNotFound,
		},
		{
			name:        "already inactive",
			token:       "inactive-token",
			findResult:  &Subscription{ID: 1, UnsubscribeToken: "inactive-token", Active: false},
			expectedErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subStore := new(mockSubscriptionStore)
			if tc.token != "" {
				subStore.On("FindByUnsubscribeToken", mock.Anything, tc.token).
					Return(tc.findResult, tc.findErr).Once()
			}
			if tc.findResult != nil && tc.findResult.Active {
				subStore.On("DeactivateByToken", mock.Anything, tc.token).Return(nil).Once()
			}

			svc := NewSubscriptionService(
				subStore, new(mockRepoStore), new(mockGitHubClient),
				new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
			)
			err := svc.Unsubscribe(context.Background(), tc.token)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
			subStore.AssertExpectations(t)
		})
	}
}

func TestGetSubscriptionsByEmail(t *testing.T) {
	tag := "v1.0.0"

	cases := []struct {
		name        string
		email       string
		subsResult  []Subscription
		subsErr     error
		repoResult  *repo.Repository
		repoErr     error
		expectedErr error
		expectedLen int
	}{
		{
			name:  "success",
			email: "test@example.com",
			subsResult: []Subscription{
				{ID: 1, RepositoryID: 1, Email: "test@example.com", Confirmed: true},
			},
			repoResult:  &repo.Repository{ID: 1, FullName: "golang/go", LastSeenTag: &tag},
			expectedLen: 1,
		},
		{
			name:        "invalid email",
			email:       "bad-email",
			expectedErr: ErrInvalidEmail,
		},
		{
			name:  "repo not found",
			email: "test@example.com",
			subsResult: []Subscription{
				{ID: 1, RepositoryID: 99, Email: "test@example.com", Confirmed: true},
			},
			repoResult:  nil,
			expectedLen: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subStore := new(mockSubscriptionStore)
			repoStore := new(mockRepoStore)

			if !errors.Is(tc.expectedErr, ErrInvalidEmail) {
				subStore.On("GetByEmail", mock.Anything, tc.email).
					Return(tc.subsResult, tc.subsErr).Once()
				for _, sub := range tc.subsResult {
					repoStore.On("GetByID", mock.Anything, sub.RepositoryID).
						Return(tc.repoResult, tc.repoErr).Once()
				}
			}

			svc := NewSubscriptionService(
				subStore, repoStore, new(mockGitHubClient),
				new(mockNotifier), newTestURLs(), new(mockTokenGenerator),
			)
			result, err := svc.GetSubscriptionsByEmail(context.Background(), tc.email)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, result, tc.expectedLen)
			subStore.AssertExpectations(t)
			repoStore.AssertExpectations(t)
		})
	}
}
