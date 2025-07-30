package balancer

import (
	"errors"
	"sync/atomic"
)

// RoundRobinBalancer implements a round-robin load balancer.
type RoundRobinBalancer struct {
	backends []*Backend
	index    uint64
}

// NewRoundRobinBalancer creates a new RoundRobinBalancer with the provided backends.
func NewRoundRobinBalancer(backends []*Backend) *RoundRobinBalancer {
	return &RoundRobinBalancer{backends: backends}
}

// NextBackend returns the next healthy backend in a round-robin fashion
// (this will forward requests cyclically between servers and skip over
// unhealthy backends). If no healthy backends are available, it returns nil.
func (r *RoundRobinBalancer) NextBackend() (*Backend, error) {
	n := len(r.backends)
	if n == 0 {
		return nil, errors.New("no backends provided")
	}
	for range n {
		idx := int(atomic.AddUint64(&r.index, 1)) % n
		next := r.backends[idx]
		if next.IsHealthy() {
			return next, nil
		}
	}
	return nil, errors.New("no healthy backends available")
}
