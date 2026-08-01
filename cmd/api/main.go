package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/W7SP/Job-Processor-API/internal/job"
)

func main() {
	app := &application{
		store: job.NewMemoryStore(),
	}

	// Creates a router (urls,py)
	r := chi.NewRouter()

	// Adds some middlewares like in settings.py in Django
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// The order matters, each middleware will be executed before every request

	r.Get("/healthz", healthHandler)      // On /healthz call healthHandler, this time specifically for GET requests
	r.Post("/jobs", app.createJobHandler) // On /jobs call app.createJobHandler
	r.Get("/jobs/{id}", app.getJobHandler)

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil { // Start the server and log the error if such exists
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type createJobRequest struct {
	Payload string `json:"payload"`
}

// application Why create this? Why not a global store?
// Because later you can test with an application wit ha mocked store
// Also application has methods that you know depend on its fields
type application struct {
	store job.Store
}

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
