package testutils

import (
	"os"
	"time"

	"manticore-mcp-server/config"
)

// LoadTestConfig loads configuration for tests from environment or returns default test config
func LoadTestConfig() *config.Config {
	// Try to load from environment first
	if cfg, err := config.Load(); err == nil {
		return cfg
	}

	// Fallback to default test configuration
	manticoreURL := os.Getenv("MANTICORE_URL")
	if manticoreURL == "" {
		manticoreURL = "http://localhost:19308" // Default for local docker-compose testing
	}

	return &config.Config{
		ManticoreURL:       manticoreURL,
		RequestTimeout:     30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		MaxResultsPerQuery: 100,
	}
}
