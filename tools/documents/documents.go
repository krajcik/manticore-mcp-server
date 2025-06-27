package documents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"manticore-mcp-server/client"
)

// Handler handles document operations
type Handler struct {
	client client.ManticoreClient
	logger *slog.Logger
}

// NewHandler creates a new document handler
func NewHandler(c client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		client: c,
		logger: logger,
	}
}

// InsertDocumentArgs represents arguments for insert_document tool
type InsertDocumentArgs struct {
	Table    string                 `json:"table" description:"Table name to insert into"`
	Cluster  string                 `json:"cluster,omitempty" description:"Cluster name (optional)"`
	Document map[string]interface{} `json:"document" description:"Document fields as key-value pairs"`
	ID       *int64                 `json:"id,omitempty" description:"Document ID (optional, auto-generated if not provided)"`
	Replace  bool                   `json:"replace,omitempty" description:"Use REPLACE instead of INSERT"`
}

// UpdateDocumentArgs represents arguments for update_document tool
type UpdateDocumentArgs struct {
	Table     string                 `json:"table" description:"Table name to update"`
	Cluster   string                 `json:"cluster,omitempty" description:"Cluster name (optional)"`
	ID        int64                  `json:"id" description:"Document ID to update"`
	Document  map[string]interface{} `json:"document" description:"Fields to update as key-value pairs"`
	Condition string                 `json:"condition,omitempty" description:"Additional WHERE condition"`
}

// DeleteDocumentArgs represents arguments for delete_document tool
type DeleteDocumentArgs struct {
	Table     string `json:"table" description:"Table name to delete from"`
	Cluster   string `json:"cluster,omitempty" description:"Cluster name (optional)"`
	ID        *int64 `json:"id,omitempty" description:"Document ID to delete (optional if condition provided)"`
	Condition string `json:"condition,omitempty" description:"WHERE condition for deletion"`
}

// InsertDocument inserts a document into Manticore table
func (h *Handler) InsertDocument(ctx context.Context, args InsertDocumentArgs) ([]map[string]interface{}, error) {
	if args.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}
	if len(args.Document) == 0 {
		return nil, fmt.Errorf("document parameter is required and cannot be empty")
	}

	// Build table name with cluster prefix if provided
	tableName := h.buildTableName(args.Cluster, args.Table)

	// Build INSERT/REPLACE query
	var sql strings.Builder
	if args.Replace {
		sql.WriteString("REPLACE INTO ")
	} else {
		sql.WriteString("INSERT INTO ")
	}
	sql.WriteString(tableName)
	sql.WriteString(" (")

	// Build column list and values
	columns := make([]string, 0, len(args.Document)+1)
	values := make([]string, 0, len(args.Document)+1)

	// Add ID if provided
	if args.ID != nil {
		columns = append(columns, "id")
		values = append(values, fmt.Sprintf("%d", *args.ID))
	}

	// Add document fields
	for column, value := range args.Document {
		// Skip nil values to avoid syntax errors
		if value == nil {
			continue
		}
		columns = append(columns, column)
		values = append(values, h.formatValue(value))
	}

	sql.WriteString(strings.Join(columns, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(values, ", "))
	sql.WriteString(")")

	h.logger.Debug("Executing insert document query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("insert document failed: %w", err)
	}

	return result, nil
}

// UpdateDocument updates a document in Manticore table
func (h *Handler) UpdateDocument(ctx context.Context, args UpdateDocumentArgs) ([]map[string]interface{}, error) {
	if args.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}
	if len(args.Document) == 0 {
		return nil, fmt.Errorf("document parameter is required and cannot be empty")
	}

	// Build table name with cluster prefix if provided
	tableName := h.buildTableName(args.Cluster, args.Table)

	// Build UPDATE query
	var sql strings.Builder
	sql.WriteString("UPDATE ")
	sql.WriteString(tableName)
	sql.WriteString(" SET ")

	// Build SET clause
	setParts := make([]string, 0, len(args.Document))
	for column, value := range args.Document {
		setParts = append(setParts, column+"="+h.formatValue(value))
	}
	sql.WriteString(strings.Join(setParts, ", "))

	// Build WHERE clause
	sql.WriteString(" WHERE id=")
	sql.WriteString(fmt.Sprintf("%d", args.ID))

	// Add additional condition if provided
	if args.Condition != "" {
		sql.WriteString(" AND (")
		sql.WriteString(args.Condition)
		sql.WriteString(")")
	}

	h.logger.Debug("Executing update document query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("update document failed: %w", err)
	}

	return result, nil
}

// DeleteDocument deletes a document from Manticore table
func (h *Handler) DeleteDocument(ctx context.Context, args DeleteDocumentArgs) ([]map[string]interface{}, error) {
	if args.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}
	if args.ID == nil && args.Condition == "" {
		return nil, fmt.Errorf("either id or condition parameter is required")
	}

	// Build table name with cluster prefix if provided
	tableName := h.buildTableName(args.Cluster, args.Table)

	// Build DELETE query
	var sql strings.Builder
	sql.WriteString("DELETE FROM ")
	sql.WriteString(tableName)
	sql.WriteString(" WHERE ")

	var whereParts []string

	// Add ID condition if provided
	if args.ID != nil {
		whereParts = append(whereParts, fmt.Sprintf("id=%d", *args.ID))
	}

	// Add custom condition if provided
	if args.Condition != "" {
		whereParts = append(whereParts, "("+args.Condition+")")
	}

	sql.WriteString(strings.Join(whereParts, " AND "))

	h.logger.Debug("Executing delete document query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("delete document failed: %w", err)
	}

	return result, nil
}

// formatValue converts a value to SQL string representation
func (h *Handler) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes according to Manticore documentation
		escapedValue := strings.ReplaceAll(v, "\\", "\\\\")         // Escape backslashes first
		escapedValue = strings.ReplaceAll(escapedValue, "'", "\\'") // Escape single quotes
		return "'" + escapedValue + "'"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case nil:
		return "NULL"
	default:
		// Convert to string as fallback
		escapedValue := strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")
		return "'" + escapedValue + "'"
	}
}

// buildTableName constructs table name with cluster prefix if provided
func (h *Handler) buildTableName(cluster, table string) string {
	if cluster != "" {
		return cluster + ":" + table
	}
	return table
}
