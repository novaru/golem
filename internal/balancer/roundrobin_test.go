package balancer

import (
	"sync"
	"testing"
)

func TestRoundRobinSelection(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}

	rr := NewRoundRobinBalancer(backends)
	got := []string{}
	for range 4 {
		b := rr.NextBackend()
		got = append(got, b.URL)
	}

	expectedUrl := []string{"http://b", "http://a", "http://b", "http://a"}
	for i := range expectedUrl {
		if got[i] != expectedUrl[i] {
			t.Errorf("round robin failed at %d: got %s, expected %s",
				i, got[i], expectedUrl[i],
			)
		}
	}
}

func TestRoundRobinWithUnhealthy(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}

	backends[1].SetHealth(false)
	rr := NewRoundRobinBalancer(backends)
	expectedUrl := "http://a"

	for i := range 4 {
		b := rr.NextBackend()
		if b.URL != expectedUrl {
			t.Errorf("round robin failed at %d: got %s, expected %s",
				i, b.URL, expectedUrl)
		}
	}
}

func TestRoundRobinConcurrent(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
		NewBackend("http://c", 1),
	}
	rr := NewRoundRobinBalancer(backends)
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			_ = rr.NextBackend()
			wg.Done()
		}()
	}
	wg.Wait()
}
