package balancer

import (
	"errors"
	"math"
	"sync"
)

type LeastConnBalancer struct {
	backends []*Backend
	mutex    sync.RWMutex
}

func NewLeastConnBalancer(backends []*Backend) *LeastConnBalancer {
	return &LeastConnBalancer{
		backends: backends,
	}
}

func (l *LeastConnBalancer) NextBackend() (*Backend, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if len(l.backends) == 0 {
		return nil, errors.New("no backends available")
	}

	var selected *Backend
	minConnections := math.MaxInt

	for _, b := range l.backends {
		b.mu.RLock()

		if !b.healthy {
			b.mu.RUnlock()
			continue
		}

		if b.connections < minConnections {
			minConnections = b.connections
			selected = b
		}

		b.mu.RUnlock()
	}

	if selected == nil {
		return nil, errors.New("no healthy backend available")
	}

	return selected, nil
}
