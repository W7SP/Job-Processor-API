package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/W7SP/Job-Processor-API/internal/bike"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx as a database/sql driver.

	"github.com/W7SP/Job-Processor-API/internal/job"
)

func main() {
	// Before starting the project you need to set DATABASE_URL:
	// $env:DATABASE_URL = 'postgres://postgres:pass123@localhost:5432/jobs_db?sslmode=disable'
	// go run ./cmd/api
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	// Open creates a connection pool; PingContext checks that PostgreSQL is reachable.
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal("failed to open database")
	}
	defer db.Close()

	connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(connectCtx)
	cancel()
	if err != nil {
		// Avoid logging the connection string, which contains the password.
		log.Fatal("failed to connect to database; check DATABASE_URL and PostgreSQL")
		// log.Fatal logs and exits, it does os.Exit(1)
	}
	log.Println("connected to PostgreSQL")

	// Ctrl+C (local development) or SIGTERM (containers) starts shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	postgresStore := job.NewPostgresStore(db)
	bikesPostgresStore := bike.NewPostgresStore(db)
	//store := job.NewMemoryStore()
	jobChannel := make(chan *job.Job, 3)
	bikeChannel := make(chan *bike.Bike, 3)
	// jobChan is the channel connecting handlers to workers.
	// Buffer of 10 means: up to 10 jobs can be waiting
	// without a worker ready, before the channel blocks.

	// Create the pool and set up 3 workers.
	//pool := job.NewWorkerPool(jobChannel, store, &job.SimulatedProcessor{})
	pool := job.NewWorkerPool(jobChannel, postgresStore, &job.SimulatedProcessor{})
	bikePool := bike.NewWorkerPool(bikeChannel, bikesPostgresStore)
	pool.LaunchRoutines(3)
	bikePool.LaunchRoutines(3)

	app := &application{
		//store:     store,
		jobStore:  postgresStore,
		bikeStore: bikesPostgresStore,
		jobChan:   jobChannel,
		bikeChan:  bikeChannel,
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

	r.Post("/bikes", app.createBikeHandler) // On /jobs call app.createJobHandler
	r.Get("/bikes", app.getAllBikesHandler)
	r.Get("/bikes/{id}", app.getBikeHandler)
	r.Delete("/bikes", app.deleteBikesHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// ListenAndServe blocks, so run it separately while main waits for a signal.
	// Buffering lets the server report its result even while main is shutting down.
	serverErr := make(chan error, 1)
	go func() {
		log.Println("starting server on :8080")
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped: %v", err)
		}
	}
	stop() // Restore normal signal handling; a second Ctrl+C can force an exit.

	// Use a fresh context: the signal context is already canceled on shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		// Handlers may still be sending jobs. Exit without closing their channel.
		log.Fatalf("HTTP shutdown failed: %v", err)
	}

	// Shutdown succeeded: no HTTP handlers can send more jobs.
	close(jobChannel)
	close(bikeChannel)
	log.Println("waiting for workers to finish queued jobs")
	pool.Wait()
	bikePool.Wait()
	log.Println("shutdown complete")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type application struct {
	//store     job.Store
	jobStore  job.Store
	bikeStore bike.Store
	jobChan   chan *job.Job
	bikeChan  chan *bike.Bike
}

type createBikeRequest struct {
	BikeType string `json:"bike_type"`
}

func (app *application) deleteBikesHandler(w http.ResponseWriter, r *http.Request) {
	err := app.bikeStore.DeleteAll(r.Context())
	if err != nil {
		log.Printf("deleteBikesHandler: %v", err)
		http.Error(w, "failed to delete bikes", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// application Why create this? Why not a global store?
// Because later you can test with an application with a mocked store
// Also application has methods that you know depend on its fields
func (app *application) createBikeHandler(w http.ResponseWriter, r *http.Request) {
	var req createBikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	b, err := app.bikeStore.Create(r.Context(), req.BikeType)
	if err != nil {
		http.Error(w, "failed to create bike", http.StatusInternalServerError)
		return
	}
	app.bikeChan <- b
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (app *application) getAllBikesHandler(w http.ResponseWriter, r *http.Request) {
	bikes, err := app.bikeStore.GetAll(r.Context())
	if err != nil {
		http.Error(w, "failed to get bikes", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bikes)
}

func (app *application) getBikeHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := app.bikeStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, bike.ErrNotFound) {
			http.Error(w, "bike not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get bike", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)

}

type createJobRequest struct {
	Payload string `json:"payload"`
}

// createJobHandler Notice this receives the same things as a normal view BUT
// Now it also has access to the fields of application (store)
func (app *application) createJobHandler(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	j, err := app.jobStore.Create(r.Context(), req.Payload)
	//j, err := app.store.Create(req.Payload) // Here we need to provide context as well? Otherwise the rest is the same?
	if err != nil {
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	// Queue the job. This blocks while the channel's ten slots are full.
	app.jobChan <- j // This continues the same?
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(j)
}

func (app *application) getJobHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	j, err := app.jobStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, job.ErrNotFound) {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

func (app *application) getAllJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := app.jobStore.GetAll(r.Context())
	if err != nil {
		http.Error(w, "failed to get jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}
