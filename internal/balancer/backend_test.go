package balancer

import (
	"sync"
	"testing"
)

func TestBackendConnectionsCount(t *testing.T) {
	b := NewBackend("http://example.com", 1)
	b.AddConnections()
	b.AddConnections()

	if b.GetConnections() != 2 {
		t.Errorf("expected 2 connections, got %d\n",
			b.GetConnections())
	}

	b.RemoveConnections()
	if b.GetConnections() != 1 {
		t.Errorf("expected 1 connections, got %d\n",
			b.GetConnections())
	}
}

func TestBackendHealthStatus(t *testing.T) {
	b := NewBackend("http://example.com", 1)

	b.SetHealth(true)
	if !b.IsHealthy() {
		t.Error("expected backend to be healthy")
	}

	b.SetHealth(false)
	if b.IsHealthy() {
		t.Error("expected backend to be unhealthy")
	}
}

func TestBackendConcurrentAccess(t *testing.T) {
	b := NewBackend("http://example.com", 1)
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			b.AddConnections()
			b.IsHealthy()
			b.SetHealth(true)
			b.RemoveConnections()
			wg.Done()
		}()
	}

	wg.Wait()
}
