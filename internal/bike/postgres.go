package bike

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

func (s *PostgresStore) DeleteAll(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bikes`)
	if err != nil {
		return fmt.Errorf("delete all bikes: %w", err)
	}
	return nil
}

func (s *PostgresStore) Create(ctx context.Context, bikeType string) (*Bike, error) {
	b := &Bike{
		ID:          uuid.NewString(),
		BikeType:    bikeType,
		BuildStatus: "in assembly...",
	}

	_, err := s.db.ExecContext(ctx, `INSERT INTO bikes (id, bike_type, build_status) VALUES ($1, $2, $3)`,
		b.ID, b.BikeType, b.BuildStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("insert bike: %w", err)
	}

	return b, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Bike, error) {
	var b Bike

	err := s.db.QueryRowContext(ctx, `SELECT id, bike_type, build_status FROM bikes WHERE id = $1`, id).Scan(&b.ID, &b.BikeType, &b.BuildStatus)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get bike: %w", err)
	}

	return &b, nil
}

func (s *PostgresStore) GetAll(ctx context.Context) (map[string]*Bike, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, bike_type, build_status FROM bikes`)
	if err != nil {
		return nil, fmt.Errorf("query bikes: %w", err)
	}
	defer rows.Close()

	bikes := make(map[string]*Bike)
	for rows.Next() {
		var b Bike
		if err := rows.Scan(&b.ID, &b.BikeType, &b.BuildStatus); err != nil {
			return nil, fmt.Errorf("scan bike: %w", err)
		}
		bikes[b.ID] = &b
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bikes: %w", err)
	}
	return bikes, nil
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id, status string) error {
	// Get a job by id and update its status
	result, err := s.db.ExecContext(ctx,
		`UPDATE bikes SET build_status = $1 WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update bike status: %w", err)
	}

	// Check if the job was found, because the above returns true even if no job with the id exists
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated bike count: %w", err)
	}
	if affected == 0 {
		// Basically if no job was affected means no job was found with this id
		return ErrNotFound
	}

	return nil
}
