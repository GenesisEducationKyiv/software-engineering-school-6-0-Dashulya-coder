package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
)

type Ledger struct {
	db *sql.DB
}

func NewLedger(db *sql.DB) *Ledger {
	return &Ledger{db: db}
}

func (l *Ledger) Reserve(ctx context.Context, dedupKey string) (bool, error) {
	const query = `INSERT INTO sent_notifications (dedup_key) VALUES ($1)
		ON CONFLICT (dedup_key) DO NOTHING`

	res, err := l.db.ExecContext(ctx, query, dedupKey)
	if err != nil {
		return false, fmt.Errorf("reserve dedup key: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return affected == 1, nil
}

var _ delivery.Ledger = (*Ledger)(nil)
