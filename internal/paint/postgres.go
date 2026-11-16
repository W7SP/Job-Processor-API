package paint

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (ps *PostgresStore) Create(ctx context.Context, color string) (*Paint, error) {
	paint := &Paint{
		ID:       uuid.NewString(),
		Color:    color,
		Prepared: false,
	}

	_, err := ps.db.ExecContext(ctx, `INSERT INTO paints (id, color, prepared) VALUES ($1, $2, $3)`,
		paint.ID, paint.Color, paint.Prepared)

	if err != nil {
		return nil, fmt.Errorf("insert paint: %w", err)
	}

	return paint, nil
}

func (ps *PostgresStore) Get(ctx context.Context, id string) (*Paint, error) {
	var paint Paint
	err := ps.db.QueryRowContext(ctx, `SELECT id, color, prepared FROM paints WHERE id = $1`, id).
		Scan(&paint.ID, &paint.Color, &paint.Prepared)

	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get paint: %w", err)
	}

	return &paint, nil
}

func (ps *PostgresStore) GetAll(ctx context.Context) (map[string]*Paint, error) {
	var paints map[string]*Paint
	rows, err := ps.db.QueryContext(ctx, `SELECT id, color, prepared FROM paints`)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var paint Paint
		if err := rows.Scan(&paint.ID, &paint.Color, &paint.Prepared); err != nil {
			return nil, fmt.Errorf("scan paint: %w", err)
		}
		paints[paint.ID] = &paint
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate paint: %w", err)
	}
	return paints, nil
}

func (ps *PostgresStore) UpdateStatus(ctx context.Context, id string) error {
	result, err := ps.db.ExecContext(ctx, `UPDATE paints SET prepared = true WHERE id = $1`, id)

	if err != nil {
		return fmt.Errorf("update paint status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated paint count: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (ps *PostgresStore) DeleteAll(ctx context.Context) error {
	_, err := ps.db.ExecContext(ctx, `DELETE FROM paints`)
	if err != nil {
		return fmt.Errorf("delete all paints: %w", err)
	}
	return nil
}
