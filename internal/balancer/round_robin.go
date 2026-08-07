package balancer

import (
	"errors"
	"sync"

	"github.com/Bor-12/load-balancer/internal/backend"
)

type RoundRobin struct {
	mutex    sync.Mutex
	backends []*backend.Backend
	next     int
}

func NewRoundRobin(backends []*backend.Backend) (*RoundRobin, error) {
	if len(backends) == 0 {
		return nil, errors.New("at least one backend is required")
	}

	return &RoundRobin{backends: backends}, nil
}

func (roundRobin *RoundRobin) Next() (*backend.Backend, error) {
	roundRobin.mutex.Lock()
	defer roundRobin.mutex.Unlock()

	for range roundRobin.backends {
		selectedBackend := roundRobin.backends[roundRobin.next]
		roundRobin.next = (roundRobin.next + 1) % len(roundRobin.backends)

		if selectedBackend.IsAlive() {
			return selectedBackend, nil
		}
	}

	return nil, errors.New("no alive backends available")
}
