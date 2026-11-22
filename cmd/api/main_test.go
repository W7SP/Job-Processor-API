package main

import (
	"bytes"
	"context"
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
// Go checks interfaces by method signature, so every method of job.Store
// (including GetAll and the ctx parameters) must exist here.
type fakeStore struct {
	createFunc func(ctx context.Context, payload string) (*job.Job, error)
	getFunc    func(ctx context.Context, id string) (*job.Job, error)
}

func (f *fakeStore) Create(ctx context.Context, payload string) (*job.Job, error) {
	return f.createFunc(ctx, payload)
}

func (f *fakeStore) Get(ctx context.Context, id string) (*job.Job, error) {
	return f.getFunc(ctx, id)
}

// GetAll isn't tested yet, but the fake must have it to satisfy job.Store.
func (f *fakeStore) GetAll(ctx context.Context) (map[string]*job.Job, error) {
	return nil, nil
}

func TestCreateJobHandler_Success(t *testing.T) {
	// The handler does app.jobChan <- j. Sending on a nil channel blocks
	// forever, so the test needs a real (buffered) channel.
	jobChan := make(chan *job.Job, 1)

	app := &application{
		jobStore: &fakeStore{
			createFunc: func(ctx context.Context, payload string) (*job.Job, error) {
				return &job.Job{ID: "fixed-id", Status: "pending", Payload: payload}, nil
			},
		},
		jobChan: jobChan,
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

	// Check that the handler really handed the job to the workers.
	select {
	case queued := <-jobChan:
		if queued.ID != "fixed-id" {
			t.Errorf("queued wrong job: %q", queued.ID)
		}
	default:
		t.Fatal("expected the job to be queued for the workers")
	}
}

// IMPORTANT: handlers that use chi.URLParam must be called through a chi router.
// If you call app.getJobHandler(rec, req) directly, chi never parsed the URL,
// so the id is "" and the test doesn't test what you think it tests.
func TestGetJobHandler_NotFound(t *testing.T) {
	// Must be a valid UUID, otherwise the handler returns 400 before reaching the store.
	const missingID = "11111111-1111-1111-1111-111111111111"
	var receivedID string

	app := &application{
		jobStore: &fakeStore{
			getFunc: func(ctx context.Context, id string) (*job.Job, error) { // Called when Store.Get is called in the handler
				receivedID = id
				return nil, job.ErrNotFound
			},
		},
	}

	r := chi.NewRouter()                   // A real chi router, so the {id} in the URL is actually parsed
	r.Get("/jobs/{id}", app.getJobHandler) // Registration only: "on /jobs/{id} call getJobHandler"

	req := httptest.NewRequest(http.MethodGet, "/jobs/"+missingID, nil) // The in-memory request
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req) // chi matches the URL, extracts id, then calls the handler

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if receivedID != missingID {
		t.Errorf("expected handler to receive id %q, got %q", missingID, receivedID)
	}
}

func TestGetJobHandler_InvalidID(t *testing.T) {
	// The empty fakeStore is deliberate: if the handler ever reached the store
	// with a bad ID, getFunc would be nil and the test would panic,
	// telling us the UUID validation broke.
	app := &application{jobStore: &fakeStore{}}

	r := chi.NewRouter()
	r.Get("/jobs/{id}", app.getJobHandler)

	req := httptest.NewRequest(http.MethodGet, "/jobs/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestCreateJobHandler_ClientGoneWhileQueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) // Create a context

	app := &application{ // Use the real application struct
		jobStore: &fakeStore{ // Add a fake store instead of the actual
			//  createFunc will be returned, when Create is called in createJobHandler (check fakeStore for more info)
			createFunc: func(_ context.Context, payload string) (*job.Job, error) {
				cancel() // the client disconnects right after the job is stored
				return &job.Job{ID: "fixed-id", Status: "pending", Payload: payload}, nil // make an actual job (no postgres)
			},
		},
		jobChan: make(chan *job.Job), // unbuffered, nobody reading so nothing will get in it (blocking)
	}

	// Build the request and send it with the context
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"payload":"x"}`)).WithContext(ctx)
	rec := httptest.NewRecorder()

	// Here .Create() is called
	// => executes createFunc (because fakeStore)
	// => context is canceled and job is created
	// => select statement is reached
	// => case 1, nothing will be added to the jobChan cuz its blocking
	// => case 2 is valid cuz context got canceled
	app.createJobHandler(rec, req)
	// !!! IMPORTANT !!! If context is canceled before Create() is called it will return err so if err != nil is true
	// and we never reach the select => a job was created BUT not queued.

	// assert what case 2 from select statement resulted
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

//func TestCreateJobHandler_ClientGoneWhileQueueFull(t *testing.T) {
//	app := &application{
//		jobStore: &fakeStore{
//			createFunc: func(ctx context.Context, payload string) (*job.Job, error) {
//				return &job.Job{ID: "fixed-id", Status: "pending", Payload: payload}, nil
//			},
//		},
//		jobChan: make(chan *job.Job), // unbuffered and nobody reading: a send would block forever
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//	cancel() // the client has already gone away
//
//	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"payload":"x"}`)).WithContext(ctx)
//	rec := httptest.NewRecorder()
//
//	app.createJobHandler(rec, req)
//
//	if rec.Code != http.StatusServiceUnavailable {
//		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
//	}
//}
