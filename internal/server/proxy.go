package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/novaru/golem/internal/balancer"
)

// ProxyServer is a simple HTTP reverse proxy that uses a load balancer to distribute requests
// across multiple backend servers. It handles incoming HTTP requests, forwards them to them
// selected backend, and returns the response to the client. It also manages backend health
// checks and connection counts.
type ProxyServer struct {
	Balancer balancer.Balancer
}

// NewProxyServer creates a new instance of ProxyServer with the provided load balancer.
func NewProxyServer(bal balancer.Balancer) *ProxyServer {
	return &ProxyServer{Balancer: bal}
}

// ServeHTTP implements the http.Handler interface for ProxyServer.
// It processes incoming HTTP requests, selects a backend using the load balancer,
// and forwards the request to the selected backend server.
func (ps *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := ps.Balancer.NextBackend()
	if backend == nil {
		http.Error(w, "No healthy backend available", http.StatusServiceUnavailable)
		return
	}

	targetURL, err := url.Parse(backend.URL)
	if err != nil {
		http.Error(w, "Invalid backend URL", http.StatusInternalServerError)
		return
	}

	backend.AddConnections()
	connectionRemoved := false

	// Function to safely remove connection once
	removeConnection := func() {
		if !connectionRemoved {
			backend.RemoveConnections()
			connectionRemoved = true
			log.Printf("[INFO] Removed connection from backend: %s (current connections: %d)", backend.URL, backend.GetConnections())
		}
	}

	defer removeConnection()

	// Log which backend is selected for the request
	log.Printf("[INFO] Forwarding %s %s to backend: %s (current connections: %d)", r.Method, r.URL.Path, backend.URL, backend.GetConnections())

	dest := *targetURL
	dest.Path = r.URL.Path
	dest.RawQuery = r.URL.RawQuery

	// Prepare request to backend
	proxyReq, err := http.NewRequest(r.Method, dest.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header.Clone()

	client := &http.Client{}
	if dest.Path == "/stream" {
		client.Timeout = 0
	} else {
		client.Timeout = 10 * time.Second
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		removeConnection()
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		backend.SetHealth(false)
		log.Printf("[ERROR] Backend %s is unavailable: %v", backend.URL, err)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// Forward response headers and status
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, supportsFlushing := w.(http.Flusher)

	// Handle streaming vs regular responses
	// This simulates a scenario where the backend can return a streaming response
	if supportsFlushing && resp.Header.Get("Content-Type") == "text/plain" {
		// Streaming response - read line by line and flush immediately
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Fprintln(w, line)
			if err != nil {
				log.Printf("[INFO] Client disconnected during streaming to %s", backend.URL)
				removeConnection()
				return
			}
			flusher.Flush()
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[INFO] Streaming ended or client disconnected: %v", err)
			removeConnection()
		}
	} else {
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("[INFO] Client disconnected during response: %v", err)
			removeConnection()
		}
	}
}
