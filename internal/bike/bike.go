package bike

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Create the data structure
type bike struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Material string `json:"Material"`
}

// ErrorNotFound Create the error not found
var ErrorNotFound = errors.New("bike not found")

// Store Interface for the storage in memory with the same things that would be required for the postgres db
type Store interface {
	Create(btype, material string) (*bike, error)
	Get(id string) (*bike, error)
}

// MemoryStore InMemory storage
type MemoryStore struct {
	mu    sync.Mutex
	bikes map[string]*bike
}

// NewMemoryStore Method that inits the in memory storage with the data (map) instantiated
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bikes: make(map[string]*bike),
	}
}

// Create The methods Create / Add that are in the itnerface
func (s *MemoryStore) Create(btype, material string) (*bike, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &bike{
		ID:       uuid.NewString(),
		Type:     btype,
		Material: material,
	}
	s.bikes[b.ID] = b
	return b, nil
}

func (s *MemoryStore) Get(id string) (*bike, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, found := s.bikes[id]
	if !found {
		return nil, ErrorNotFound
	}
	return b, nil

}
