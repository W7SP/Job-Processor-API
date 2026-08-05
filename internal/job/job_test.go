package job

import (
	"errors"
	"testing"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.Create("resize-image")
	if err != nil {
		t.Fatalf("unexpected error on Create: %v", err) // This stops the test and fails here
	}

	if created.Status != "pending" {
		t.Errorf("expected status %q, got %q", "pending", created.Status) // This records the fail but continues, then still fails the test
	}

	if created.Payload != "resize-image" {
		t.Errorf("expected payload %q, got %q", "resize-image", created.Payload) // This records the fail but continues, then still fails the test
	}
	
	fetched, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}

	if fetched.Payload != "resize-image" {
		t.Errorf("expected payload %q, got %q", "resize-image", fetched.Payload)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
