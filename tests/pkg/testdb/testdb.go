package testdb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TruncateTables(t *testing.T, db *sql.DB) {
	t.Helper()

	const q = "TRUNCATE TABLE subscriptions, repositories RESTART IDENTITY CASCADE"

	_, err := db.ExecContext(context.Background(), q)
	require.NoError(t, err, "truncate tables")
}

func InsertRepo(t *testing.T, db *sql.DB, fullName, owner, name string) int64 {
	t.Helper()

	const q = `INSERT INTO repositories (full_name, owner, name) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	err := db.QueryRowContext(context.Background(), q, fullName, owner, name).Scan(&id)
	require.NoError(t, err, "insert repo")

	return id
}

func InsertSubscription(
	t *testing.T,
	db *sql.DB,
	email string,
	repoID int64,
	confirmToken, unsubToken string,
) {
	t.Helper()

	const q = `
		INSERT INTO subscriptions (email, repository_id, confirm_token, unsubscribe_token)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.ExecContext(context.Background(), q, email, repoID, confirmToken, unsubToken)
	require.NoError(t, err, "insert subscription")
}

func ConfirmSubscription(t *testing.T, db *sql.DB, confirmToken string) {
	t.Helper()

	const q = `UPDATE subscriptions SET confirmed = TRUE WHERE confirm_token = $1`

	_, err := db.ExecContext(context.Background(), q, confirmToken)
	require.NoError(t, err, "confirm subscription")
}
