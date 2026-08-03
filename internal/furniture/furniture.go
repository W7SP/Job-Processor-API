package furniture

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Furniture struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Material string `json:"material"`
}

var ErrorNotFound = fmt.Errorf("furniture not found")

type Store interface {
	Create(ftype, material string) (*Furniture, error)
	Get(id string) (*Furniture, error)
}

type MemoryStore struct {
	mu sync.Mutex
	furniture map[string]*Furniture
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		furniture: make(map[string]*Furniture),
	}
}

func (s *MemoryStore) Create(ftype, material string) (*Furniture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	furniture := &Furniture{
		ID:      uuid.NewString(),
		Type:    ftype,
		Material: material,
	}

	s.furniture[furniture.ID] = furniture
	return furniture, nil
}

func (s *MemoryStore) Get(id string) (*Furniture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	furniture, ok := s.furniture[id]
	if !ok {
		return nil, ErrorNotFound
	}
	return furniture, nil
}