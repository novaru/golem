package main

import (
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
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	metrics.SetLoadBalancerInfo("v1.0.0", cfg.Method)

	backends := []*balancer.Backend{}
	for _, url := range cfg.Backends {
		backend := balancer.NewBackend(url, 1)
		backends = append(backends, backend)
	}

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
