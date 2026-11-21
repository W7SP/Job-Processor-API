package job

import (
	"errors"
	"testing"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	store := NewMemoryStore()

	created, err := store.Create(t.Context(), "resize-image")
	if err != nil {
		t.Fatalf("unexpected error on Create: %v", err) // This stops the test and fails here
	}

	if created.Status != "pending" {
		t.Errorf("expected status %q, got %q", "pending", created.Status) // This records the fail but continues, then still fails the test
	}

	if created.Payload != "resize-image" {
		t.Errorf("expected payload %q, got %q", "resize-image", created.Payload) // This records the fail but continues, then still fails the test
	}

	fetched, err := store.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}

	if fetched.Payload != "resize-image" {
		t.Errorf("expected payload %q, got %q", "resize-image", fetched.Payload)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get(t.Context(), "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_UpdateStatus(t *testing.T) {
	store := NewMemoryStore()
	j, err := store.Create(t.Context(), "resize-image")
	if err != nil {
		t.Fatalf("unexpected error on Create: %v", err)
	}

	errFromUpdate := store.UpdateStatus(t.Context(), j.ID, "running")
	if errFromUpdate != nil {
		t.Fatalf("unexpected error on Update: %v", err)
	}

	updated, err := store.Get(t.Context(), j.ID)
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if updated.Status != "running" {
		t.Errorf("expected status %q, got %q", "running", updated.Status)
	}
	if j.Status != "pending" {
		t.Errorf("created snapshot changed: got status %q", j.Status)
	}
}

func TestMemoryStore_ReturnedJobsAreIndependent(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(t.Context(), "original")
	if err != nil {
		t.Fatal(err)
	}
	id := created.ID
	created.Payload = "changed through Create"

	fetched, err := store.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Payload != "original" {
		t.Fatal("changing the Create result changed the stored job")
	}
	fetched.Payload = "changed through Get"

	all, err := store.GetAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if all[id].Payload != "original" {
		t.Fatal("changing the Get result changed the stored job")
	}
	all[id].Payload = "changed through GetAll"
	delete(all, id)

	fetched, err = store.Get(t.Context(), id)
	if err != nil {
		t.Fatalf("changing the returned map removed the stored job: %v", err)
	}
	if fetched.Payload != "original" {
		t.Fatal("changing the GetAll result changed the stored job")
	}
}
