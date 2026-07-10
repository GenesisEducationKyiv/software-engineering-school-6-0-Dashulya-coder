package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
)

var ErrNotFound = errors.New("record not found")

type SubscriptionRepository interface {
	CreateForSaga(ctx context.Context, sub *subscription.Subscription, sagaID string) (bool, error)
	CancelBySaga(ctx context.Context, sagaID string) error
	FindByConfirmToken(ctx context.Context, token string) (*subscription.Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (*subscription.Subscription, error)
	GetByEmail(ctx context.Context, email string) ([]subscription.Subscription, error)
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

func (r *SubscriptionRepositoryImpl) CreateForSaga(
	ctx context.Context,
	sub *subscription.Subscription,
	sagaID string,
) (bool, error) {
	query := `
		INSERT INTO subscriptions (
			email, repository_id, confirm_token, unsubscribe_token,
			saga_id, confirmed, active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, FALSE, TRUE, NOW(), NOW())
		ON CONFLICT (email, repository_id) DO UPDATE
		SET confirm_token     = EXCLUDED.confirm_token,
		    unsubscribe_token = EXCLUDED.unsubscribe_token,
		    saga_id           = EXCLUDED.saga_id,
		    confirmed         = FALSE,
		    active            = TRUE,
		    updated_at        = NOW()
		WHERE subscriptions.confirmed = FALSE
		   OR subscriptions.active   = FALSE
		RETURNING id, confirm_token, unsubscribe_token
	`

	err := r.db.QueryRowContext(
		ctx, query,
		sub.Email,
		sub.RepositoryID,
		sub.ConfirmToken,
		sub.UnsubscribeToken,
		sagaID,
	).Scan(&sub.ID, &sub.ConfirmToken, &sub.UnsubscribeToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

func (r *SubscriptionRepositoryImpl) CancelBySaga(ctx context.Context, sagaID string) error {
	const query = `DELETE FROM subscriptions WHERE saga_id = $1`

	if _, err := r.db.ExecContext(ctx, query, sagaID); err != nil {
		return err
	}

	return nil
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
