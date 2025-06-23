package main

import (
	"fmt"
	"os"

	"github.com/novaru/golem/config"
)

func main() {
	cfg, err := config.ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Parsed config: port=%d, backends=%v, method=%s\n", cfg.Port, cfg.Backends, cfg.Method)
}
