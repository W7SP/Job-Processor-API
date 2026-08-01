package job

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Job represents a unit of work submitted to the API.
// Basically a basic python class with only state and no behavior
type Job struct {
	ID      string `json:"id"`     // Here these JSON tags will be used by JSON as the convention is lower case
	Status  string `json:"status"` // The capital ones are for use in Go
	Payload string `json:"payload"`
}

// ErrNotFound This is done for more precise debugging later.
// We can use errors.Is and check if the error message that we get contains error of type ErrNotFound
var ErrNotFound = errors.New("job not found")

// Store defines what any job storage backend must support.
// Basically the backend that will work with Jobs MUST be able to create and get a job
type Store interface {
	Create(payload string) (*Job, error)
	Get(id string) (*Job, error)
} // Why not UPDATE and DELETE? Cuz being explicit and specific is cool
// The bigger the interface, the weaker the abstraction

// MemoryStore is an in-memory implementation of Store, safe for concurrent use.
// This is done so we can lock and unlock the jobs map for in-memory operations.
// Imagine two POST REQUESTS trying to create a job at the same time -> BAD
type MemoryStore struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

// NewMemoryStore First, any instance returned from this will share the same mutex so that jobs is always safe during concurrency
// Second this will give us an instance with an empty map if we just do :=MemoryStore{} the map will be nil -> BAD
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*Job),
	}
}

func (s *MemoryStore) Create(payload string) (*Job, error) {
	s.mu.Lock()         // Locks the jobs map
	defer s.mu.Unlock() // Unlocks the jobs map before the func returns

	j := &Job{ // Creates a job -> Duh...
		ID:      uuid.NewString(),
		Status:  "pending",
		Payload: payload,
	}
	s.jobs[j.ID] = j
	return j, nil
}

func (s *MemoryStore) Get(id string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return j, nil
}
