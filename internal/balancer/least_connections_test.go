package balancer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestLeastConnections_SelectsBackendWithLowestActiveRequests(t *testing.T) {
	leastConnections, backends := newTestLeastConnectionsWithBackends(t, "A", "B", "C")
	backends[0].IncrementActive()
	backends[0].IncrementActive()
	backends[1].IncrementActive()

	selectedBackend, err := leastConnections.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selectedBackend.ID != "C" {
		t.Fatalf("expected backend C, got %s", selectedBackend.ID)
	}
}

func TestLeastConnections_SkipsUnhealthyBackend(t *testing.T) {
	leastConnections, backends := newTestLeastConnectionsWithBackends(t, "A", "B", "C")
	backends[2].SetAlive(false)
	backends[0].IncrementActive()
	backends[1].IncrementActive()
	backends[1].IncrementActive()

	selectedBackend, err := leastConnections.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selectedBackend.ID != "A" {
		t.Fatalf("expected backend A, got %s", selectedBackend.ID)
	}
}

func TestLeastConnections_TieBreaksFairly(t *testing.T) {
	leastConnections := newTestLeastConnections(t, "A", "B", "C")
	expectedIDs := []string{"A", "B", "C", "A", "B", "C"}

	for _, expectedID := range expectedIDs {
		selectedBackend, err := leastConnections.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func TestLeastConnections_AllUnhealthyReturnsError(t *testing.T) {
	leastConnections, backends := newTestLeastConnectionsWithBackends(t, "A", "B")
	for _, testBackend := range backends {
		testBackend.SetAlive(false)
	}

	_, err := leastConnections.Next()
	if err == nil {
		t.Fatal("expected error when all backends are unhealthy")
	}
}

func TestLeastConnections_ConcurrentAccess(t *testing.T) {
	leastConnections := newTestLeastConnections(t, "A", "B", "C")
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

			selectedBackend, err := leastConnections.Next()
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

func newTestLeastConnections(t testing.TB, ids ...string) *LeastConnections {
	t.Helper()

	leastConnections, _ := newTestLeastConnectionsWithBackends(t, ids...)

	return leastConnections
}

func newTestLeastConnectionsWithBackends(t testing.TB, ids ...string) (*LeastConnections, []*backend.Backend) {
	t.Helper()

	backends := make([]*backend.Backend, 0, len(ids))
	for index, id := range ids {
		testBackend, err := backend.New(id, fmt.Sprintf("http://localhost:%d", 8081+index))
		if err != nil {
			t.Fatalf("failed to create backend: %v", err)
		}

		backends = append(backends, testBackend)
	}

	leastConnections, err := NewLeastConnections(backends)
	if err != nil {
		t.Fatalf("failed to create least connections: %v", err)
	}

	return leastConnections, backends
}
