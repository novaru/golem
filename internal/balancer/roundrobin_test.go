package balancer

import (
	"sync"
	"testing"
)

func TestNewRoundRobinBalancer(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://test", 1),
	}

	rr := NewRoundRobinBalancer(backends)
	if rr == nil {
		t.Fatal("NewRoundRobinBalancer returned nil")
	}

	b, _ := rr.NextBackend()
	if b == nil {
		t.Fatal("NextBackend() returned nil after initialization")
	}
}

func TestRoundRobinSelection(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}

	rr := NewRoundRobinBalancer(backends)
	got := []string{}
	for range 4 {
		b, _ := rr.NextBackend()
		if b == nil {
			t.Fatal("NextBackend() returned nil")
		}
		got = append(got, b.URL)
	}

	expectedURL := []string{"http://b", "http://a", "http://b", "http://a"}
	if len(got) != len(expectedURL) {
		t.Fatalf("expected %d results, got %d", len(expectedURL), len(got))
	}

	for i := range expectedURL {
		if got[i] != expectedURL[i] {
			t.Errorf("round robin failed at %d: got %s, expected %s",
				i, got[i], expectedURL[i],
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
	expectedURL := "http://a"

	for i := range 4 {
		b, _ := rr.NextBackend()
		if b == nil {
			t.Fatal("NextBackend() returned nil when healthy backend exists")
		}
		if b.URL != expectedURL {
			t.Errorf("round robin failed at %d: got %s, expected %s",
				i, b.URL, expectedURL)
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

	// Collect results to verify concurrent access works correctly
	results := make([]string, 100)
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			b, _ := rr.NextBackend()
			if b != nil {
				results[index] = b.URL
			}
		}(i)
	}
	wg.Wait()

	// Verify all results are from valid backends
	validURLs := map[string]bool{
		"http://a": true,
		"http://b": true,
		"http://c": true,
	}

	for i, url := range results {
		if url == "" {
			t.Errorf("result at index %d is empty", i)
		} else if !validURLs[url] {
			t.Errorf("result at index %d has invalid URL: %s", i, url)
		}
	}
}

func TestRoundRobinEmptyBackends(t *testing.T) {
	rr := NewRoundRobinBalancer([]*Backend{})
	b, _ := rr.NextBackend()
	if b != nil {
		t.Errorf("expected nil for empty backends, got %v", b)
	}
}

func TestRoundRobinNilBackends(t *testing.T) {
	rr := NewRoundRobinBalancer(nil)
	b, _ := rr.NextBackend()
	if b != nil {
		t.Errorf("expected nil for nil backends, got %v", b)
	}
}

func TestRoundRobinAllUnhealthy(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}

	backends[0].SetHealth(false)
	backends[1].SetHealth(false)

	rr := NewRoundRobinBalancer(backends)
	b, _ := rr.NextBackend()
	if b != nil {
		t.Errorf("expected nil when all backends unhealthy, got %v", b)
	}
}

func TestRoundRobinSingleBackend(t *testing.T) {
	backends := []*Backend{
		NewBackend("http://single", 1),
	}

	rr := NewRoundRobinBalancer(backends)

	// Test multiple calls return the same backend
	for range 4 {
		b, _ := rr.NextBackend()
		if b == nil {
			t.Fatal("NextBackend() returned nil")
		}
		if b.URL != "http://single" {
			t.Errorf("expected http://single, got %s", b.URL)
		}
	}
}

func TestRoundRobinHealthToggling(t *testing.T) {
	// both are healthy initially
	backends := []*Backend{
		NewBackend("http://a", 1),
		NewBackend("http://b", 1),
	}

	rr := NewRoundRobinBalancer(backends)

	b1, _ := rr.NextBackend()
	b2, _ := rr.NextBackend()

	if b1 == nil || b2 == nil {
		t.Fatal("Expected backends to be available")
	}

	// Make one unhealthy
	backends[0].SetHealth(false)

	// Should only return the healthy one
	for range 4 {
		b, _ := rr.NextBackend()
		if b == nil {
			t.Fatal("NextBackend() returned nil")
		}
		if b.URL != "http://b" {
			t.Errorf("expected only http://b, got %s", b.URL)
		}
	}

	// Toggle health back
	backends[0].SetHealth(true)

	// Should resume round-robin
	urls := make(map[string]bool)
	for range 4 {
		b, _ := rr.NextBackend()
		if b != nil {
			urls[b.URL] = true
		}
	}

	if !urls["http://a"] || !urls["http://b"] {
		t.Error("Expected both backends to be used after health recovery")
	}
}
