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

type mockMailer struct {
	mock.Mock
}

func (m *mockMailer) SendConfirmation(email, link string) error {
	args := m.Called(email, link)
	return args.Error(0)
}

func (m *mockMailer) SendNewRelease(email, repo, tag, name, url string) error {
	args := m.Called(email, repo, tag, name, url)
	return args.Error(0)
}
