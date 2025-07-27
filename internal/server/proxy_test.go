package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/novaru/golem/internal/balancer"
)

func TestProxyServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		backendHandler http.HandlerFunc
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Successful response",
			backendHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("hello from backend"))
			},
			path:           "/test",
			expectedStatus: http.StatusOK,
			expectedBody:   "hello from backend",
		},
		{
			name: "Not found on backend",
			backendHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("not found"))
			},
			path:           "/missing",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(tt.backendHandler)
			defer backend.Close()

			bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{
				balancer.NewBackend(backend.URL, 1),
			})
			proxy := NewProxyServer(bal)

			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			proxy.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d; got %d", tt.expectedStatus, rr.Code)
			}

			if strings.TrimSpace(rr.Body.String()) != tt.expectedBody {
				t.Errorf("Expected body %q; got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestProxyStreamingResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for i := range 3 {
			fmt.Fprintf(w, "line %d\n", i+1)
			w.(http.Flusher).Flush()
			time.Sleep(200 * time.Millisecond)
		}
	}))
	defer backend.Close()

	bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{
		balancer.NewBackend(backend.URL, 1),
	})
	proxy := NewProxyServer(bal)

	req := httptest.NewRequest("GET", "/stream", nil)
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	scanner := bufio.NewScanner(rr.Body)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	expected := []string{"line 1", "line 2", "line 3"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}
	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Line %d: expected %q, got %q", i+1, expected[i], line)
		}
	}
}

func TestProxyNoHealthyBackends(t *testing.T) {
	backend := balancer.NewBackend("http://127.0.0.1:12345", 1)
	backend.SetHealth(false)

	bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{backend})
	proxy := NewProxyServer(bal)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "No healthy backend available") {
		t.Errorf("Unexpected error message: %s", rr.Body.String())
	}
}

func TestProxyBackendUnreachable(t *testing.T) {
	// Use an invalid backend URL that will fail immediately
	backend := balancer.NewBackend("http://127.0.0.1:12345", 1)
	bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{backend})
	proxy := NewProxyServer(bal)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Backend unavailable") {
		t.Errorf("Unexpected error message: %s", rr.Body.String())
	}

	if backend.IsHealthy() {
		t.Error("Backend should be marked as unhealthy after failure")
	}
}

func TestProxyConnectionTracking(t *testing.T) {
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("done"))
	}))
	defer backendServer.Close()

	backend := balancer.NewBackend(backendServer.URL, 1)
	bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{backend})
	proxy := NewProxyServer(bal)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	proxy.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	// Wait to ensure connection count is decremented
	time.Sleep(100 * time.Millisecond)

	if backend.GetConnections() != 0 {
		t.Errorf("Expected 0 connections, got %d", backend.GetConnections())
	}
}

func BenchmarkProxyServeHTTP(b *testing.B) {
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "benchmark response")
	}))
	defer backend.Close()

	bal, _ := balancer.NewBalancer("roundrobin", []*balancer.Backend{
		balancer.NewBackend(backend.URL, 1),
	})
	proxy := NewProxyServer(bal)

	b.ResetTimer()
	for b.Loop() {
		req := httptest.NewRequest("GET", "/bench", nil)
		rr := httptest.NewRecorder()
		proxy.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("Request failed with code %d", rr.Code)
		}
	}
}

func TestProxyLeastConnBalancingWithStreaming(t *testing.T) {
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Simulate long-running streaming connection
		for i := range 10 {
			fmt.Fprintf(w, "backend1-line-%d\n", i+1)
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "backend2-response")
	}))
	defer backend2.Close()

	be1 := balancer.NewBackend(backend1.URL, 1)
	be2 := balancer.NewBackend(backend2.URL, 1)

	bal, _ := balancer.NewBalancer("leastconn", []*balancer.Backend{be1, be2})
	proxy := NewProxyServer(bal)

	// Make two concurrent streaming requests to backend1 (since they start with 0 connections each)
	// This should create an imbalance: be1.connections = 2, be2.connections = 0
	be2.SetHealth(false)

	var responses []*httptest.ResponseRecorder

	req1 := httptest.NewRequest("GET", "/stream", nil)
	rr1 := httptest.NewRecorder()
	go func() {
		proxy.ServeHTTP(rr1, req1)
	}()

	time.Sleep(10 * time.Millisecond)

	req2 := httptest.NewRequest("GET", "/stream", nil)
	rr2 := httptest.NewRecorder()
	go func() {
		proxy.ServeHTTP(rr2, req2)
	}()

	time.Sleep(10 * time.Millisecond)

	if be1.GetConnections() != 2 {
		t.Errorf("Expected backend1 to have 2 connections, got %d", be1.GetConnections())
	}
	if be2.GetConnections() != 0 {
		t.Errorf("Expected backend2 to have 0 connections, got %d", be2.GetConnections())
	}

	be2.SetHealth(true)

	// Third request - it should go to backend2 (least connections)
	req3 := httptest.NewRequest("GET", "/test", nil)
	rr3 := httptest.NewRecorder()
	proxy.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Errorf("Expected status 200 for third request, got %d", rr3.Code)
	}

	if !strings.Contains(rr3.Body.String(), "backend2-response") {
		t.Errorf("Expected response from backend2, got: %s", rr3.Body.String())
	}

	time.Sleep(100 * time.Millisecond)
	if be1.GetConnections() != 2 {
		t.Errorf("Expected backend1 to still have 2 connections, got %d", be1.GetConnections())
	}
	if be2.GetConnections() != 0 {
		t.Errorf("Expected backend2 to have 0 connections after request completion, got %d", be2.GetConnections())
	}

	time.Sleep(1 * time.Second)
	responses = append(responses, rr1, rr2)
}
