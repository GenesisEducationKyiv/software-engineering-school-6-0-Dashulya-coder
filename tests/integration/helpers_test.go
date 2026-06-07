package integration_test

import (
	"database/sql"
	"log/slog"
	"os"
	"testing"

	"github.com/Dashulya-coder/CaseTaskNotifier/tests/pkg/testenv"
)

var testDB *sql.DB

func TestMain(tm *testing.M) {
	if os.Getenv("INTEGRATION") == "" {
		os.Exit(0)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	if err := testenv.RunMigrations(dbURL); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	db, err := testenv.ConnectDB(dbURL)
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
