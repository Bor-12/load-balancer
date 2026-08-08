package backend

import "testing"

func TestBackend_StoresParsedURL(t *testing.T) {
	backend, err := New("backend-1", "http://localhost:8081")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	if backend.ID != "backend-1" {
		t.Fatalf("expected ID backend-1, got %s", backend.ID)
	}

	if backend.URL.String() != "http://localhost:8081" {
		t.Fatalf("expected URL http://localhost:8081, got %s", backend.URL.String())
	}

	if backend.Weight != 1 {
		t.Fatalf("expected default weight %d, got %d", 1, backend.Weight)
	}

	if !backend.IsAlive() {
		t.Fatal("expected backend to start alive")
	}
}

func TestBackend_StoresWeight(t *testing.T) {
	backend, err := NewWithWeight("backend-1", "http://localhost:8081", 5)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	if backend.Weight != 5 {
		t.Fatalf("expected weight %d, got %d", 5, backend.Weight)
	}
}

func TestBackend_RejectsInvalidWeight(t *testing.T) {
	_, err := NewWithWeight("backend-1", "http://localhost:8081", 0)
	if err == nil {
		t.Fatal("expected error with invalid weight")
	}
}
