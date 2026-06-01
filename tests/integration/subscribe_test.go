package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
	t.Cleanup(func() { testdb.TruncateTables(t, testDB) })

	gh := new(mockGitHubClient)
	gh.On("RepositoryExists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

	ml := new(mockMailer)
	ml.On("SendConfirmation", mock.Anything, mock.Anything).Return(nil)

	srv := testapp.NewServer(t, testDB, gh, ml)
	body := `{"email":"user@example.com","repo":"cli/cli"}`

	first := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
	assert.Equal(t, http.StatusOK, first.StatusCode)

	second := testhttp.DoPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
	assert.Equal(t, http.StatusConflict, second.StatusCode)
}
