package balancer

import (
	"net/http"
	"time"
)

// HealthChecker periodically checks backend health.
type HealthChecker struct {
	Backends []*Backend
	Interval time.Duration
	StopChan chan struct{}
}

// NewHealthChecker creates a new HealthChecker instance.
func NewHealthChecker(backends []*Backend, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		Backends: backends,
		Interval: interval,
		StopChan: make(chan struct{}),
	}
}

// Start begins the health checking process.
// It runs in a separate goroutine and checks each backend at the specified interval.
func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(hc.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hc.checkAll()
			case <-hc.StopChan:
				return
			}
		}
	}()
}

// Stop stops the health checking process.
func (hc *HealthChecker) Stop() {
	hc.StopChan <- struct{}{}
}

// checkAll iterates over all backends and checks their health status.
func (hc *HealthChecker) checkAll() {
	for _, b := range hc.Backends {
		go hc.checkBackend(b)
	}
}

// checkBackend checks the health of a single backend.
func (hc *HealthChecker) checkBackend(b *Backend) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(b.URL + "/health")
	if err != nil || resp.StatusCode >= 400 {
		b.SetHealth(false)
	} else {
		b.SetHealth(true)
	}
	if resp != nil {
		resp.Body.Close()
	}
}
