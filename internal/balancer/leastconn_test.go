package balancer

import (
	"sync"
	"testing"
)

func TestNewLeastConnBalancer(t *testing.T) {
	t.Run("creates balancer with backends", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: true, connections: 0, weight: 1},
			{URL: "http://backend2.com", healthy: true, connections: 0, weight: 1},
		}

		balancer := NewLeastConnBalancer(backends)

		if balancer == nil {
			t.Fatal("expected balancer to be created, got nil")
		}

		if len(balancer.backends) != 2 {
			t.Errorf("expected 2 backends, got %d", len(balancer.backends))
		}

		if balancer.backends[0].URL != "http://backend1.com" {
			t.Errorf("expected first backend URL to be 'http://backend1.com', got %s", balancer.backends[0].URL)
		}
	})

	t.Run("creates balancer with empty backends", func(t *testing.T) {
		balancer := NewLeastConnBalancer([]*Backend{})

		if balancer == nil {
			t.Fatal("expected balancer to be created, got nil")
		}

		if len(balancer.backends) != 0 {
			t.Errorf("expected 0 backends, got %d", len(balancer.backends))
		}
	})

	t.Run("creates balancer with nil backends", func(t *testing.T) {
		balancer := NewLeastConnBalancer(nil)

		if balancer == nil {
			t.Fatal("expected balancer to be created, got nil")
		}

		if balancer.backends != nil {
			t.Errorf("expected backends to be nil, got %v", balancer.backends)
		}
	})
}

func TestLeastConnBalancer_NextBackend(t *testing.T) {
	t.Run("returns backend with least connections", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: true, connections: 5, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend2.com", healthy: true, connections: 2, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend3.com", healthy: true, connections: 8, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected == nil {
			t.Fatal("expected a backend to be selected, got nil")
		}

		if selected.URL != "http://backend2.com" {
			t.Errorf("expected backend2 (least connections), got %s", selected.URL)
		}
	})

	t.Run("skips unhealthy backends", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: false, connections: 1, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend2.com", healthy: true, connections: 5, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend3.com", healthy: false, connections: 2, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected == nil {
			t.Fatal("expected a backend to be selected, got nil")
		}

		if selected.URL != "http://backend2.com" {
			t.Errorf("expected backend2 (only healthy), got %s", selected.URL)
		}
	})

	t.Run("returns nil when no healthy backends", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: false, connections: 1, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend2.com", healthy: false, connections: 2, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected != nil {
			t.Errorf("expected nil when no healthy backends, got %v", selected)
		}
	})

	t.Run("returns nil when no backends", func(t *testing.T) {
		balancer := NewLeastConnBalancer([]*Backend{})
		selected := balancer.NextBackend()

		if selected != nil {
			t.Errorf("expected nil when no backends, got %v", selected)
		}
	})

	t.Run("handles single backend", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: true, connections: 10, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected == nil {
			t.Fatal("expected a backend to be selected, got nil")
		}

		// check wether you are my only wife or a behemoth who came from the abyss
		if selected.URL != "http://backend1.com" {
			t.Errorf("expected backend1, got %s", selected.URL)
		}
	})

	t.Run("handles backends with same connection count", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: true, connections: 5, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend2.com", healthy: true, connections: 5, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend3.com", healthy: true, connections: 5, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected == nil {
			t.Fatal("expected a backend to be selected, got nil")
		}

		// Should return the first one encountered with minimum connections
		if selected.URL != "http://backend1.com" {
			t.Errorf("expected backend1 (first with min connections), got %s", selected.URL)
		}
	})

	t.Run("handles zero connections", func(t *testing.T) {
		backends := []*Backend{
			{URL: "http://backend1.com", healthy: true, connections: 0, weight: 1, mu: sync.RWMutex{}},
			{URL: "http://backend2.com", healthy: true, connections: 3, weight: 1, mu: sync.RWMutex{}},
		}

		balancer := NewLeastConnBalancer(backends)
		selected := balancer.NextBackend()

		if selected == nil {
			t.Fatal("expected a backend to be selected, got nil")
		}

		if selected.URL != "http://backend1.com" {
			t.Errorf("expected backend1 (zero connections), got %s", selected.URL)
		}
	})
}
