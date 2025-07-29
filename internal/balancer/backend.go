package balancer

import (
	"sync"

	"github.com/novaru/golem/internal/metrics"
)

// Backend represents a connection to a backend server.
type Backend struct {
	URL         string
	healthy     bool
	connections int
	weight      int

	mu sync.RWMutex
}

// SetHealth updates the health status of the backend.
func (b *Backend) SetHealth(healthy bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.healthy = healthy
	metrics.UpdateBackendHealth(b.URL, healthy)
}

// IsHealthy returns whether the backend is healthy.
func (b *Backend) IsHealthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.healthy
}

// NewBackend creates and returns a new Backend instance.
func NewBackend(url string, weight int) *Backend {
	metrics.UpdateBackendHealth(url, true)
	return &Backend{
		URL:     url,
		healthy: true,
		weight:  weight,
	}
}

// AddConnections increments the current connection count.
func (b *Backend) AddConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.connections++
}

// RemoveConnections decrements the current connection count.
func (b *Backend) RemoveConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.connections > 0 {
		b.connections--
	}
}

// GetConnections returns the current number of active connections.
func (b *Backend) GetConnections() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connections
}
