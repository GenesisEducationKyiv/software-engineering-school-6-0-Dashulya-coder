package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	httphandlers "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	httprouter "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/mailer"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/token"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

const dbPingTimeout = 5 * time.Second

var testDB *sql.DB

type stubGitHubClient struct {
	repoExists bool
	repoErr    error
}

func (s *stubGitHubClient) RepositoryExists(_ context.Context, _, _ string) (bool, error) {
	return s.repoExists, s.repoErr
}

func (s *stubGitHubClient) GetLatestRelease(_ context.Context, _, _ string) (string, string, error) {
	return "", "", nil
}

type noopMailer struct{}

func (m *noopMailer) SendConfirmation(_, _ string) error        { return nil }
func (m *noopMailer) SendNewRelease(_, _, _, _, _ string) error { return nil }

var _ github.Client = (*stubGitHubClient)(nil)
var _ mailer.Mailer = (*noopMailer)(nil)

func TestMain(tm *testing.M) {
	if os.Getenv("INTEGRATION") == "" {
		os.Exit(0)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	if err := runMigrations(dbURL); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	db, err := connectDB(dbURL)
	if err != nil {
		slog.Error("connect db failed", "error", err)
		os.Exit(1)
	}

	testDB = db

	code := tm.Run()

	if err := db.Close(); err != nil {
		slog.Error("close db failed", "error", err)
	}

	os.Exit(code)
}

func runMigrations(dbURL string) error {
	_, f, _, _ := runtime.Caller(0)
	dir := filepath.Clean(filepath.Join(filepath.Dir(f), "..", "..", "migrations"))

	m, err := migrate.New("file://"+dir, dbURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func connectDB(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

func newServer(t *testing.T, gh github.Client) *httptest.Server {
	t.Helper()

	subRepo := repository.NewSubscriptionRepository(testDB)
	repoRepo := repository.NewGitHubRepository(testDB)
	urls := urlbuilder.New("http://test")
	svc := subscription.NewSubscriptionService(
		subRepo, repoRepo, gh, &noopMailer{}, urls, token.New(),
	)
	handler := httphandlers.NewSubscriptionHandler(svc)
	srv := httptest.NewServer(httprouter.New(handler))
	t.Cleanup(srv.Close)

	return srv
}

func truncateTables(t *testing.T) {
	t.Helper()

	const q = "TRUNCATE TABLE subscriptions, repositories RESTART IDENTITY CASCADE"

	if _, err := testDB.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func insertRepo(t *testing.T, fullName, owner, name string) int64 {
	t.Helper()

	const q = `INSERT INTO repositories (full_name, owner, name) VALUES ($1, $2, $3) RETURNING id`

	var id int64
	if err := testDB.QueryRowContext(context.Background(), q, fullName, owner, name).Scan(&id); err != nil {
		t.Fatalf("insert repo: %v", err)
	}

	return id
}

func insertSubscription(t *testing.T, email string, repoID int64, confirmToken, unsubToken string) {
	t.Helper()

	const q = `
		INSERT INTO subscriptions (email, repository_id, confirm_token, unsubscribe_token)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := testDB.ExecContext(context.Background(), q, email, repoID, confirmToken, unsubToken); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
}

func confirmSubscriptionInDB(t *testing.T, confirmToken string) {
	t.Helper()

	const q = `UPDATE subscriptions SET confirmed = TRUE WHERE confirm_token = $1`

	if _, err := testDB.ExecContext(context.Background(), q, confirmToken); err != nil {
		t.Fatalf("confirm subscription: %v", err)
	}
}

func doGet(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("new GET request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}

	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close response body: %v", err)
		}
	})

	return resp
}

func doPost(t *testing.T, client *http.Client, url, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodPost, url, strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("new POST request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}

	t.Cleanup(func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("close response body: %v", err)
		}
	})

	return resp
}

func TestPostSubscribe(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		ghExists   bool
		wantStatus int
	}{
		{
			name:       "valid subscription",
			body:       `{"email":"user@example.com","repo":"cli/cli"}`,
			ghExists:   true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid email format",
			body:       `{"email":"not-an-email","repo":"cli/cli"}`,
			ghExists:   false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid repo format",
			body:       `{"email":"user@example.com","repo":"noslash"}`,
			ghExists:   false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repo not found on github",
			body:       `{"email":"user@example.com","repo":"owner/nonexistent"}`,
			ghExists:   false,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { truncateTables(t) })

			srv := newServer(t, &stubGitHubClient{repoExists: tc.ghExists})

			resp := doPost(t, srv.Client(), srv.URL+"/api/subscribe", tc.body)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestPostSubscribeDuplicate(t *testing.T) {
	t.Cleanup(func() { truncateTables(t) })

	srv := newServer(t, &stubGitHubClient{repoExists: true})
	body := `{"email":"user@example.com","repo":"cli/cli"}`

	first := doPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first subscribe: expected %d, got %d", http.StatusOK, first.StatusCode)
	}

	second := doPost(t, srv.Client(), srv.URL+"/api/subscribe", body)
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("second subscribe: expected %d, got %d", http.StatusConflict, second.StatusCode)
	}
}

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
			name: "valid token",
			setup: func(t *testing.T) {
				repoID := insertRepo(t, "cli/cli", "cli", "cli")
				insertSubscription(t, "user@example.com", repoID, confirmToken, unsubToken)
			},
			urlToken:   confirmToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "token not found",
			setup:      func(_ *testing.T) {},
			urlToken:   "does-not-exist",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { truncateTables(t) })

			srv := newServer(t, &stubGitHubClient{repoExists: false})
			tc.setup(t)

			resp := doGet(t, srv.Client(), srv.URL+"/api/confirm/"+tc.urlToken)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

func TestGetUnsubscribe(t *testing.T) {
	const (
		confirmToken = "integ-confirm-token-xyz"
		unsubToken   = "integ-unsub-token-xyz"
	)

	cases := []struct {
		name       string
		setup      func(t *testing.T)
		urlToken   string
		wantStatus int
	}{
		{
			name: "valid token",
			setup: func(t *testing.T) {
				repoID := insertRepo(t, "cli/cli", "cli", "cli")
				insertSubscription(t, "user@example.com", repoID, confirmToken, unsubToken)
			},
			urlToken:   unsubToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "token not found",
			setup:      func(_ *testing.T) {},
			urlToken:   "no-such-token",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { truncateTables(t) })

			srv := newServer(t, &stubGitHubClient{repoExists: false})
			tc.setup(t)

			resp := doGet(t, srv.Client(), srv.URL+"/api/unsubscribe/"+tc.urlToken)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

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
			name: "confirmed subscription returned",
			setup: func(t *testing.T) {
				repoID := insertRepo(t, "cli/cli", "cli", "cli")
				insertSubscription(t, "user@example.com", repoID, confirmToken, unsubToken)
				confirmSubscriptionInDB(t, confirmToken)
			},
			email:      "user@example.com",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "no subscriptions for email",
			setup:      func(_ *testing.T) {},
			email:      "nobody@example.com",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "invalid email",
			setup:      func(_ *testing.T) {},
			email:      "not-an-email",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty email",
			setup:      func(_ *testing.T) {},
			email:      "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { truncateTables(t) })

			srv := newServer(t, &stubGitHubClient{repoExists: false})
			tc.setup(t)

			resp := doGet(t, srv.Client(), srv.URL+"/api/subscriptions?email="+tc.email)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}

			if tc.wantStatus == http.StatusOK {
				var subs []json.RawMessage
				if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
					t.Fatalf("decode subscriptions response: %v", err)
				}

				if len(subs) != tc.wantCount {
					t.Fatalf("expected %d subscription(s), got %d", tc.wantCount, len(subs))
				}
			}
		})
	}
}
