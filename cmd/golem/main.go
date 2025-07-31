package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/novaru/golem/config"
	"github.com/novaru/golem/internal/balancer"
	"github.com/novaru/golem/internal/metrics"
	"github.com/novaru/golem/internal/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var cfg *config.Config
	var backendWeights map[string]int
	var err error

	configFile, err := config.FindConfigFile()
	if err == nil {
		cfg, backendWeights, err = config.LoadConfigFromFile(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file %s: %v", configFile, err)
		}
		fmt.Printf("Loaded config from %s\n", configFile)
	} else {
		backendWeights = make(map[string]int)
		cfg = &config.Config{
			Port:     8080,
			Backends: nil,
			Method:   "roundrobin",
		}
	}

	originalPort := cfg.Port
	originalMethod := cfg.Method

	// Parse flags (override file)
	flag.IntVar(&cfg.Port, "port", originalPort, "Port to listen on")
	flag.Var(&cfg.Backends, "backend", "Backend server URL (comma-separated or repeated)")
	flag.StringVar(&cfg.Method, "method", originalMethod, "Load balancing method")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	backends := []*balancer.Backend{}
	for _, url := range cfg.Backends {
		weight := 1
		if w, found := backendWeights[url]; found {
			weight = w
		}
		backends = append(backends, balancer.NewBackend(url, weight))
	}

	metrics.SetLoadBalancerInfo("v1.0.0", cfg.Method)

	bal, err := balancer.NewBalancer(cfg.Method, backends)
	if err != nil {
		log.Fatalf("Failed to create new balancer: %v", err)
	}

	healthChecker := balancer.NewHealthChecker(backends, 5*time.Second)
	healthChecker.Start()
	defer healthChecker.Stop()

	proxy := server.NewProxyServer(bal)
	addr := fmt.Sprintf(":%d", cfg.Port)

	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	mux.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Listening on %s, backends=%v, method=%s\n", addr, cfg.Backends, cfg.Method)
	log.Fatal(http.ListenAndServe(addr, mux))
}
