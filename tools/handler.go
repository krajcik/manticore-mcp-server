package tools

import (
	"log/slog"
)

// Handler manages MCP tool implementations
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new tool handler
func NewHandler(client interface{}, logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}
