package config

import (
	"flag"
	"os"
	"reflect"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	// Valid config
	cfg := &Config{Port: 8080, Backends: StringSlice{"http://b1"}, Method: "roundrobin"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}

	// Missing backends
	cfg = &Config{Port: 8080, Backends: StringSlice{}, Method: "roundrobin"}
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected error for missing backends")
	}

	// Invalid method
	cfg = &Config{Port: 8080, Backends: StringSlice{"http://b1"}, Method: "foo"}
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected error for invalid method")
	}

	// Invalid ports
	cfg = &Config{Port: 0, Backends: StringSlice{"http://b1"}, Method: "roundrobin"}
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected error for invalid port")
	}
}

func TestStringSliceFlag(t *testing.T) {
	var s StringSlice
	s.Set("a,b,c")
	expected := StringSlice{"a", "b", "c"}
	if !reflect.DeepEqual(s, expected) {
		t.Errorf("expected %v, got %v", expected, s)
	}
	s = StringSlice{}
	s.Set("a")
	s.Set("b")
	if !reflect.DeepEqual(s, StringSlice{"a", "b"}) {
		t.Errorf("expected [a b], got %v", s)
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs := os.Args // preserve original args and restore later
	defer func() { os.Args = oldArgs }()

	// Define test cases
	tests := []struct {
		name     string
		args     []string
		expected *Config
		wantErr  bool
		errMsg   string
	}{
		{
			name: "custom values",
			args: []string{
				"cmd",
				"-port=9000",
				"-backend=http://b1,http://b2",
				"-method=leastconn",
			},
			expected: &Config{
				Port:     9000,
				Backends: StringSlice{"http://b1", "http://b2"},
				Method:   "leastconn",
			},
			wantErr: false,
		},
		{
			name: "default values",
			args: []string{
				"cmd",
				"-backend=http://default-backend.com",
			},
			expected: &Config{
				Port:     8080, // Default port
				Backends: StringSlice{"http://default-backend.com"},
				Method:   "roundrobin", // Default method
			},
			wantErr: false,
		},
		{
			name: "default values with multiple backend flags",
			args: []string{
				"cmd",
				"-backend=http://backend1.com",
				"-backend=http://backend2.com",
				"-backend=http://backend3.com",
			},
			expected: &Config{
				Port:     8080, // Default port
				Backends: StringSlice{"http://backend1.com", "http://backend2.com", "http://backend3.com"},
				Method:   "roundrobin", // Default method
			},
			wantErr: false,
		},
		{
			name: "error - no backends",
			args: []string{
				"cmd",
				"-port=9000",
				"-method=leastconn",
			},
			expected: nil,
			wantErr:  true,
			errMsg:   "at least one backend must be specified",
		},
		{
			name: "error - invalid method",
			args: []string{
				"cmd",
				"-backend=http://backend.com",
				"-method=invalid_method",
			},
			expected: nil,
			wantErr:  true,
			errMsg:   "unsupported load balancing method: invalid_method",
		},
		{
			name: "error - invalid port",
			args: []string{
				"cmd",
				"-backend=http://backend.com",
				"-port=99999", // Port too large
			},
			expected: nil,
			wantErr:  true,
			errMsg:   "invalid port: 99999",
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test environment
			os.Args = tc.args
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Call the function being tested
			cfg, err := ParseFlags()

			// Check error
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseFlags() expected error, got nil")
				} else if tc.errMsg != "" && err.Error() != tc.errMsg {
					t.Errorf("ParseFlags() error = %v, want %v", err.Error(), tc.errMsg)
				}
				return
			}

			// Check success case
			if err != nil {
				t.Fatalf("ParseFlags() unexpected error: %v", err)
			}

			// Compare expected and actual config values
			if cfg.Port != tc.expected.Port {
				t.Errorf("ParseFlags() Port = %v, want %v", cfg.Port, tc.expected.Port)
			}
			if cfg.Method != tc.expected.Method {
				t.Errorf("ParseFlags() Method = %v, want %v", cfg.Method, tc.expected.Method)
			}
			if !reflect.DeepEqual(cfg.Backends, tc.expected.Backends) {
				t.Errorf("ParseFlags() Backends = %v, want %v", cfg.Backends, tc.expected.Backends)
			}
		})
	}
}
