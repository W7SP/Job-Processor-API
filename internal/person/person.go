package person

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

type Person struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Age int `json:"age"`
	Occupation string `json:"occupation"`
}

var ErrorNotFound = errors.New("not found")

type Store interface {
	Create(name, occupation string, age int) (*Person, error)
	Get(id string) (*Person, error)
}

type MemoryStore struct {
	mu sync.Mutex
	store map[string]*Person
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: make(map[string]*Person),
	}
}

func (m *MemoryStore) Create(name, occupation string, age int) (*Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	person := &Person{
		ID: uuid.NewString(),
		Name: name,
		Occupation: occupation,
		Age: age,
	}

	m.store[person.ID] = person
	return person, nil
}

func (m *MemoryStore) Get(id string) (*Person, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	person, ok := m.store[id]
	if !ok {return nil, ErrorNotFound}
	return person, nil
}
