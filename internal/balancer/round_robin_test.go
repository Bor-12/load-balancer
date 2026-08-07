package balancer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestRoundRobin_ReturnsBackendsInOrder(t *testing.T) {
	roundRobin := newTestRoundRobin(t, "A", "B", "C")

	expectedIDs := []string{"A", "B", "C"}

	for _, expectedID := range expectedIDs {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func TestRoundRobin_WrapsAround(t *testing.T) {
	roundRobin := newTestRoundRobin(t, "A", "B", "C")

	expectedIDs := []string{"A", "B", "C", "A", "B", "C"}

	for _, expectedID := range expectedIDs {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func TestRoundRobin_WithSingleBackend(t *testing.T) {
	roundRobin := newTestRoundRobin(t, "A")

	for range 3 {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != "A" {
			t.Fatalf("expected backend A, got %s", selectedBackend.ID)
		}
	}
}

func TestRoundRobin_ReturnsErrorWithNoBackends(t *testing.T) {
	_, err := NewRoundRobin(nil)
	if err == nil {
		t.Fatal("expected error with no backends")
	}
}

func TestRoundRobin_ConcurrentAccess(t *testing.T) {
	roundRobin := newTestRoundRobin(t, "A", "B", "C")
	validIDs := map[string]bool{
		"A": true,
		"B": true,
		"C": true,
	}

	var waitGroup sync.WaitGroup
	for range 100 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			selectedBackend, err := roundRobin.Next()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !validIDs[selectedBackend.ID] {
				t.Errorf("unexpected backend ID %s", selectedBackend.ID)
			}
		}()
	}

	waitGroup.Wait()
}

func TestRoundRobin_SkipsUnhealthyBackend(t *testing.T) {
	roundRobin, backends := newTestRoundRobinWithBackends(t, "A", "B", "C")
	backends[1].SetAlive(false)

	expectedIDs := []string{"A", "C", "A", "C"}

	for _, expectedID := range expectedIDs {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func TestRoundRobin_ReturnsErrorWhenAllBackendsUnhealthy(t *testing.T) {
	roundRobin, backends := newTestRoundRobinWithBackends(t, "A", "B", "C")
	for _, testBackend := range backends {
		testBackend.SetAlive(false)
	}

	_, err := roundRobin.Next()
	if err == nil {
		t.Fatal("expected error when all backends are unhealthy")
	}
}

func TestRoundRobin_ReincludesRecoveredBackend(t *testing.T) {
	roundRobin, backends := newTestRoundRobinWithBackends(t, "A", "B", "C")
	backends[1].SetAlive(false)

	for _, expectedID := range []string{"A", "C"} {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}

	backends[1].SetAlive(true)

	for _, expectedID := range []string{"A", "B", "C"} {
		selectedBackend, err := roundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func newTestRoundRobin(t *testing.T, ids ...string) *RoundRobin {
	t.Helper()

	roundRobin, _ := newTestRoundRobinWithBackends(t, ids...)

	return roundRobin
}

func newTestRoundRobinWithBackends(t *testing.T, ids ...string) (*RoundRobin, []*backend.Backend) {
	t.Helper()

	backends := make([]*backend.Backend, 0, len(ids))
	for index, id := range ids {
		testBackend, err := backend.New(id, fmt.Sprintf("http://localhost:%d", 8081+index))
		if err != nil {
			t.Fatalf("failed to create backend: %v", err)
		}

		backends = append(backends, testBackend)
	}

	roundRobin, err := NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("failed to create round robin: %v", err)
	}

	return roundRobin, backends
}
