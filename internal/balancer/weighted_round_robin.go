package balancer

import (
	"errors"
	"sync"

	"github.com/Bor-12/load-balancer/internal/backend"
)

type WeightedRoundRobin struct {
	mutex          sync.Mutex
	backends       []*backend.Backend
	currentWeights []int
}

func NewWeightedRoundRobin(backends []*backend.Backend) (*WeightedRoundRobin, error) {
	if len(backends) == 0 {
		return nil, errors.New("at least one backend is required")
	}

	for _, candidate := range backends {
		if candidate.Weight < 1 {
			return nil, errors.New("backend weight must be at least 1")
		}
	}

	return &WeightedRoundRobin{
		backends:       backends,
		currentWeights: make([]int, len(backends)),
	}, nil
}

func (roundRobin *WeightedRoundRobin) Next() (*backend.Backend, error) {
	roundRobin.mutex.Lock()
	defer roundRobin.mutex.Unlock()

	selectedIndex := -1
	totalWeight := 0

	for index, candidate := range roundRobin.backends {
		if !candidate.IsAlive() {
			continue
		}

		totalWeight += candidate.Weight
		roundRobin.currentWeights[index] += candidate.Weight

		if selectedIndex == -1 || roundRobin.currentWeights[index] > roundRobin.currentWeights[selectedIndex] {
			selectedIndex = index
		}
	}

	if selectedIndex == -1 {
		return nil, errors.New("no alive backends available")
	}

	roundRobin.currentWeights[selectedIndex] -= totalWeight

	return roundRobin.backends[selectedIndex], nil
}
