package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
)

type Config struct {
	ManticoreURL    string        `long:"manticore-url" env:"MANTICORE_URL" default:"http://localhost:9308" description:"Manticore Search server URL"`
	RequestTimeout  time.Duration `long:"request-timeout" env:"REQUEST_TIMEOUT" default:"30s" description:"HTTP request timeout"`
	MaxRetries      int           `long:"max-retries" env:"MAX_RETRIES" default:"3" description:"Maximum number of retry attempts"`
	RetryDelay      time.Duration `long:"retry-delay" env:"RETRY_DELAY" default:"1s" description:"Delay between retry attempts"`
	EnvFile         string        `long:"env-file" description:"Path to .env file for local development"`
	Debug           bool          `long:"debug" env:"DEBUG" description:"Enable debug logging"`
}

func Load() (*Config, error) {
	var cfg Config

	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if cfg.EnvFile != "" {
		if err := godotenv.Load(cfg.EnvFile); err != nil {
			slog.Warn("Failed to load .env file", "file", cfg.EnvFile, "error", err)
		}
	} else {
		_ = godotenv.Load()
	}

	if _, err := parser.Parse(); err != nil {
		return nil, fmt.Errorf("failed to parse config after loading env: %w", err)
	}

	return &cfg, nil
}