package testenv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const dbPingTimeout = 5 * time.Second

func RunMigrations(dbURL string) error {
	pcs := make([]uintptr, 1)
	runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs)
	frame, _ := frames.Next()
	dir := filepath.Clean(filepath.Join(filepath.Dir(frame.File), "..", "..", "..", "migrations"))

	m, err := migrate.New("file://"+dir, dbURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func ConnectDB(dbURL string) (*sql.DB, error) {
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
