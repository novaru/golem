package balancer

type LeastConnBalancer struct {
	backends []*Backend
}

func NewLeastConnBalancer(backends []*Backend) *LeastConnBalancer {
	return &LeastConnBalancer{backends: backends}
}

func (l *LeastConnBalancer) NextBackend() *Backend {
	var selected *Backend
	minConnections := int(^uint(0) >> 1) // max int value (2^63 - 1)

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

	return selected
}
