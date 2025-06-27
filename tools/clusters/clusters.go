package clusters

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"manticore-mcp-server/client"
)

// Handler handles cluster operations
type Handler struct {
	client client.ManticoreClient
	logger *slog.Logger
}

// NewHandler creates a new cluster handler
func NewHandler(c client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		client: c,
		logger: logger,
	}
}

// CreateClusterArgs represents arguments for create_cluster tool
type CreateClusterArgs struct {
	Name  string   `json:"name" description:"Cluster name"`
	Path  string   `json:"path,omitempty" description:"Data directory path (optional)"`
	Nodes []string `json:"nodes,omitempty" description:"List of nodes (host:port format)"`
}

// JoinClusterArgs represents arguments for join_cluster tool
type JoinClusterArgs struct {
	Name  string   `json:"name" description:"Cluster name to join"`
	At    string   `json:"at" description:"Address of existing cluster node (host:port)"`
	Nodes []string `json:"nodes,omitempty" description:"Explicit list of cluster nodes (optional)"`
	Path  string   `json:"path,omitempty" description:"Custom path for cluster data (optional)"`
}

// AlterClusterArgs represents arguments for alter_cluster tool
type AlterClusterArgs struct {
	Name      string   `json:"name" description:"Cluster name"`
	Operation string   `json:"operation" description:"Operation: add, drop, update_nodes"`
	Table     string   `json:"table,omitempty" description:"Table name (for add/drop operations)"`
	Nodes     []string `json:"nodes,omitempty" description:"Nodes list (for update_nodes operation)"`
}

// DeleteClusterArgs represents arguments for delete_cluster tool
type DeleteClusterArgs struct {
	Name string `json:"name" description:"Cluster name to delete"`
}

// ShowClusterStatusArgs represents arguments for show_cluster_status tool
type ShowClusterStatusArgs struct {
	Pattern string `json:"pattern,omitempty" description:"Optional LIKE pattern to filter status variables"`
}

// SetClusterArgs represents arguments for set_cluster tool
type SetClusterArgs struct {
	Name     string `json:"name" description:"Cluster name"`
	Variable string `json:"variable" description:"Cluster variable name"`
	Value    string `json:"value" description:"Variable value"`
	Global   bool   `json:"global,omitempty" description:"Set as global variable"`
}

// CreateCluster creates a new replication cluster
func (h *Handler) CreateCluster(ctx context.Context, args CreateClusterArgs) ([]map[string]interface{}, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	var sql strings.Builder
	sql.WriteString("CREATE CLUSTER ")
	sql.WriteString(args.Name)

	// Add path if provided
	if args.Path != "" {
		sql.WriteString(" '")
		sql.WriteString(strings.ReplaceAll(args.Path, "'", "''"))
		sql.WriteString("' AS path")
	}

	// Add nodes if provided
	if len(args.Nodes) > 0 {
		if args.Path != "" {
			sql.WriteString(", ")
		} else {
			sql.WriteString(" ")
		}
		sql.WriteString("'")
		sql.WriteString(strings.Join(args.Nodes, ","))
		sql.WriteString("' AS nodes")
	}

	h.logger.Debug("Executing create cluster query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("create cluster failed: %w", err)
	}

	return result, nil
}

// JoinCluster joins an existing replication cluster
func (h *Handler) JoinCluster(ctx context.Context, args JoinClusterArgs) ([]map[string]interface{}, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	if args.At == "" && len(args.Nodes) == 0 {
		return nil, fmt.Errorf("either 'at' address or 'nodes' list is required")
	}

	var sql strings.Builder
	sql.WriteString("JOIN CLUSTER ")
	sql.WriteString(args.Name)

	if args.At != "" {
		sql.WriteString(" AT '")
		sql.WriteString(strings.ReplaceAll(args.At, "'", "''"))
		sql.WriteString("'")
	} else if len(args.Nodes) > 0 {
		sql.WriteString(" '")
		sql.WriteString(strings.Join(args.Nodes, ";"))
		sql.WriteString("' AS nodes")
	}

	// Add custom path if provided
	if args.Path != "" {
		sql.WriteString(" '")
		sql.WriteString(strings.ReplaceAll(args.Path, "'", "''"))
		sql.WriteString("' AS path")
	}

	h.logger.Debug("Executing join cluster query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("join cluster failed: %w", err)
	}

	return result, nil
}

// AlterCluster modifies cluster configuration
func (h *Handler) AlterCluster(ctx context.Context, args AlterClusterArgs) ([]map[string]interface{}, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	if args.Operation == "" {
		return nil, fmt.Errorf("operation is required")
	}

	var sql strings.Builder
	sql.WriteString("ALTER CLUSTER ")
	sql.WriteString(args.Name)

	switch strings.ToLower(args.Operation) {
	case "add":
		if args.Table == "" {
			return nil, fmt.Errorf("table name is required for add operation")
		}
		sql.WriteString(" ADD ")
		sql.WriteString(args.Table)

	case "drop":
		if args.Table == "" {
			return nil, fmt.Errorf("table name is required for drop operation")
		}
		sql.WriteString(" DROP ")
		sql.WriteString(args.Table)

	case "update_nodes":
		sql.WriteString(" UPDATE nodes")

	default:
		return nil, fmt.Errorf("unsupported operation: %s", args.Operation)
	}

	h.logger.Debug("Executing alter cluster query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("alter cluster failed: %w", err)
	}

	return result, nil
}

// DeleteCluster deletes a replication cluster
func (h *Handler) DeleteCluster(ctx context.Context, args DeleteClusterArgs) ([]map[string]interface{}, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	sql := "DELETE CLUSTER " + args.Name

	h.logger.Debug("Executing delete cluster query", "sql", sql)

	result, err := h.client.ExecuteSQL(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("delete cluster failed: %w", err)
	}

	return result, nil
}

// ShowClusterStatus shows cluster status and configuration
func (h *Handler) ShowClusterStatus(ctx context.Context, args ShowClusterStatusArgs) ([]map[string]interface{}, error) {
	var sql strings.Builder
	sql.WriteString("SHOW STATUS")

	if args.Pattern != "" {
		sql.WriteString(" LIKE '")
		escapedPattern := strings.ReplaceAll(args.Pattern, "'", "''")
		sql.WriteString(escapedPattern)
		sql.WriteString("'")
	}

	h.logger.Debug("Executing show cluster status query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("show cluster status failed: %w", err)
	}

	return result, nil
}

// SetCluster sets cluster variables
func (h *Handler) SetCluster(ctx context.Context, args SetClusterArgs) ([]map[string]interface{}, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}
	if args.Variable == "" {
		return nil, fmt.Errorf("variable name is required")
	}
	if args.Value == "" {
		return nil, fmt.Errorf("variable value is required")
	}

	var sql strings.Builder
	sql.WriteString("SET CLUSTER ")
	sql.WriteString(args.Name)

	if args.Global {
		sql.WriteString(" GLOBAL")
	}

	sql.WriteString(" '")
	sql.WriteString(strings.ReplaceAll(args.Variable, "'", "''"))
	sql.WriteString("' = ")

	// Handle different value types
	if h.isNumeric(args.Value) {
		sql.WriteString(args.Value)
	} else {
		sql.WriteString("'")
		sql.WriteString(strings.ReplaceAll(args.Value, "'", "''"))
		sql.WriteString("'")
	}

	h.logger.Debug("Executing set cluster query", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("set cluster failed: %w", err)
	}

	return result, nil
}

// isNumeric checks if a string represents a numeric value
func (h *Handler) isNumeric(s string) bool {
	if s == "" {
		return false
	}

	dotCount := 0
	for i, char := range s {
		if i == 0 && char == '-' {
			continue // Allow negative numbers
		}
		if char == '.' {
			dotCount++
			if dotCount > 1 {
				return false // Multiple dots not allowed
			}
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
