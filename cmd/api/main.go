package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter() // Creates a router (urls,py)

	r.Use(middleware.Logger)    // Adds some middlewares like in settings.py in DJango
	r.Use(middleware.Recoverer) // The order matters, each middleware will be executed before every request

	r.Get("/healthz", healthHandler) // On /healthz call healthHandler, this time specifically for GET requests

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil { // Start the server and log the error if such exists
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
