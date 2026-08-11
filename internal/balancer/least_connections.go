package balancer

import (
	"errors"
	"sync"

	"github.com/Bor-12/load-balancer/internal/backend"
)

type LeastConnections struct {
	mutex    sync.Mutex
	backends []*backend.Backend
	next     int
}

func NewLeastConnections(backends []*backend.Backend) (*LeastConnections, error) {
	if len(backends) == 0 {
		return nil, errors.New("at least one backend is required")
	}

	return &LeastConnections{backends: backends}, nil
}

func (leastConnections *LeastConnections) Next() (*backend.Backend, error) {
	leastConnections.mutex.Lock()
	defer leastConnections.mutex.Unlock()

	selectedBackend := (*backend.Backend)(nil)
	selectedIndex := -1
	lowestActiveRequests := 0

	for offset := range leastConnections.backends {
		index := (leastConnections.next + offset) % len(leastConnections.backends)
		candidate := leastConnections.backends[index]

		if !candidate.IsAlive() {
			continue
		}

		activeRequests := candidate.ActiveCount()
		if selectedBackend == nil || activeRequests < lowestActiveRequests {
			selectedBackend = candidate
			selectedIndex = index
			lowestActiveRequests = activeRequests
		}
	}

	if selectedBackend == nil {
		return nil, errors.New("no alive backends available")
	}

	leastConnections.next = (selectedIndex + 1) % len(leastConnections.backends)

	return selectedBackend, nil
}
