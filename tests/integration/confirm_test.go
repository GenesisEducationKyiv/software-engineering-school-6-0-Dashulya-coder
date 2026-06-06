package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testapp"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testdb"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testhttp"
)

func TestGetConfirm(t *testing.T) {
	const (
		confirmToken = "integ-confirm-token-abc"
		unsubToken   = "integ-unsub-token-abc"
	)

	cases := []struct {
		name       string
		setup      func(t *testing.T)
		urlToken   string
		wantStatus int
	}{
		{
			name: "OK",
			setup: func(t *testing.T) {
				repoID := testdb.InsertRepo(t, testDB, "cli/cli", "cli", "cli")
				testdb.InsertSubscription(t, testDB, "user@example.com", repoID, confirmToken, unsubToken)
			},
			urlToken:   confirmToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Error_TokenNotFound",
			setup:      func(_ *testing.T) {},
			urlToken:   "does-not-exist",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

			srv := testapp.NewServer(t, testDB, new(mockGitHubClient), new(mockMailer))
			tc.setup(t)

			resp := testhttp.DoGet(t, srv.Client(), srv.URL+"/api/confirm/"+tc.urlToken)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
