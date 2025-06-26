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
func NewHandler(client client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		Search:    search.NewHandler(client, logger),
		Tables:    tables.NewHandler(client, logger),
		Documents: documents.NewHandler(client, logger),
		Clusters:  clusters.NewHandler(client, logger),
		logger:    logger,
	}
}
