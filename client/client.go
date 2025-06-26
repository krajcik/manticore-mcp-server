package client

import (
	"log/slog"
)

// Client provides access to Manticore Search API
type Client struct {
	logger *slog.Logger
}

// New creates a new Manticore client
func New(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		logger: logger,
	}
}
