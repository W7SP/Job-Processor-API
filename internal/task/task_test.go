package task

import (
	"errors"
	"testing"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	store := NewMemoryStore()
	var actualAction = "wash the dishes"
	task, err := store.Create(actualAction)
	if err != nil {
		t.Fatal("Unexpected error on Create: ", err)
	}

	if task.Action != "wash the dishes" {
		t.Fatalf("Wrong task action, expected %v , got %v", actualAction, task.Action)
	}

	if task.Status != "pending" {
		t.Fatalf("Wrong task status, expected %v , got %v", "pending", task.Status)
	}

	fetched, err := store.Get(task.ID)
	if err != nil {
		t.Fatal("Unexpected error on Get: ", err)
	}

	if fetched.ID != task.ID {
		t.Fatalf("Wrong task id, expected %v , got %v", fetched.ID, task.ID)
	}

	if fetched.Action != actualAction {
		t.Fatalf("Wrong task action, expected %v , got %v", actualAction, fetched.Action)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get("does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("Expected ErrNotFound, got ", err)
	}
}
