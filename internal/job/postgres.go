package job

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

func (s *PostgresStore) Create(ctx context.Context, payload string) (*Job, error) {
	// Populate the job struct
	j := &Job{
		ID:      uuid.NewString(),
		Status:  "pending",
		Payload: payload,
	}

	// Insert the job into the database
	// Here if the caller passed context with timeout, if the timeout expires, the query will stop?
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (id, status, payload) VALUES ($1, $2, $3)`,
		j.ID, j.Status, j.Payload,
	)

	if err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}

	return j, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Job, error) {
	// Create an empty job
	var j Job

	// Make a SELECT query to the db asking for the job with provided id
	err := s.db.QueryRowContext(ctx,
		`SELECT id, status, payload FROM jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.Status, &j.Payload)
	// The SCAN part populates the j variable

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	return &j, nil
}

func (s *PostgresStore) GetAll(ctx context.Context) (map[string]*Job, error) {
	// Get all the rows (jobs) from the table
	rows, err := s.db.QueryContext(ctx, `SELECT id, status, payload FROM jobs`)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	// Release the data after done
	defer rows.Close()

	// Create an empty map of jobs to populate it with the rows from the db
	jobs := make(map[string]*Job)

	// rows.Next loops over all jobs and returns false when it finishes or an error occurs
	// Basically the for loop iterates until false
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.Status, &j.Payload); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		// Populate the jobs map
		jobs[j.ID] = &j
	}

	// This checks if there was an error during the loop
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, nil
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id, status string) error {
	// Get a job by id and update its status
	result, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = $1 WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}

	// Check if the job was found, because the above returns true even if no job with the id exists
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated job count: %w", err)
	}
	if affected == 0 {
		// Basically if no job was affected means no job was found with this id
		return ErrNotFound
	}

	return nil
}
