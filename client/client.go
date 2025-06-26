package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"manticore-mcp-server/config"
	"manticore-mcp-server/types"
)

//go:generate moq -out client_mock.go . ManticoreClient

// ManticoreClient defines the interface for Manticore Search operations
type ManticoreClient interface {
	ExecuteSQL(ctx context.Context, query string) ([]map[string]interface{}, error)
	Ping(ctx context.Context) error
}

// Client provides access to Manticore Search API
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	maxRetries int
	retryDelay time.Duration
}

// New creates a new Manticore client
func New(cfg *config.Config, logger *slog.Logger) ManticoreClient {
	return &Client{
		baseURL: cfg.ManticoreURL,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		logger:     logger,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
	}
}

// ExecuteSQL executes a SQL query against Manticore
func (c *Client) ExecuteSQL(ctx context.Context, query string) ([]map[string]interface{}, error) {
	endpoint := "/sql?mode=raw"

	respBody, err := c.doRawRequest(ctx, "POST", endpoint, query)
	if err != nil {
		return nil, fmt.Errorf("SQL request failed: %w", err)
	}

	// Response is an array of result sets in raw mode
	results, ok := respBody.([]interface{})
	if !ok || len(results) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Get first result set
	firstResult, ok := results[0].(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	data, ok := firstResult["data"].([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	result := make([]map[string]interface{}, len(data))
	for i, item := range data {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result[i] = row
	}

	return result, nil
}

// Ping checks if Manticore server is reachable
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ExecuteSQL(ctx, "SHOW STATUS")
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

func (c *Client) doRawRequest(ctx context.Context, method, endpoint, query string) (interface{}, error) {
	url := c.baseURL + endpoint

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("Retrying request", "attempt", attempt, "url", url)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader([]byte(query)))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 && attempt < c.maxRetries {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, &types.HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(bodyBytes),
			}
		}

		var result interface{}
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}
