package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testapp"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testdb"
	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testhttp"
)

func TestGetSubscriptions(t *testing.T) {
	const (
		confirmToken = "subs-confirm-token-abc"
		unsubToken   = "subs-unsub-token-abc"
	)

	cases := []struct {
		name       string
		setup      func(t *testing.T)
		email      string
		wantStatus int
		wantCount  int
	}{
		{
			name: "OK_ConfirmedSubscription",
			setup: func(t *testing.T) {
				repoID := testdb.InsertRepo(t, testDB, "cli/cli", "cli", "cli")
				testdb.InsertSubscription(t, testDB, "user@example.com", repoID, confirmToken, unsubToken)
				testdb.ConfirmSubscription(t, testDB, confirmToken)
			},
			email:      "user@example.com",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "OK_NoSubscriptions",
			setup:      func(_ *testing.T) {},
			email:      "nobody@example.com",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "Error_InvalidEmail",
			setup:      func(_ *testing.T) {},
			email:      "not-an-email",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Error_EmptyEmail",
			setup:      func(_ *testing.T) {},
			email:      "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

			srv := testapp.NewServer(t, testDB, new(mockGitHubClient), new(mockMailer))
			tc.setup(t)

			resp := testhttp.DoGet(t, srv.Client(), srv.URL+"/api/subscriptions?email="+tc.email)
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantStatus == http.StatusOK {
				var subs []json.RawMessage
				err := json.NewDecoder(resp.Body).Decode(&subs)
				require.NoError(t, err)
				assert.Len(t, subs, tc.wantCount)
			}
		})
	}
}
