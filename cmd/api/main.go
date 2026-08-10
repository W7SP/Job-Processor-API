package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/W7SP/Job-Processor-API/internal/job"
	"github.com/W7SP/Job-Processor-API/internal/task"
)

func main() {
	store := job.NewMemoryStore()
	jobChannel := make(chan *job.Job, 10)
	// jobChan is the channel connecting handlers to workers.
	// Buffer of 100 means: up to 100 jobs can be waiting
	// without a worker ready, before the channel blocks.

	// Create the pool and set up 3 workers.
	pool := job.NewWorkerPool(jobChannel, store)
	pool.LaunchRoutines(3)

	app := &application{
		store:      job.NewMemoryStore(),
		storeTasks: task.NewMemoryStore(),
		jobChan:    jobChannel,
	}

	// Creates a router (urls,py)
	r := chi.NewRouter()

	// Adds some middlewares like in settings.py in Django
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// The order matters, each middleware will be executed before every request

	r.Get("/healthz", healthHandler)      // On /healthz call healthHandler, this time specifically for GET requests
	r.Post("/jobs", app.createJobHandler) // On /jobs call app.createJobHandler
	r.Get("/jobs/", app.getAllJobsHandler)
	r.Get("/jobs/{id}", app.getJobHandler)

	r.Post("/tasks", app.createTaskHandler)
	r.Get("/tasks/{id}", app.getTaskHandler)

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil { // Start the server and log the error if such exists
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type application struct {
	store      job.Store
	storeTasks task.Store
	jobChan    chan *job.Job
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

	j, err := app.store.Create(req.Payload)
	if err != nil {
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}

	// Send the job to workers without blocking the HTTP handler.
	// If the channel is full (100 jobs already waiting), we fail fast
	// rather than making the client wait indefinitely.
	app.jobChan <- j
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(j)
}

func (app *application) getJobHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	j, err := app.store.Get(id)
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
	jobs, err := app.store.GetAll()
	if err != nil {
		http.Error(w, "failed to get jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

type createTaskRequest struct {
	Action string `json:"action"`
}

func (app *application) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	t, err := app.storeTasks.Create(req.Action)
	if err != nil {
		http.Error(w, "failed to create task", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}

func (app *application) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := app.storeTasks.Get(id)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get task", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}
