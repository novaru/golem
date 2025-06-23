package config

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

// Supported methods for load balancing
var SupportedMethods = map[string]bool{
	"roundrobin": true,
	"leastconn":  true,
	"weighted":   true,
}

// StringSlice is a custom type that implements flag.Value interface
// to handle a slice of strings for command-line flags.
type StringSlice []string

// Returns the elements of the StringSlice as a comma-separated string.
func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

// Set implements the flag.Value interface for StringSlice.
// It allows setting multiple values from a comma-separated string or repeated flags.
func (s *StringSlice) Set(value string) error {
	if value == "" {
		return nil
	}
	parts := strings.SplitSeq(value, ",")
	for part := range parts {
		*s = append(*s, strings.TrimSpace(part))
	}
	return nil
}

type Config struct {
	Port     int
	Backends StringSlice
	Method   string
}

// ParseFlags parses command-line flags and returns a Config struct.
// It uses the flag package to define and parse the flags.
// The flags include port, backend URLs, and load balancing method.
func ParseFlags() (*Config, error) {
	var cfg Config
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.Var(
		&cfg.Backends,
		"backend",
		"Backend server URL (comma-separated or repeated)",
	)
	flag.StringVar(
		&cfg.Method,
		"method",
		"roundrobin", // Default load balancing method.
		"Load balancing method (roundrobin, leastconn, weighted)",
	)
	flag.Parse()
	return &cfg, cfg.Validate()
}

// Validate checks the configuration for correctness.
// It ensures that at least one backend is specified, the method is supported,
// and the port is within the valid range (1-65535).
func (c *Config) Validate() error {
	if len(c.Backends) == 0 {
		return errors.New("at least one backend must be specified")
	}
	if !SupportedMethods[c.Method] {
		return fmt.Errorf("unsupported load balancing method: %s", c.Method)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	return nil
}
