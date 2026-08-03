package task

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

type Task struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Action string `json:"action"`
}

var ErrNotFound = errors.New("task not found")

type Store interface {
	Create(action string) (*Task, error)
	Get(id string) (*Task, error)
}

type MemoryStore struct {
	mu    sync.Mutex
	tasks map[string]*Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]*Task),
	}
}

func (s *MemoryStore) Create(action string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var task = &Task{
		ID: uuid.NewString(),
		Status: "pending",
		Action: action,
	}

	s.tasks[task.ID] = task
	return task, nil
}

func (s *MemoryStore) Get(id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return task, nil
}