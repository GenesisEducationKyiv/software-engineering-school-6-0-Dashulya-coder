package integration_test

import (
	"context"

	"github.com/stretchr/testify/mock"
)

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

type stubNotifier struct {
	reserveErr  error
	commitErr   error
	cancelCalls int
}

func (s *stubNotifier) ReserveConfirmation(_ context.Context, _, _, _ string) error {
	return s.reserveErr
}

func (s *stubNotifier) CommitConfirmation(_ context.Context, _ string) error {
	return s.commitErr
}

func (s *stubNotifier) CancelConfirmation(_ context.Context, _ string) error {
	s.cancelCalls++
	return nil
}
