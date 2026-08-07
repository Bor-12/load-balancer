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

	if !backend.IsAlive() {
		t.Fatal("expected backend to start alive")
	}
}
