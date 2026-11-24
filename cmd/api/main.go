package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx as a database/sql driver.

	"github.com/W7SP/Job-Processor-API/internal/job"
)

func main() {
	// JSON logger writing to stdout (containers are expected to log to stdout).
	// SetDefault makes the package-level slog.Info(...) calls use this logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Before starting the project you need to set DATABASE_URL:
	// $env:DATABASE_URL = 'postgres://postgres:pass123@localhost:5432/jobs_db?sslmode=disable'
	// go run ./cmd/api
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		//databaseURL = "postgres://postgres:pass123@localhost:5432/jobs_db?sslmode=disable"
		slog.Error("DATABASE_URL must be set")
		os.Exit(1)
	}

	// Open creates a connection pool; PingContext checks that PostgreSQL is reachable.
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("failed to open database")
		os.Exit(1)
	}
	defer db.Close()

	connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(connectCtx)
	cancel()
	if err != nil {
		// Avoid logging the connection string, which contains the password.
		slog.Error("failed to connect to database; check DATABASE_URL and PostgreSQL")
		os.Exit(1) // slog has no Fatal: log the error, then exit ourselves
	}
	slog.Info("connected to PostgreSQL")

	// Ctrl+C (local development) or SIGTERM (containers) starts shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	postgresStore := job.NewPostgresStore(db)
	//store := job.NewMemoryStore()
	jobChannel := make(chan *job.Job, 3)
	// jobChan is the channel connecting handlers to workers.
	// Buffer of 3 means: up to 3 jobs can be waiting
	// without a worker ready, before the channel blocks.

	// Create the pool and set up 3 workers.
	//pool := job.NewWorkerPool(jobChannel, store, &job.SimulatedProcessor{})
	pool := job.NewWorkerPool(jobChannel, postgresStore, &job.SimulatedProcessor{})

	pool.LaunchRoutines(3)

	app := &application{
		//store:     store,
		jobStore: postgresStore,
		jobChan:  jobChannel,
	}

	// Creates a router (urls,py)
	r := chi.NewRouter()

	// Adds some middlewares like in settings.py in Django
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// The order matters, each middleware will be executed before every request

	r.Get("/healthz", healthHandler)      // On /healthz call healthHandler, this time specifically for GET requests
	r.Post("/jobs", app.createJobHandler) // On /jobs call app.createJobHandler
	r.Get("/jobs", app.getAllJobsHandler)
	r.Get("/jobs/{id}", app.getJobHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ListenAndServe blocks, so run it separately while main waits for a signal.
	// Buffering lets the server report its result even while main is shutting down.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server stopped", "error", err)
		}
	}
	stop() // Restore normal signal handling; a second Ctrl+C can force an exit.

	// Use a fresh context: the signal context is already canceled on shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		// Handlers may still be sending jobs. Exit without closing their channel.
		slog.Error("HTTP shutdown failed", "error", err)
		os.Exit(1)
	}

	// Shutdown succeeded: no HTTP handlers can send more jobs.
	close(jobChannel)
	slog.Info("waiting for workers to finish queued jobs")
	pool.Wait()
	slog.Info("shutdown complete")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type application struct {
	//store     job.Store
	jobStore job.Store
	jobChan  chan *job.Job
}
type createJobRequest struct {
	Payload string `json:"payload"`
}

// application Why create this? Why not a global store?
// Because later you can test with an application with a mocked store
// Also application has methods that you know depend on its fields

// createJobHandler Notice this receives the same things as a normal view BUT
// Now it also has access to the fields of application (store)
func (app *application) createJobHandler(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	j, err := app.jobStore.Create(r.Context(), req.Payload)
	if err != nil {
		// The client gets a generic message; the real cause goes to our logs.
		slog.Error("create job failed", "error", err)
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	// Queue the job. This blocks while the channel's three slots are full.
	select {
	case app.jobChan <- j: // queued, carry on
	case <-r.Context().Done(): // client left before there was room
		slog.Warn("job created but not queued", "job_id", j.ID, "error", r.Context().Err())
		http.Error(w, "server busy, try again", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(j)
}

func (app *application) getJobHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid job ID: must be a UUID", http.StatusBadRequest)
		return
	}

	j, err := app.jobStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, job.ErrNotFound) {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		slog.Error("get job failed", "job_id", id, "error", err)
		http.Error(w, "failed to get job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

func (app *application) getAllJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := app.jobStore.GetAll(r.Context())
	if err != nil {
		slog.Error("get all jobs failed", "error", err)
		http.Error(w, "failed to get jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
