package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

// ErrNotFound is returned by repository methods when the requested record does not exist.
// All implementations must return this error (not sql.ErrNoRows or similar) so callers
// remain independent of the storage technology.
var ErrNotFound = errors.New("record not found")

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *subscription.Subscription) error
	FindByConfirmToken(ctx context.Context, token string) (*subscription.Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*subscription.Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]subscription.Subscription, error)
	ExistsByEmailAndRepo(ctx context.Context, email string, repoID int64) (bool, error)
	ConfirmByToken(ctx context.Context, token string) error
	DeactivateByToken(ctx context.Context, token string) error
	GetAllConfirmedActive(ctx context.Context) ([]subscription.Subscription, error)
	GetConfirmedActiveByRepo(ctx context.Context, repoID int64) ([]subscription.Subscription, error)
}

type SubscriptionRepositoryImpl struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepositoryImpl {
	return &SubscriptionRepositoryImpl{db: db}
}

func (r *SubscriptionRepositoryImpl) Create(
	ctx context.Context,
	sub *subscription.Subscription,
) error {
	query := `
		INSERT INTO subscriptions (
			email,
			repository_id,
			confirm_token,
			unsubscribe_token
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, confirmed, active, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		sub.Email,
		sub.RepositoryID,
		sub.ConfirmToken,
		sub.UnsubscribeToken,
	).Scan(
		&sub.ID,
		&sub.Confirmed,
		&sub.Active,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
}

func (r *SubscriptionRepositoryImpl) FindByConfirmToken(
	ctx context.Context,
	token string,
) (*subscription.Subscription, error) {
	query := `
		SELECT id, email, repository_id, confirmed, active,
		       confirm_token, unsubscribe_token, created_at, updated_at
		FROM subscriptions
		WHERE confirm_token = $1
	`

	var sub subscription.Subscription

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&sub.ID,
		&sub.Email,
		&sub.RepositoryID,
		&sub.Confirmed,
		&sub.Active,
		&sub.ConfirmToken,
		&sub.UnsubscribeToken,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &sub, nil
}

func (r *SubscriptionRepositoryImpl) FindByUnsubscribeToken(
	ctx context.Context,
	token string,
) (*subscription.Subscription, error) {
	query := `
		SELECT id, email, repository_id, confirmed, active,
		       confirm_token, unsubscribe_token, created_at, updated_at
		FROM subscriptions
		WHERE unsubscribe_token = $1
	`

	var sub subscription.Subscription

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&sub.ID,
		&sub.Email,
		&sub.RepositoryID,
		&sub.Confirmed,
		&sub.Active,
		&sub.ConfirmToken,
		&sub.UnsubscribeToken,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &sub, nil
}

func (r *SubscriptionRepositoryImpl) GetByEmail(
	ctx context.Context,
	email string,
) ([]subscription.Subscription, error) {
	query := `
		SELECT id, email, repository_id, confirmed, active,
		       confirm_token, unsubscribe_token, created_at, updated_at
		FROM subscriptions
		WHERE email = $1 AND active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var subs []subscription.Subscription

	for rows.Next() {
		var sub subscription.Subscription

		err := rows.Scan(
			&sub.ID,
			&sub.Email,
			&sub.RepositoryID,
			&sub.Confirmed,
			&sub.Active,
			&sub.ConfirmToken,
			&sub.UnsubscribeToken,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *SubscriptionRepositoryImpl) ExistsByEmailAndRepo(
	ctx context.Context,
	email string,
	repoID int64,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM subscriptions
			WHERE email = $1 AND repository_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email, repoID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *SubscriptionRepositoryImpl) ConfirmByToken(
	ctx context.Context,
	token string,
) error {
	query := `
		UPDATE subscriptions
		SET confirmed = TRUE,
		    updated_at = NOW()
		WHERE confirm_token = $1
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SubscriptionRepositoryImpl) DeactivateByToken(
	ctx context.Context,
	token string,
) error {
	query := `
		UPDATE subscriptions
		SET active = FALSE,
		    updated_at = NOW()
		WHERE unsubscribe_token = $1
	`

	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SubscriptionRepositoryImpl) GetAllConfirmedActive(
	ctx context.Context,
) ([]subscription.Subscription, error) {
	query := `
		SELECT id, email, repository_id, confirmed, active,
		       confirm_token, unsubscribe_token, created_at, updated_at
		FROM subscriptions
		WHERE confirmed = TRUE AND active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var subs []subscription.Subscription

	for rows.Next() {
		var sub subscription.Subscription

		err := rows.Scan(
			&sub.ID,
			&sub.Email,
			&sub.RepositoryID,
			&sub.Confirmed,
			&sub.Active,
			&sub.ConfirmToken,
			&sub.UnsubscribeToken,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *SubscriptionRepositoryImpl) GetConfirmedActiveByRepo(
	ctx context.Context,
	repoID int64,
) ([]subscription.Subscription, error) {
	query := `
		SELECT id, email, repository_id, confirmed, active,
		       confirm_token, unsubscribe_token, created_at, updated_at
		FROM subscriptions
		WHERE repository_id = $1
		  AND confirmed = TRUE
		  AND active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("failed to close rows", "error", err)
		}
	}()

	var subs []subscription.Subscription

	for rows.Next() {
		var sub subscription.Subscription

		err := rows.Scan(
			&sub.ID,
			&sub.Email,
			&sub.RepositoryID,
			&sub.Confirmed,
			&sub.Active,
			&sub.ConfirmToken,
			&sub.UnsubscribeToken,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}
