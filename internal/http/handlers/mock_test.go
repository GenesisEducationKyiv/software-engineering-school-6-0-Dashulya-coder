package handlers_test

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) Subscribe(ctx context.Context, email, repo string) error {
	return m.Called(ctx, email, repo).Error(0)
}

func (m *mockService) Confirm(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockService) Unsubscribe(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *mockService) GetSubscriptionsByEmail(
	ctx context.Context,
	email string,
) ([]subscription.SubscriptionView, error) {
	args := m.Called(ctx, email)
	return args.Get(0).([]subscription.SubscriptionView), args.Error(1)
}
