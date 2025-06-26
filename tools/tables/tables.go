package tables

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"manticore-mcp-server/client"
)

// Handler handles table management operations
type Handler struct {
	client client.ManticoreClient
	logger *slog.Logger
}

// NewHandler creates a new table handler
func NewHandler(client client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		client: client,
		logger: logger,
	}
}

// ShowTablesArgs represents arguments for show_tables tool
type ShowTablesArgs struct {
	Pattern string `json:"pattern,omitempty" description:"Optional LIKE pattern to filter table names"`
	Cluster string `json:"cluster,omitempty" description:"Cluster name (optional)"`
}

// DescribeTableArgs represents arguments for describe_table tool
type DescribeTableArgs struct {
	Table   string `json:"table" description:"Table name to describe"`
	Cluster string `json:"cluster,omitempty" description:"Cluster name (optional)"`
}

// ShowTables lists all tables in Manticore
func (h *Handler) ShowTables(ctx context.Context, args ShowTablesArgs) ([]map[string]interface{}, error) {
	var sql strings.Builder
	sql.WriteString("SHOW TABLES")

	if args.Pattern != "" {
		sql.WriteString(" LIKE '")
		// Escape single quotes in pattern according to Manticore documentation
		escapedPattern := strings.ReplaceAll(args.Pattern, "\\", "\\\\") // Escape backslashes first
		escapedPattern = strings.ReplaceAll(escapedPattern, "'", "\\'")  // Escape single quotes
		sql.WriteString(escapedPattern)
		sql.WriteString("'")
	}

	h.logger.Debug("Executing show tables query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("show tables failed: %w", err)
	}

	return result, nil
}

// DescribeTable shows table structure
func (h *Handler) DescribeTable(ctx context.Context, args DescribeTableArgs) ([]map[string]interface{}, error) {
	if args.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	// Build table name with cluster prefix if provided
	tableName := h.buildTableName(args.Cluster, args.Table)
	// Note: Table names in DESCRIBE don't need escaping as they're identifiers, not string literals
	sql := "DESCRIBE " + tableName

	h.logger.Debug("Executing describe table query", "sql", sql)

	result, err := h.client.ExecuteSQL(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("describe table failed: %w", err)
	}

	return result, nil
}

// buildTableName constructs table name with cluster prefix if provided
func (h *Handler) buildTableName(cluster, table string) string {
	if cluster != "" {
		return cluster + ":" + table
	}
	return table
}
