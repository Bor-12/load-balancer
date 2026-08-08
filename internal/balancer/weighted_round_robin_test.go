package balancer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Bor-12/load-balancer/internal/backend"
)

func TestWeightedRoundRobin_EqualWeightsBehavesEvenly(t *testing.T) {
	weightedRoundRobin := newTestWeightedRoundRobin(t, weightedBackendSpec{"A", 1}, weightedBackendSpec{"B", 1}, weightedBackendSpec{"C", 1})

	expectedIDs := []string{"A", "B", "C", "A", "B", "C"}
	for _, expectedID := range expectedIDs {
		selectedBackend, err := weightedRoundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if selectedBackend.ID != expectedID {
			t.Fatalf("expected backend %s, got %s", expectedID, selectedBackend.ID)
		}
	}
}

func TestWeightedRoundRobin_RespectsWeights(t *testing.T) {
	weightedRoundRobin := newTestWeightedRoundRobin(t, weightedBackendSpec{"A", 5}, weightedBackendSpec{"B", 3}, weightedBackendSpec{"C", 2})

	counts := map[string]int{}
	for range 10 {
		selectedBackend, err := weightedRoundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts[selectedBackend.ID]++
	}

	if counts["A"] != 5 {
		t.Fatalf("expected backend A count %d, got %d", 5, counts["A"])
	}
	if counts["B"] != 3 {
		t.Fatalf("expected backend B count %d, got %d", 3, counts["B"])
	}
	if counts["C"] != 2 {
		t.Fatalf("expected backend C count %d, got %d", 2, counts["C"])
	}
}

func TestWeightedRoundRobin_SkipsUnhealthyBackend(t *testing.T) {
	weightedRoundRobin, backends := newTestWeightedRoundRobinWithBackends(t, weightedBackendSpec{"A", 5}, weightedBackendSpec{"B", 3}, weightedBackendSpec{"C", 2})
	backends[0].SetAlive(false)

	counts := map[string]int{}
	for range 5 {
		selectedBackend, err := weightedRoundRobin.Next()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts[selectedBackend.ID]++
	}

	if counts["A"] != 0 {
		t.Fatalf("expected backend A count %d, got %d", 0, counts["A"])
	}
	if counts["B"] != 3 {
		t.Fatalf("expected backend B count %d, got %d", 3, counts["B"])
	}
	if counts["C"] != 2 {
		t.Fatalf("expected backend C count %d, got %d", 2, counts["C"])
	}
}

func TestWeightedRoundRobin_AllUnhealthyReturnsError(t *testing.T) {
	weightedRoundRobin, backends := newTestWeightedRoundRobinWithBackends(t, weightedBackendSpec{"A", 5}, weightedBackendSpec{"B", 3})
	for _, testBackend := range backends {
		testBackend.SetAlive(false)
	}

	_, err := weightedRoundRobin.Next()
	if err == nil {
		t.Fatal("expected error when all backends are unhealthy")
	}
}

func TestWeightedRoundRobin_ConcurrentAccess(t *testing.T) {
	weightedRoundRobin := newTestWeightedRoundRobin(t, weightedBackendSpec{"A", 5}, weightedBackendSpec{"B", 3}, weightedBackendSpec{"C", 2})
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

			selectedBackend, err := weightedRoundRobin.Next()
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

type weightedBackendSpec struct {
	id     string
	weight int
}

func newTestWeightedRoundRobin(t *testing.T, specs ...weightedBackendSpec) *WeightedRoundRobin {
	t.Helper()

	weightedRoundRobin, _ := newTestWeightedRoundRobinWithBackends(t, specs...)

	return weightedRoundRobin
}

func newTestWeightedRoundRobinWithBackends(t *testing.T, specs ...weightedBackendSpec) (*WeightedRoundRobin, []*backend.Backend) {
	t.Helper()

	backends := make([]*backend.Backend, 0, len(specs))
	for index, spec := range specs {
		testBackend, err := backend.NewWithWeight(spec.id, fmt.Sprintf("http://localhost:%d", 8081+index), spec.weight)
		if err != nil {
			t.Fatalf("failed to create backend: %v", err)
		}

		backends = append(backends, testBackend)
	}

	weightedRoundRobin, err := NewWeightedRoundRobin(backends)
	if err != nil {
		t.Fatalf("failed to create weighted round robin: %v", err)
	}

	return weightedRoundRobin, backends
}
