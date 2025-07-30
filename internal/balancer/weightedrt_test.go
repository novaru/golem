package balancer

import (
	"testing"
	"time"
)

func TestWeightedResponseTimeBalancer_NextBackend(t *testing.T) {
	tests := []struct {
		name                 string
		backends             []*Backend
		responseTimeSetup    []responseTimeSetup
		healthSetup          []healthSetup
		expectedDistribution map[string]float64 // Expected distribution percentage (approximate)
		iterations           int
		tolerance            float64 // Tolerance for distribution verification
		seed                 int64   // Fixed seed for reproducible tests
	}{
		{
			name:                 "no backends",
			backends:             []*Backend{},
			expectedDistribution: map[string]float64{},
			iterations:           100,
			seed:                 12345,
		},
		{
			name: "all backends unhealthy",
			backends: []*Backend{
				NewBackend("http://backend1:8080", 1),
				NewBackend("http://backend2:8080", 1),
			},
			healthSetup: []healthSetup{
				{backendURL: "http://backend1:8080", healthy: false},
				{backendURL: "http://backend2:8080", healthy: false},
			},
			expectedDistribution: map[string]float64{},
			iterations:           100,
			seed:                 12345,
		},
		{
			name: "single healthy backend",
			backends: []*Backend{
				NewBackend("http://backend1:8080", 1),
				NewBackend("http://backend2:8080", 1),
			},
			healthSetup: []healthSetup{
				{backendURL: "http://backend1:8080", healthy: true},
				{backendURL: "http://backend2:8080", healthy: false},
			},
			expectedDistribution: map[string]float64{
				"http://backend1:8080": 100.0,
			},
			iterations: 100,
			tolerance:  0.0,
			seed:       12345,
		},
		{
			name: "equal response times should distribute evenly",
			backends: []*Backend{
				NewBackend("http://backend1:8080", 1),
				NewBackend("http://backend2:8080", 1),
			},
			responseTimeSetup: []responseTimeSetup{
				{backendURL: "http://backend1:8080", responseTime: 100 * time.Millisecond, requestCount: 10},
				{backendURL: "http://backend2:8080", responseTime: 100 * time.Millisecond, requestCount: 10},
			},
			expectedDistribution: map[string]float64{
				"http://backend1:8080": 50.0,
				"http://backend2:8080": 50.0,
			},
			iterations: 10000,
			tolerance:  5.0,
			seed:       12345,
		},
		{
			name: "different response times should favor faster backend",
			backends: []*Backend{
				NewBackend("http://fast:8080", 1),
				NewBackend("http://slow:8080", 1),
			},
			responseTimeSetup: []responseTimeSetup{
				{backendURL: "http://fast:8080", responseTime: 50 * time.Millisecond, requestCount: 10},
				{backendURL: "http://slow:8080", responseTime: 200 * time.Millisecond, requestCount: 10},
			},
			// Calculated weights: fast = 1000/(50+1) ≈ 19.6, slow = 1000/(200+1) ≈ 5.0
			// Distribution: fast = 19.6/(19.6+5.0) ≈ 79.7%, slow ≈ 20.3%
			expectedDistribution: map[string]float64{
				"http://fast:8080": 79.7,
				"http://slow:8080": 20.3,
			},
			iterations: 10000,
			tolerance:  5.0,
			seed:       12345,
		},
		{
			name: "very different response times",
			backends: []*Backend{
				NewBackend("http://very-fast:8080", 1),
				NewBackend("http://medium:8080", 1),
				NewBackend("http://very-slow:8080", 1),
			},
			responseTimeSetup: []responseTimeSetup{
				{backendURL: "http://very-fast:8080", responseTime: 10 * time.Millisecond, requestCount: 20},
				{backendURL: "http://medium:8080", responseTime: 100 * time.Millisecond, requestCount: 20},
				{backendURL: "http://very-slow:8080", responseTime: 500 * time.Millisecond, requestCount: 20},
			},
			// Calculated weights: very-fast = 1000/(10+1) ≈ 90.9, medium = 1000/(100+1) ≈ 9.9, very-slow = 1000/(500+1) ≈ 2.0
			// Total = 102.8, Distribution: very-fast ≈ 88.4%, medium ≈ 9.6%, very-slow ≈ 1.9%
			expectedDistribution: map[string]float64{
				"http://very-fast:8080": 88.4,
				"http://medium:8080":    9.6,
				"http://very-slow:8080": 1.9,
			},
			iterations: 10000,
			tolerance:  5.0,
			seed:       12345,
		},
		{
			name: "new backend with no history gets default weight",
			backends: []*Backend{
				NewBackend("http://established:8080", 1),
				NewBackend("http://new:8080", 1),
			},
			responseTimeSetup: []responseTimeSetup{
				{backendURL: "http://established:8080", responseTime: 200 * time.Millisecond, requestCount: 20},
				// "new" backend has no recorded response times (gets default weight of 10.0)
			},
			// Calculated weights: established = 1000/(200+1) ≈ 5.0, new = 10.0
			// Total = 15.0, Distribution: established ≈ 33.3%, new ≈ 66.7%
			expectedDistribution: map[string]float64{
				"http://established:8080": 33.3,
				"http://new:8080":         66.7,
			},
			iterations: 10000,
			tolerance:  5.0,
			seed:       12345,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balancer := NewWeightedResponseTimeBalancer(tt.backends)

			// Set fixed seed for reproducible tests
			balancer.SetSeed(tt.seed)

			// Create backend map for easy lookup
			backendMap := make(map[string]*Backend)
			for _, backend := range tt.backends {
				backendMap[backend.URL] = backend
			}

			for _, setup := range tt.healthSetup {
				if backend, exists := backendMap[setup.backendURL]; exists {
					backend.SetHealth(setup.healthy)
				}
			}

			// Setup response times
			for _, setup := range tt.responseTimeSetup {
				if backend, exists := backendMap[setup.backendURL]; exists {
					for i := 0; i < setup.requestCount; i++ {
						balancer.RecordResponseTime(backend, setup.responseTime)
					}
				}
			}

			// DEBUG: Print actual weights for verification
			if testing.Verbose() {
				t.Logf("Weights for test %s:", tt.name)
				totalWeight := 0.0
				weights := make(map[string]float64)
				for _, backend := range tt.backends {
					if backend.IsHealthy() {
						weight := balancer.GetWeight(backend)
						weights[backend.URL] = weight
						totalWeight += weight
						t.Logf("  %s: weight=%.2f", backend.URL, weight)
					}
				}
				t.Logf("  Total weight: %.2f", totalWeight)
				for url, weight := range weights {
					percentage := (weight / totalWeight) * 100.0
					t.Logf("  %s: expected %.1f%%", url, percentage)
				}
			}

			distribution := make(map[string]int)
			nilCount := 0

			for i := 0; i < tt.iterations; i++ {
				backend, err := balancer.NextBackend()
				if err != nil {
					nilCount++
				} else {
					distribution[backend.URL]++
				}
			}

			if len(tt.expectedDistribution) == 0 {
				if nilCount != tt.iterations {
					t.Errorf("Expected all %d iterations to return nil, got %d nil responses", tt.iterations, nilCount)
				}
				return
			}

			// Check distribution percentages
			for expectedBackend, expectedPercentage := range tt.expectedDistribution {
				actualCount := distribution[expectedBackend]
				actualPercentage := float64(actualCount) / float64(tt.iterations) * 100.0

				if abs(actualPercentage-expectedPercentage) > tt.tolerance {
					t.Errorf("Backend %s: expected ~%.1f%%, got %.1f%% (count: %d/%d)",
						expectedBackend, expectedPercentage, actualPercentage, actualCount, tt.iterations)
				}
			}

			// Ensure no unexpected backends were selected
			for backendURL := range distribution {
				if _, expected := tt.expectedDistribution[backendURL]; !expected {
					t.Errorf("Unexpected backend %s was selected %d times", backendURL, distribution[backendURL])
				}
			}
		})
	}
}

// Helper function to calculate expected distributions for a test case
func TestWeightedResponseTimeBalancer_CalculateExpectedDistribution(t *testing.T) {
	t.Helper()

	// This is a helper test to verify our weight calculations
	backends := []*Backend{
		NewBackend("http://fast:8080", 1),
		NewBackend("http://slow:8080", 1),
	}

	balancer := NewWeightedResponseTimeBalancer(backends)

	// Setup response times
	for range 10 {
		balancer.RecordResponseTime(backends[0], 50*time.Millisecond)
		balancer.RecordResponseTime(backends[1], 200*time.Millisecond)
	}

	// Calculate weights
	fastWeight := balancer.GetWeight(backends[0])
	slowWeight := balancer.GetWeight(backends[1])
	totalWeight := fastWeight + slowWeight

	fastPercentage := (fastWeight / totalWeight) * 100.0
	slowPercentage := (slowWeight / totalWeight) * 100.0

	t.Logf("Fast backend: weight=%.2f, percentage=%.1f%%", fastWeight, fastPercentage)
	t.Logf("Slow backend: weight=%.2f, percentage=%.1f%%", slowWeight, slowPercentage)
	t.Logf("Total weight: %.2f", totalWeight)
}

// Rest of the test functions remain the same...
func TestWeightedResponseTimeBalancer_RecordResponseTime(t *testing.T) {
	tests := []struct {
		name          string
		backendURL    string
		responseTimes []time.Duration
		expectedAvg   time.Duration
		tolerance     time.Duration
	}{
		{
			name:          "single response time",
			backendURL:    "http://test:8080",
			responseTimes: []time.Duration{100 * time.Millisecond},
			expectedAvg:   100 * time.Millisecond,
			tolerance:     1 * time.Millisecond,
		},
		{
			name:          "multiple response times",
			backendURL:    "http://test:8080",
			responseTimes: []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 150 * time.Millisecond},
			expectedAvg:   100 * time.Millisecond,
			tolerance:     1 * time.Millisecond,
		},
		{
			name:          "varying response times",
			backendURL:    "http://test:8080",
			responseTimes: []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond, 40 * time.Millisecond},
			expectedAvg:   25 * time.Millisecond,
			tolerance:     1 * time.Millisecond,
		},
		{
			name:          "zero response time",
			backendURL:    "http://test:8080",
			responseTimes: []time.Duration{0},
			expectedAvg:   0,
			tolerance:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewBackend(tt.backendURL, 1)
			backends := []*Backend{backend}
			balancer := NewWeightedResponseTimeBalancer(backends)

			// Record response times
			for _, responseTime := range tt.responseTimes {
				balancer.RecordResponseTime(backend, responseTime)
			}

			// Check average
			avgTime := balancer.GetAverageResponseTime(backend)
			if abs64(int64(avgTime-tt.expectedAvg)) > int64(tt.tolerance) {
				t.Errorf("Expected average response time ~%v, got %v", tt.expectedAvg, avgTime)
			}
		})
	}
}

// Helper types and functions

type responseTimeSetup struct {
	backendURL   string
	responseTime time.Duration
	requestCount int
}

type healthSetup struct {
	backendURL string
	healthy    bool
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
