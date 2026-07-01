package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Dashulya-coder/CaseTaskNotifier/notifier/internal/delivery"
)

type Deliveries struct {
	db *sql.DB
}

func NewDeliveries(db *sql.DB) *Deliveries {
	return &Deliveries{db: db}
}

func (d *Deliveries) Reserve(ctx context.Context, del delivery.Delivery) (bool, error) {
	const query = `INSERT INTO confirmation_deliveries (saga_id, email, confirm_url, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (saga_id) DO NOTHING`

	res, err := d.db.ExecContext(ctx, query,
		del.SagaID, del.Email, del.ConfirmURL, string(delivery.StatusPending))
	if err != nil {
		return false, fmt.Errorf("reserve confirmation: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return affected == 1, nil
}

func (d *Deliveries) Get(ctx context.Context, sagaID string) (*delivery.Delivery, error) {
	const query = `SELECT saga_id, email, confirm_url, status
		FROM confirmation_deliveries WHERE saga_id = $1`

	var out delivery.Delivery
	var status string

	err := d.db.QueryRowContext(ctx, query, sagaID).
		Scan(&out.SagaID, &out.Email, &out.ConfirmURL, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get confirmation: %w", err)
	}

	out.Status = delivery.DeliveryStatus(status)
	return &out, nil
}

func (d *Deliveries) MarkSent(ctx context.Context, sagaID string) error {
	const query = `UPDATE confirmation_deliveries
		SET status = $2, updated_at = now()
		WHERE saga_id = $1 AND status = $3`

	if _, err := d.db.ExecContext(ctx, query,
		sagaID, string(delivery.StatusSent), string(delivery.StatusPending)); err != nil {
		return fmt.Errorf("mark confirmation sent: %w", err)
	}
	return nil
}

func (d *Deliveries) Cancel(ctx context.Context, sagaID string) error {
	const query = `UPDATE confirmation_deliveries
		SET status = $2, updated_at = now()
		WHERE saga_id = $1 AND status = $3`

	if _, err := d.db.ExecContext(ctx, query,
		sagaID, string(delivery.StatusCanceled), string(delivery.StatusPending)); err != nil {
		return fmt.Errorf("cancel confirmation: %w", err)
	}
	return nil
}

var _ delivery.Deliveries = (*Deliveries)(nil)
