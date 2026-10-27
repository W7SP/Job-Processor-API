package ereader

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

// EReader represents a unit of work submitted to the API
type eReader struct {
	ID string `json:"id"`
	Size string `json:"size"`
}

var ErrNotFound = errors.New("EReader not found")

type Store interface {
	Create(size string) (*eReader, error)
	Get(id string) (*eReader, error)
}

type MemoryStore struct {
	mu sync.RWMutex
	eReaders map[string]*eReader
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		eReaders: make(map[string]*eReader),
	}
}

func (s *MemoryStore) Create(size string) (*eReader, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := &eReader{
		ID: uuid.NewString(),
		Size: size,
	}

	s.eReaders[e.ID] = e
	return e, nil
}

func (s *MemoryStore) Get(id string) (*eReader, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.eReaders[id]
	if !ok {
		return nil, ErrNotFound
	}

	return e, nil
}