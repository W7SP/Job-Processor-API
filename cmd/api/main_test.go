package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/W7SP/Job-Processor-API/internal/job"
	"github.com/go-chi/chi/v5"
)

// fakeStore is a test-only implementation of job.Store.
// It satisfies the interface just like MemoryStore does, but lets us
// control exactly what it returns — no randomness, no real storage.
type fakeStore struct {
	createFunc func(payload string) (*job.Job, error)
	getFunc    func(id string) (*job.Job, error)
}

func (f *fakeStore) Create(payload string) (*job.Job, error) {
	return f.createFunc(payload)
}

func (f *fakeStore) Get(id string) (*job.Job, error) {
	return f.getFunc(id)
}

func TestCreateJobHandler_Success(t *testing.T) {
	app := &application{
		store: &fakeStore{
			createFunc: func(payload string) (*job.Job, error) {
				return &job.Job{ID: "fixed-id", Status: "pending", Payload: payload}, nil
			},
		},
	}

	body := bytes.NewBufferString(`{"payload":"resize-image"}`)
	req := httptest.NewRequest(http.MethodPost, "/jobs", body)
	rec := httptest.NewRecorder()

	app.createJobHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var got job.Job
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if got.ID != "fixed-id" {
		t.Errorf("expected ID %q, got %q", "fixed-id", got.ID)
	}
}

// An example of a fake passing test
func TestGetJobHandler_NotFound_IMPROPER(t *testing.T) {
	var receivedID string

	app := &application{
		store: &fakeStore{
			getFunc: func(id string) (*job.Job, error) { // Add a functionality to getFunc, it is called when Store.Get is called
				receivedID = id // This is to check what exactly ID are we using, chi WILL NOT parse this
				return nil, job.ErrNotFound
			},
		},
	}

	// Create request towards /jobs/does-not-exist with an empty body
	req := httptest.NewRequest(http.MethodGet, "/jobs/does-not-exist", nil)
	rec := httptest.NewRecorder() // In memory server

	app.getJobHandler(rec, req) // Send the request

	if rec.Code != http.StatusNotFound {
		// We will receive this error bacause our ID is "", so same result as a wrong ID, so the test is not proper
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	t.Logf("id received by store.Get: %q", receivedID)
}

func TestGetJobHandler_NotFound(t *testing.T) {
	var receivedID string

	app := &application{
		store: &fakeStore{
			getFunc: func(id string) (*job.Job, error) { // This is called when Store.Get is called in the handler (view)
				receivedID = id
				return nil, job.ErrNotFound
			},
		},
	}

	r := chi.NewRouter()                   // We use an actual chi router so that we can actually parse the ID in getJobHandler
	r.Get("/jobs/{id}", app.getJobHandler) // This is registration
	// It does NOT create a request it says on /jobs/{id} call app.getJobHandler

	req := httptest.NewRequest(http.MethodGet, "/jobs/does-not-exist", nil) // This is the request (in memory)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req) // route it through chi, not calling the handler directly
	/*
		This is where matching actually happens: chi takes the request from step 2,
		Walks through all the rules registered in step 1,
		Finds that /jobs/does-not-exist matches the pattern /jobs/{id}
		Extracts "does-not-exist" as the value for id, attaches that captured value onto the request
	*/
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if receivedID != "does-not-exist" {
		t.Errorf("expected handler to receive id %q, got %q", "does-not-exist", receivedID)
	}
}
