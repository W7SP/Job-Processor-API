package car

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

type Car struct {
	ID string `json:"id"`
	Brand string `json:"brand"`
}

var ErrNotFound = errors.New("car not found")

type Store interface {
	Create(brand string) (*Car, error)
	Get(id string) (*Car, error)
}

type MemoryStore struct {
	mu sync.Mutex
	cars map[string]*Car
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		cars: make(map[string]*Car),
	}
}

func (s *MemoryStore) Create(brand string) (*Car, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c := &Car{
		ID: uuid.NewString(),
		Brand: brand,
	}

	s.cars[c.ID] = c
	return c, nil
}

func (s *MemoryStore) Get(id string) (*Car, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.cars[id]
	if !ok {
		return nil, ErrNotFound
	}
	return c, nil
}