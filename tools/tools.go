package tools

import (
	"log/slog"

	"manticore-mcp-server/client"
	"manticore-mcp-server/tools/clusters"
	"manticore-mcp-server/tools/documents"
	"manticore-mcp-server/tools/search"
	"manticore-mcp-server/tools/tables"
)

// Handler aggregates all tool handlers
type Handler struct {
	Search    *search.Handler
	Tables    *tables.Handler
	Documents *documents.Handler
	Clusters  *clusters.Handler
	logger    *slog.Logger
}

// NewHandler creates a new aggregated tool handler
func NewHandler(c client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		Search:    search.NewHandler(c, logger),
		Tables:    tables.NewHandler(c, logger),
		Documents: documents.NewHandler(c, logger),
		Clusters:  clusters.NewHandler(c, logger),
		logger:    logger,
	}
}
