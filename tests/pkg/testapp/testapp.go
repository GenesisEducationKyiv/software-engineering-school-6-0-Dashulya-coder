package testapp

import (
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/github"
	httphandlers "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/handlers"
	httprouter "github.com/Dashulya-coder/CaseTaskNotifier/internal/http/router"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/repository"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/subscription"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/token"
	"github.com/Dashulya-coder/CaseTaskNotifier/internal/urlbuilder"
)

func NewServer(
	t *testing.T,
	db *sql.DB,
	gh github.Client,
	notifier subscription.ConfirmationNotifier,
) *httptest.Server {
	t.Helper()

	subRepo := repository.NewSubscriptionRepository(db)
	repoRepo := repository.NewGitHubRepository(db)
	urls := urlbuilder.New("http://test")
	svc := subscription.NewSubscriptionService(
		subRepo, repoRepo, gh, notifier, urls, token.New(),
	)
	handler := httphandlers.NewSubscriptionHandler(svc)
	srv := httptest.NewServer(httprouter.New(handler))
	t.Cleanup(srv.Close)

	return srv
}
