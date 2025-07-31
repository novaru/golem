package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type BackendConfig struct {
	URL    string `json:"url"`
	Weight int    `json:"weight,omitempty"`
}

// FileConfig represents configuration loaded from a file
type FileConfig struct {
	Port     int             `json:"port"`
	Backends []BackendConfig `json:"backends"`
	Method   string          `json:"method"`
}

// LoadConfigFromFile loads config from a JSON file
// returns the main Config and a map[URL]weight
func LoadConfigFromFile(path string) (*Config, map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var fileConfig struct {
		Port     int             `json:"port"`
		Backends []BackendConfig `json:"backends"`
		Method   string          `json:"method"`
	}

	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var urls []string
	weights := make(map[string]int)

	for _, b := range fileConfig.Backends {
		urls = append(urls, b.URL)
		if b.Weight <= 0 {
			weights[b.URL] = 1
		} else {
			weights[b.URL] = b.Weight
		}
	}

	config := &Config{
		Port:     fileConfig.Port,
		Backends: StringSlice(urls),
		Method:   fileConfig.Method,
	}

	if err := config.Validate(); err != nil {
		return nil, nil, err
	}

	return config, weights, nil
}

// FindConfigFile looks for a config file in standard locations
func FindConfigFile() (string, error) {
	locations := []string{
		"./golem.json",
		"./config.json",
		"/etc/golem/config.json",
		filepath.Join(os.Getenv("HOME"), ".golem/config.json"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", os.ErrNotExist
}
