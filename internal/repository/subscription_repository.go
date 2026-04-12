package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/model"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	FindByConfirmToken(ctx context.Context, token string) (*model.Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*model.Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]model.Subscription, error)
	ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error)
	ConfirmByToken(ctx context.Context, token string) error
	DeactivateByToken(ctx context.Context, token string) error
	GetAllConfirmedActive(ctx context.Context) ([]model.Subscription, error)
	GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]model.Subscription, error)
}

type subscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	return errors.New("not implemented")
}

func (r *subscriptionRepository) FindByConfirmToken(ctx context.Context, token string) (*model.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (r *subscriptionRepository) FindByUnsubscribeToken(ctx context.Context, token string) (*model.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (r *subscriptionRepository) GetByEmail(ctx context.Context, email string) ([]model.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (r *subscriptionRepository) ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error) {
	return false, errors.New("not implemented")
}

func (r *subscriptionRepository) ConfirmByToken(ctx context.Context, token string) error {
	return errors.New("not implemented")
}

func (r *subscriptionRepository) DeactivateByToken(ctx context.Context, token string) error {
	return errors.New("not implemented")
}

func (r *subscriptionRepository) GetAllConfirmedActive(ctx context.Context) ([]model.Subscription, error) {
	return nil, errors.New("not implemented")
}

func (r *subscriptionRepository) GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]model.Subscription, error) {
	return nil, errors.New("not implemented")
}
