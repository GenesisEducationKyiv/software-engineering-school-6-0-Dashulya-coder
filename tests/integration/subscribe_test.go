package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testapp"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testdb"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testhttp"
)

func TestPostSubscribe(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		ghExists   bool
		wantStatus int
	}{
		{
			name:       "OK",
			body:       `{"email":"user@example.com","repo":"cli/cli"}`,
			ghExists:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Error_InvalidEmail",
			body:       `{"email":"not-an-email","repo":"cli/cli"}`,
			ghExists:   false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Error_InvalidRepo",
			body:       `{"email":"user@example.com","repo":"noslash"}`,
			ghExists:   false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Error_RepoNotFound",
			body:       `{"email":"user@example.com","repo":"owner/nonexistent"}`,
			ghExists:   false,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

			gh := new(mockGitHubClient)
			gh.On("RepositoryExists", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.ghExists, nil)

			ml := new(mockMailer)
			ml.On("SendConfirmation", mock.Anything, mock.Anything).Return(nil)

			srv := testapp.NewServer(t, testDB, gh, ml)

			resp := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", tc.body)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestPostSubscribe_Duplicate(t *testing.T) {
	const body = `{"email":"user@example.com","repo":"cli/cli"}`

	t.Run("SecondSubscribeBeforeConfirm_Refreshes", func(t *testing.T) {
		t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

		gh := new(mockGitHubClient)
		gh.On("RepositoryExists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		ml := new(mockMailer)
		ml.On("SendConfirmation", mock.Anything, mock.Anything).Return(nil)
		srv := testapp.NewServer(t, testDB, gh, ml)

		first := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		assert.Equal(t, http.StatusOK, first.StatusCode)

		second := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		assert.Equal(t, http.StatusOK, second.StatusCode)
	})

	t.Run("SubscribeAfterConfirm_ReturnsConflict", func(t *testing.T) {
		t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

		gh := new(mockGitHubClient)
		gh.On("RepositoryExists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		ml := new(mockMailer)
		ml.On("SendConfirmation", mock.Anything, mock.Anything).Return(nil)
		srv := testapp.NewServer(t, testDB, gh, ml)

		first := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		require.Equal(t, http.StatusOK, first.StatusCode)

		token := fetchConfirmToken(t, "user@example.com", "cli/cli")
		confirm := testhttp.DoGet(t, srv.Client(), srv.URL+"/api/confirm/"+token)
		require.Equal(t, http.StatusOK, confirm.StatusCode)

		second := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		assert.Equal(t, http.StatusConflict, second.StatusCode)
	})

	t.Run("SubscribeAfterUnsubscribe_Reactivates", func(t *testing.T) {
		t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

		gh := new(mockGitHubClient)
		gh.On("RepositoryExists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		ml := new(mockMailer)
		ml.On("SendConfirmation", mock.Anything, mock.Anything).Return(nil)
		srv := testapp.NewServer(t, testDB, gh, ml)

		first := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		require.Equal(t, http.StatusOK, first.StatusCode)

		confirmTok := fetchConfirmToken(t, "user@example.com", "cli/cli")
		confirm := testhttp.DoGet(t, srv.Client(), srv.URL+"/api/confirm/"+confirmTok)
		require.Equal(t, http.StatusOK, confirm.StatusCode)

		unsubTok := fetchUnsubscribeToken(t, "user@example.com", "cli/cli")
		unsub := testhttp.DoGet(t, srv.Client(), srv.URL+"/api/unsubscribe/"+unsubTok)
		require.Equal(t, http.StatusOK, unsub.StatusCode)

		second := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
		assert.Equal(t, http.StatusOK, second.StatusCode)
	})
}

func fetchConfirmToken(t *testing.T, email, repoFullName string) string {
	t.Helper()
	const q = `
		SELECT s.confirm_token
		FROM subscriptions s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.email = $1 AND r.full_name = $2
	`
	var tok string
	err := testDB.QueryRowContext(context.Background(), q, email, repoFullName).Scan(&tok)
	require.NoError(t, err)
	return tok
}

func fetchUnsubscribeToken(t *testing.T, email, repoFullName string) string {
	t.Helper()
	const q = `
		SELECT s.unsubscribe_token
		FROM subscriptions s
		JOIN repositories r ON r.id = s.repository_id
		WHERE s.email = $1 AND r.full_name = $2
	`
	var tok string
	err := testDB.QueryRowContext(context.Background(), q, email, repoFullName).Scan(&tok)
	require.NoError(t, err)
	return tok
}
