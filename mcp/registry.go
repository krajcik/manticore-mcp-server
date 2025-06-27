package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"manticore-mcp-server/config"
	"manticore-mcp-server/tools"
	"manticore-mcp-server/tools/clusters"
	"manticore-mcp-server/tools/documents"
	"manticore-mcp-server/tools/search"
	"manticore-mcp-server/tools/tables"

	mcp_golang "github.com/metoro-io/mcp-golang"
)

// Response represents unified response format for all MCP tools
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains metadata about the response
type Meta struct {
	Total     int    `json:"total"`
	Count     int    `json:"count"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Table     string `json:"table,omitempty"`
	Cluster   string `json:"cluster,omitempty"`
	Operation string `json:"operation,omitempty"`
}

// Registry handles MCP tool registration
type Registry struct {
	tools  *tools.Handler
	config *config.Config
	logger *slog.Logger
}

// NewRegistry creates a new MCP tool registry
func NewRegistry(toolsHandler *tools.Handler, cfg *config.Config, logger *slog.Logger) *Registry {
	return &Registry{
		tools:  toolsHandler,
		config: cfg,
		logger: logger,
	}
}

// RegisterAll registers all Manticore tools with MCP server
func (r *Registry) RegisterAll(server *mcp_golang.Server) error {
	r.logger.Info("Registering all Manticore tools with MCP...")

	// Register search tools
	if err := r.registerSearchTools(server); err != nil {
		return fmt.Errorf("failed to register search tools: %w", err)
	}

	// Register table management tools
	if err := r.registerTableTools(server); err != nil {
		return fmt.Errorf("failed to register table tools: %w", err)
	}

	// Register document operation tools
	if err := r.registerDocumentTools(server); err != nil {
		return fmt.Errorf("failed to register document tools: %w", err)
	}

	// Register cluster tools
	if err := r.registerClusterTools(server); err != nil {
		return fmt.Errorf("failed to register cluster tools: %w", err)
	}

	r.logger.Info("All Manticore tools registered successfully")
	return nil
}

// registerSearchTools registers search-related tools
func (r *Registry) registerSearchTools(server *mcp_golang.Server) error {
	// Search tool
	err := server.RegisterTool("search", "Perform full-text search in Manticore index with advanced options",
		func(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
			return r.handleSearchTool(args)
		})
	if err != nil {
		return err
	}

	r.logger.Debug("Search tools registered")
	return nil
}

// registerTableTools registers table management tools
func (r *Registry) registerTableTools(server *mcp_golang.Server) error {
	// Show tables tool
	err := server.RegisterTool("show_tables", "List all tables/indexes in Manticore",
		func(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
			return r.handleShowTablesTool(args)
		})
	if err != nil {
		return err
	}

	// Describe table tool
	err = server.RegisterTool("describe_table", "Get detailed information about table schema",
		func(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
			return r.handleDescribeTableTool(args)
		})
	if err != nil {
		return err
	}

	r.logger.Debug("Table management tools registered")
	return nil
}

// registerDocumentTools registers document operation tools
func (r *Registry) registerDocumentTools(server *mcp_golang.Server) error {
	// Insert document tool
	err := server.RegisterTool("insert_document", "Insert a new document into Manticore index",
		func(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
			return r.handleInsertDocumentTool(args)
		})
	if err != nil {
		return err
	}

	r.logger.Debug("Document operation tools registered")
	return nil
}

// registerClusterTools registers cluster management tools
func (r *Registry) registerClusterTools(server *mcp_golang.Server) error {
	// Show cluster status tool
	err := server.RegisterTool("show_cluster_status", "Show status of cluster nodes",
		func(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
			return r.handleClusterStatusTool(args)
		})
	if err != nil {
		return err
	}

	r.logger.Debug("Cluster tools registered")
	return nil
}

// handleSearchTool processes search requests
func (r *Registry) handleSearchTool(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
	// Convert map to search args struct
	searchArgs, err := r.mapToSearchArgs(args)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Invalid search arguments: %v", err))
	}

	// Apply default limit from config
	if searchArgs.Limit <= 0 {
		searchArgs.Limit = r.config.MaxResultsPerQuery
	}

	// Execute search
	ctx := context.Background()
	results, err := r.tools.Search.Execute(ctx, *searchArgs)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Search failed: %v", err))
	}

	// Create response
	response := &Response{
		Success: true,
		Data:    results,
		Meta: &Meta{
			Total:     len(results),
			Count:     len(results),
			Limit:     searchArgs.Limit,
			Offset:    searchArgs.Offset,
			Table:     searchArgs.Table,
			Cluster:   searchArgs.Cluster,
			Operation: "search",
		},
	}

	return r.successResponse(response)
}

// handleShowTablesTool processes show tables requests
func (r *Registry) handleShowTablesTool(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
	tablesArgs := tables.ShowTablesArgs{
		Pattern: r.getStringArg(args, "pattern"),
		Cluster: r.getStringArg(args, "cluster"),
	}

	ctx := context.Background()
	tablesList, err := r.tools.Tables.ShowTables(ctx, tablesArgs)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Failed to show tables: %v", err))
	}

	response := &Response{
		Success: true,
		Data:    tablesList,
		Meta: &Meta{
			Total:     len(tablesList),
			Count:     len(tablesList),
			Cluster:   tablesArgs.Cluster,
			Operation: "show_tables",
		},
	}

	return r.successResponse(response)
}

// handleDescribeTableTool processes describe table requests
func (r *Registry) handleDescribeTableTool(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
	table := r.getStringArg(args, "table")
	if table == "" {
		return r.errorResponse("Table parameter is required")
	}

	describeArgs := tables.DescribeTableArgs{
		Table:   table,
		Cluster: r.getStringArg(args, "cluster"),
	}

	ctx := context.Background()
	schema, err := r.tools.Tables.DescribeTable(ctx, describeArgs)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Failed to describe table: %v", err))
	}

	response := &Response{
		Success: true,
		Data:    schema,
		Meta: &Meta{
			Table:     describeArgs.Table,
			Cluster:   describeArgs.Cluster,
			Operation: "describe_table",
		},
	}

	return r.successResponse(response)
}

// handleInsertDocumentTool processes document insertion requests
func (r *Registry) handleInsertDocumentTool(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
	table := r.getStringArg(args, "table")
	if table == "" {
		return r.errorResponse("Table parameter is required")
	}

	document, exists := args["document"]
	if !exists {
		return r.errorResponse("Document parameter is required")
	}

	documentMap, ok := document.(map[string]interface{})
	if !ok {
		return r.errorResponse("Document must be a valid object")
	}

	insertArgs := documents.InsertDocumentArgs{
		Table:    table,
		Cluster:  r.getStringArg(args, "cluster"),
		Document: documentMap,
		Replace:  r.getBoolArg(args, "replace"),
	}

	// Handle optional ID
	if idVal := r.getIntArg(args, "id"); idVal != 0 {
		id := int64(idVal)
		insertArgs.ID = &id
	}

	ctx := context.Background()
	result, err := r.tools.Documents.InsertDocument(ctx, insertArgs)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Failed to insert document: %v", err))
	}

	response := &Response{
		Success: true,
		Data:    result,
		Meta: &Meta{
			Table:     insertArgs.Table,
			Cluster:   insertArgs.Cluster,
			Operation: "insert_document",
		},
	}

	return r.successResponse(response)
}

// handleClusterStatusTool processes cluster status requests
func (r *Registry) handleClusterStatusTool(args map[string]interface{}) (*mcp_golang.ToolResponse, error) {
	statusArgs := clusters.ShowClusterStatusArgs{
		Pattern: r.getStringArg(args, "pattern"),
	}

	ctx := context.Background()
	status, err := r.tools.Clusters.ShowClusterStatus(ctx, statusArgs)
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Failed to get cluster status: %v", err))
	}

	response := &Response{
		Success: true,
		Data:    status,
		Meta: &Meta{
			Operation: "cluster_status",
		},
	}

	return r.successResponse(response)
}

// Helper methods

func (r *Registry) successResponse(response *Response) (*mcp_golang.ToolResponse, error) {
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return r.errorResponse(fmt.Sprintf("Failed to serialize response: %v", err))
	}

	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(string(jsonData)),
	), nil
}

func (r *Registry) errorResponse(message string) (*mcp_golang.ToolResponse, error) {
	response := &Response{
		Success: false,
		Error:   message,
	}

	jsonData, _ := json.MarshalIndent(response, "", "  ")
	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(string(jsonData)),
	), nil
}

func (r *Registry) getStringArg(args map[string]interface{}, key string) string {
	if val, exists := args[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (r *Registry) getIntArg(args map[string]interface{}, key string) int {
	if val, exists := args[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// mapToSearchArgs converts map arguments to search Args struct
func (r *Registry) mapToSearchArgs(args map[string]interface{}) (*search.Args, error) {
	searchArgs := &search.Args{
		// Basic parameters
		Query:   r.getStringArg(args, "query"),
		Table:   r.getStringArg(args, "table"),
		Cluster: r.getStringArg(args, "cluster"),

		// Pagination
		Limit:  r.getIntArg(args, "limit"),
		Offset: r.getIntArg(args, "offset"),

		// Field selection
		Fields: r.getStringSliceArg(args, "fields"),

		// Search options
		Ranker:              r.getStringArg(args, "ranker"),
		MatchMode:           r.getStringArg(args, "match_mode"),
		MaxMatches:          r.getIntArg(args, "max_matches"),
		Cutoff:              r.getIntArg(args, "cutoff"),
		MaxQueryTime:        r.getIntArg(args, "max_query_time"),
		FieldWeights:        r.getStringIntMapArg(args, "field_weights"),
		NotTermsOnlyAllowed: r.getIntArg(args, "not_terms_only_allowed"),
		BooleanSimplify:     r.getIntArg(args, "boolean_simplify"),
		AccurateAggregation: r.getIntArg(args, "accurate_aggregation"),
		RandSeed:            r.getIntArg(args, "rand_seed"),
		Comment:             r.getStringArg(args, "comment"),
		AgentQueryTimeout:   r.getIntArg(args, "agent_query_timeout"),
		RetryCount:          r.getIntArg(args, "retry_count"),
		RetryDelay:          r.getIntArg(args, "retry_delay"),
		Morphology:          r.getStringArg(args, "morphology"),
		TokenFilter:         r.getStringArg(args, "token_filter"),
		MaxPredictedTime:    r.getIntArg(args, "max_predicted_time"),

		// Ordering
		OrderBy:   r.getStringSliceArg(args, "order_by"),
		GroupBy:   r.getStringSliceArg(args, "group_by"),
		GroupSort: r.getStringArg(args, "group_sort"),

		// Filtering
		Where: r.getStringSliceArg(args, "where"),

		// Query mode
		UseHTTP: r.getBoolArg(args, "use_http"),
	}

	// Handle highlighting options
	if highlightData, exists := args["highlight"]; exists {
		if highlightMap, ok := highlightData.(map[string]interface{}); ok {
			searchArgs.Highlight = &search.HighlightOptions{
				Enabled:         r.getBoolFromMap(highlightMap, "enabled"),
				Fields:          r.getStringSliceFromMap(highlightMap, "fields"),
				Limit:           r.getIntFromMap(highlightMap, "limit"),
				LimitPerField:   r.getIntFromMap(highlightMap, "limit_per_field"),
				LimitWords:      r.getIntFromMap(highlightMap, "limit_words"),
				Around:          r.getIntFromMap(highlightMap, "around"),
				StartTag:        r.getStringFromMap(highlightMap, "start_tag"),
				EndTag:          r.getStringFromMap(highlightMap, "end_tag"),
				NumberFragments: r.getIntFromMap(highlightMap, "number_of_fragments"),
			}
		}
	}

	// Handle fuzzy options
	if fuzzyData, exists := args["fuzzy"]; exists {
		if fuzzyMap, ok := fuzzyData.(map[string]interface{}); ok {
			searchArgs.Fuzzy = &search.FuzzyOptions{
				Enabled:  r.getBoolFromMap(fuzzyMap, "enabled"),
				Distance: r.getIntFromMap(fuzzyMap, "distance"),
				Preserve: r.getIntFromMap(fuzzyMap, "preserve"),
				Layouts:  r.getStringSliceFromMap(fuzzyMap, "layouts"),
			}
		}
	}

	// Handle boolean query
	if boolQueryData, exists := args["bool_query"]; exists {
		if boolQueryMap, ok := boolQueryData.(map[string]interface{}); ok {
			boolQuery, err := r.mapToBoolQuery(boolQueryMap)
			if err != nil {
				return nil, fmt.Errorf("invalid bool_query: %w", err)
			}
			searchArgs.BoolQuery = boolQuery
		}
	}

	// Validation
	if searchArgs.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	return searchArgs, nil
}

func (r *Registry) getBoolArg(args map[string]interface{}, key string) bool {
	if val, exists := args[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (r *Registry) getStringSliceArg(args map[string]interface{}, key string) []string {
	if val, exists := args[key]; exists {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		// Handle case where it's already []string
		if strSlice, ok := val.([]string); ok {
			return strSlice
		}
	}
	return nil
}

func (r *Registry) getStringIntMapArg(args map[string]interface{}, key string) map[string]int {
	if val, exists := args[key]; exists {
		if mapData, ok := val.(map[string]interface{}); ok {
			result := make(map[string]int)
			for k, v := range mapData {
				switch value := v.(type) {
				case int:
					result[k] = value
				case float64:
					result[k] = int(value)
				}
			}
			return result
		}
	}
	return nil
}

// Helper methods for nested map access
func (r *Registry) getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, exists := m[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (r *Registry) getStringFromMap(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (r *Registry) getIntFromMap(m map[string]interface{}, key string) int {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

func (r *Registry) getStringSliceFromMap(m map[string]interface{}, key string) []string {
	if val, exists := m[key]; exists {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if strSlice, ok := val.([]string); ok {
			return strSlice
		}
	}
	return nil
}

// mapToBoolQuery converts map to BoolQuery structure
func (r *Registry) mapToBoolQuery(boolMap map[string]interface{}) (*search.BoolQuery, error) {
	boolQuery := &search.BoolQuery{}

	// Handle must clauses
	if mustData, exists := boolMap["must"]; exists {
		if mustSlice, ok := mustData.([]interface{}); ok {
			clauses, err := r.mapToQueryClauses(mustSlice)
			if err != nil {
				return nil, fmt.Errorf("invalid must clauses: %w", err)
			}
			boolQuery.Must = clauses
		}
	}

	// Handle should clauses
	if shouldData, exists := boolMap["should"]; exists {
		if shouldSlice, ok := shouldData.([]interface{}); ok {
			clauses, err := r.mapToQueryClauses(shouldSlice)
			if err != nil {
				return nil, fmt.Errorf("invalid should clauses: %w", err)
			}
			boolQuery.Should = clauses
		}
	}

	// Handle must_not clauses
	if mustNotData, exists := boolMap["must_not"]; exists {
		if mustNotSlice, ok := mustNotData.([]interface{}); ok {
			clauses, err := r.mapToQueryClauses(mustNotSlice)
			if err != nil {
				return nil, fmt.Errorf("invalid must_not clauses: %w", err)
			}
			boolQuery.MustNot = clauses
		}
	}

	return boolQuery, nil
}

// mapToQueryClauses converts slice of maps to QueryClause slice
func (r *Registry) mapToQueryClauses(clauseSlice []interface{}) ([]search.QueryClause, error) {
	clauses := make([]search.QueryClause, 0, len(clauseSlice))

	for _, clauseData := range clauseSlice {
		if clauseMap, ok := clauseData.(map[string]interface{}); ok {
			clause := search.QueryClause{}

			// Get type
			if typeVal, exists := clauseMap["type"]; exists {
				if typeStr, ok := typeVal.(string); ok {
					clause.Type = typeStr
				}
			}

			// Get data
			if dataVal, exists := clauseMap["data"]; exists {
				clause.Data = dataVal
			}

			clauses = append(clauses, clause)
		}
	}

	return clauses, nil
}
