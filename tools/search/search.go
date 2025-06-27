package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"manticore-mcp-server/client"
)

var (
	ErrInvalidQueryFormat   = errors.New("invalid query format")
	ErrInvalidMatchQuery    = errors.New("invalid match query format")
	ErrInvalidBoolQuery     = errors.New("invalid bool query format")
	ErrUnsupportedQueryType = errors.New("unsupported query type")
	ErrQueryProcessed       = errors.New("query processed")
)

// Handler handles search-related operations
type Handler struct {
	client client.ManticoreClient
	logger *slog.Logger
}

// NewHandler creates a new search handler
func NewHandler(c client.ManticoreClient, logger *slog.Logger) *Handler {
	return &Handler{
		client: c,
		logger: logger,
	}
}

// Args represents arguments for search tool
type Args struct {
	// Query parameters
	Query   string `json:"query,omitempty" description:"Simple search query text"`
	Table   string `json:"table" description:"Table name to search in"`
	Cluster string `json:"cluster,omitempty" description:"Cluster name (optional)"`

	// Complex boolean query
	BoolQuery *BoolQuery `json:"bool_query,omitempty" description:"Complex boolean query with must/should/must_not clauses"`

	// Pagination
	Limit  int `json:"limit,omitempty" description:"Maximum number of results (default: 10)"`
	Offset int `json:"offset,omitempty" description:"Offset for pagination (default: 0)"`

	// Field selection
	Fields []string `json:"fields,omitempty" description:"Fields to return in results (default: all)"`

	// Search options
	Ranker              string         `json:"ranker,omitempty" description:"Ranking function: proximity_bm25, bm25, none, wordcount, proximity, matchany, fieldmask, sph04, expr, export"`
	MatchMode           string         `json:"match_mode,omitempty" description:"Match mode: all, any, phrase, boolean, extended (default: extended)"`
	MaxMatches          int            `json:"max_matches,omitempty" description:"Maximum matches to retain in RAM (default: 1000)"`
	Cutoff              int            `json:"cutoff,omitempty" description:"Maximum matches to process (0 = no limit)"`
	MaxQueryTime        int            `json:"max_query_time,omitempty" description:"Maximum query time in milliseconds (0 = no limit)"`
	FieldWeights        map[string]int `json:"field_weights,omitempty" description:"Field weight multipliers for ranking"`
	NotTermsOnlyAllowed int            `json:"not_terms_only_allowed,omitempty" description:"Allow queries with only negation (0/1)"`
	BooleanSimplify     int            `json:"boolean_simplify,omitempty" description:"Enable query simplification (0/1, default: 1)"`
	AccurateAggregation int            `json:"accurate_aggregation,omitempty" description:"Guarantee aggregate accuracy (0/1, default: 0)"`
	RandSeed            int            `json:"rand_seed,omitempty" description:"Seed for ORDER BY RAND() queries"`
	Comment             string         `json:"comment,omitempty" description:"User comment for query log"`
	AgentQueryTimeout   int            `json:"agent_query_timeout,omitempty" description:"Remote query timeout in milliseconds"`
	RetryCount          int            `json:"retry_count,omitempty" description:"Distributed retry count"`
	RetryDelay          int            `json:"retry_delay,omitempty" description:"Distributed retry delay in milliseconds"`
	Morphology          string         `json:"morphology,omitempty" description:"Set to 'none' to disable stemming/lemmatizing"`
	TokenFilter         string         `json:"token_filter,omitempty" description:"Query-time token filter (lib:plugin:settings)"`
	MaxPredictedTime    int            `json:"max_predicted_time,omitempty" description:"Maximum predicted search time"`

	// Ordering
	OrderBy   []string `json:"order_by,omitempty" description:"Order by fields (e.g., ['weight() DESC', 'id ASC'])"`
	GroupBy   []string `json:"group_by,omitempty" description:"Group by fields"`
	GroupSort string   `json:"group_sort,omitempty" description:"Group sort expression"`

	// Highlighting
	Highlight *HighlightOptions `json:"highlight,omitempty" description:"Highlighting options"`

	// Fuzzy search
	Fuzzy *FuzzyOptions `json:"fuzzy,omitempty" description:"Fuzzy search options"`

	// Filtering
	Where []string `json:"where,omitempty" description:"Additional WHERE conditions"`

	// Query mode
	UseHTTP bool `json:"use_http,omitempty" description:"Use HTTP JSON API instead of SQL (supports complex boolean queries)"`
}

// HighlightOptions represents highlighting configuration
type HighlightOptions struct {
	Enabled         bool     `json:"enabled,omitempty" description:"Enable highlighting"`
	Fields          []string `json:"fields,omitempty" description:"Fields to highlight (default: all text fields)"`
	Limit           int      `json:"limit,omitempty" description:"Maximum highlighted snippets"`
	LimitPerField   int      `json:"limit_per_field,omitempty" description:"Maximum snippets per field"`
	LimitWords      int      `json:"limit_words,omitempty" description:"Maximum words in snippets"`
	Around          int      `json:"around,omitempty" description:"Words around match"`
	StartTag        string   `json:"start_tag,omitempty" description:"Opening highlight tag (default: <b>)"`
	EndTag          string   `json:"end_tag,omitempty" description:"Closing highlight tag (default: </b>)"`
	NumberFragments int      `json:"number_of_fragments,omitempty" description:"Number of fragments to return"`
}

// FuzzyOptions represents fuzzy search configuration
type FuzzyOptions struct {
	Enabled  bool     `json:"enabled,omitempty" description:"Enable fuzzy search"`
	Distance int      `json:"distance,omitempty" description:"Maximum edit distance (default: 2)"`
	Preserve int      `json:"preserve,omitempty" description:"Preserve original query (0/1)"`
	Layouts  []string `json:"layouts,omitempty" description:"Keyboard layouts for fuzzy matching"`
}

// Execute performs full-text search in Manticore index
func (h *Handler) Execute(ctx context.Context, args Args) ([]map[string]interface{}, error) {
	if args.Table == "" {
		return nil, fmt.Errorf("table parameter is required")
	}

	// Check if we need to use HTTP API for complex queries
	if args.UseHTTP || args.BoolQuery != nil {
		return h.executeHTTPQuery(ctx, args)
	}

	// Use SQL for simple queries
	return h.executeSQLQuery(ctx, args)
}

// executeSQLQuery performs search using SQL interface
func (h *Handler) executeSQLQuery(ctx context.Context, args Args) ([]map[string]interface{}, error) {
	if args.Query == "" && len(args.Where) == 0 {
		return nil, fmt.Errorf("query parameter is required for SQL search when no WHERE conditions are provided")
	}

	// Set defaults
	if args.Limit <= 0 {
		args.Limit = 10
	}
	if args.MatchMode == "" {
		args.MatchMode = "extended"
	}

	// Build the SQL query
	sql, err := h.buildSQL(args)
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	h.logger.Debug("Executing SQL search query", "sql", sql)

	result, err := h.client.ExecuteSQL(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("SQL search failed: %w", err)
	}

	return result, nil
}

// executeHTTPQuery performs search using HTTP JSON API
func (h *Handler) executeHTTPQuery(ctx context.Context, args Args) ([]map[string]interface{}, error) {
	// Set defaults
	if args.Limit <= 0 {
		args.Limit = 10
	}

	// Build HTTP query
	qb := NewQueryBuilder(args.Cluster, args.Table)
	httpQuery, err := qb.BuildHTTPQuery(args)
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP query: %w", err)
	}

	h.logger.Debug("Executing HTTP search query", "query", httpQuery)

	// Convert to JSON and execute via HTTP
	// Note: This would require HTTP client implementation
	// For now, we'll simulate by converting to SQL equivalent
	return h.simulateHTTPQuery(ctx, httpQuery)
}

// simulateHTTPQuery simulates HTTP query execution by converting to SQL
func (h *Handler) simulateHTTPQuery(ctx context.Context, httpQuery map[string]interface{}) ([]map[string]interface{}, error) {
	// This is a simplified simulation - in practice, you'd need a proper HTTP client
	// For complex boolean queries, you might need to use Manticore's HTTP endpoint directly

	table, ok := httpQuery["table"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid table in HTTP query")
	}

	// Build basic SQL from HTTP query
	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")
	sql.WriteString(table)

	// Add simple WHERE clause based on query
	if query, exists := httpQuery["query"]; exists {
		sql.WriteString(" WHERE ")
		if err := h.appendQueryToSQL(&sql, query); err != nil {
			return nil, err
		}
	}

	// Add LIMIT
	if limit, exists := httpQuery["limit"]; exists {
		if limitInt, ok := limit.(int); ok && limitInt > 0 {
			sql.WriteString(" LIMIT ")
			sql.WriteString(strconv.Itoa(limitInt))
		}
	}

	// Add OFFSET
	if offset, exists := httpQuery["offset"]; exists {
		if offsetInt, ok := offset.(int); ok && offsetInt > 0 {
			sql.WriteString(" OFFSET ")
			sql.WriteString(strconv.Itoa(offsetInt))
		}
	}

	h.logger.Debug("Simulated HTTP query as SQL", "sql", sql.String())

	result, err := h.client.ExecuteSQL(ctx, sql.String())
	if err != nil {
		return nil, fmt.Errorf("simulated HTTP search failed: %w", err)
	}

	return result, nil
}

// appendQueryToSQL converts query object to SQL WHERE clause
func (h *Handler) appendQueryToSQL(sql *strings.Builder, query interface{}) error {
	queryMap, ok := query.(map[string]interface{})
	if !ok {
		return ErrInvalidQueryFormat
	}

	if err := h.handleMatchQuery(sql, queryMap); err != nil {
		if errors.Is(err, ErrQueryProcessed) {
			return nil
		}
		return err
	}
	if h.handleMatchAllQuery(sql, queryMap) {
		return nil
	}
	if h.handleQueryStringQuery(sql, queryMap) {
		return nil
	}
	if err := h.handleBoolQuery(sql, queryMap); err != nil {
		if errors.Is(err, ErrQueryProcessed) {
			return nil
		}
		return err
	}

	return ErrUnsupportedQueryType
}

// handleMatchQuery processes match queries
func (h *Handler) handleMatchQuery(sql *strings.Builder, queryMap map[string]interface{}) error {
	match, exists := queryMap["match"]
	if !exists {
		return nil
	}

	matchMap, ok := match.(map[string]interface{})
	if !ok {
		return ErrInvalidMatchQuery
	}

	for field, value := range matchMap {
		if field == "*" {
			sql.WriteString("MATCH('")
			sql.WriteString(strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "''"))
			sql.WriteString("')")
		} else {
			sql.WriteString("MATCH('@")
			sql.WriteString(field)
			sql.WriteString(" ")
			sql.WriteString(strings.ReplaceAll(fmt.Sprintf("%v", value), "'", "''"))
			sql.WriteString("')")
		}
		break // Take first match
	}
	return ErrQueryProcessed
}

// handleMatchAllQuery processes match_all queries
func (h *Handler) handleMatchAllQuery(sql *strings.Builder, queryMap map[string]interface{}) bool {
	if _, exists := queryMap["match_all"]; exists {
		sql.WriteString("1=1") // Match all documents
		return true
	}
	return false
}

// handleQueryStringQuery processes query_string queries
func (h *Handler) handleQueryStringQuery(sql *strings.Builder, queryMap map[string]interface{}) bool {
	if queryString, exists := queryMap["query_string"]; exists {
		sql.WriteString("MATCH('")
		sql.WriteString(strings.ReplaceAll(fmt.Sprintf("%v", queryString), "'", "''"))
		sql.WriteString("')")
		return true
	}
	return false
}

// handleBoolQuery processes bool queries
func (h *Handler) handleBoolQuery(sql *strings.Builder, queryMap map[string]interface{}) error {
	boolQuery, exists := queryMap["bool"]
	if !exists {
		return nil
	}

	boolMap, ok := boolQuery.(map[string]interface{})
	if !ok {
		return ErrInvalidBoolQuery
	}

	conditions, err := h.processMustClauses(boolMap)
	if err != nil {
		return err
	}

	if len(conditions) > 0 {
		sql.WriteString(strings.Join(conditions, " AND "))
	} else {
		sql.WriteString("1=1")
	}
	return ErrQueryProcessed
}

// processMustClauses processes must clauses in bool queries
func (h *Handler) processMustClauses(boolMap map[string]interface{}) ([]string, error) {
	var conditions []string

	if must, exists := boolMap["must"]; exists {
		mustSlice, ok := must.([]interface{})
		if ok {
			for _, mustClause := range mustSlice {
				var subSQL strings.Builder
				if err := h.appendQueryToSQL(&subSQL, mustClause); err != nil {
					return nil, err
				}
				conditions = append(conditions, "("+subSQL.String()+")")
			}
		}
	}

	return conditions, nil
}

// buildSQL constructs the complete SQL query for simple searches
func (h *Handler) buildSQL(args Args) (string, error) {
	var sql strings.Builder

	// SELECT clause
	sql.WriteString("SELECT ")
	if len(args.Fields) > 0 {
		sql.WriteString(strings.Join(args.Fields, ", "))
	} else {
		sql.WriteString("*")
	}

	// Add highlighting if requested
	if args.Highlight != nil && args.Highlight.Enabled {
		highlightFunc := h.buildHighlightFunction(args.Highlight, args.Query)
		if highlightFunc != "" {
			sql.WriteString(", ")
			sql.WriteString(highlightFunc)
		}
	}

	// FROM clause with cluster support
	sql.WriteString(" FROM ")
	tableName := h.buildTableName(args.Cluster, args.Table)
	sql.WriteString(tableName)

	// WHERE clause
	sql.WriteString(" WHERE MATCH('")
	escapedQuery := strings.ReplaceAll(args.Query, "'", "''")
	sql.WriteString(escapedQuery)
	sql.WriteString("')")

	// Additional WHERE conditions
	for _, condition := range args.Where {
		sql.WriteString(" AND (")
		sql.WriteString(condition)
		sql.WriteString(")")
	}

	// GROUP BY clause
	if len(args.GroupBy) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(args.GroupBy, ", "))

		if args.GroupSort != "" {
			sql.WriteString(" ORDER BY ")
			sql.WriteString(args.GroupSort)
		}
	}

	// ORDER BY clause (if not using GROUP BY)
	if len(args.GroupBy) == 0 && len(args.OrderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(args.OrderBy, ", "))
	}

	// LIMIT clause
	if args.Limit > 0 {
		sql.WriteString(" LIMIT ")
		sql.WriteString(strconv.Itoa(args.Limit))
	}

	// OFFSET clause
	if args.Offset > 0 {
		sql.WriteString(" OFFSET ")
		sql.WriteString(strconv.Itoa(args.Offset))
	}

	// OPTION clause
	options := h.buildOptions(args)
	if options != "" {
		sql.WriteString(" OPTION ")
		sql.WriteString(options)
	}

	return sql.String(), nil
}

// buildOptions constructs the OPTION clause
func (h *Handler) buildOptions(args Args) string {
	var options []string

	h.addBasicSQLOptions(&options, args)
	h.addAdvancedSQLOptions(&options, args)
	h.addAgentSQLOptions(&options, args)
	h.addFuzzySQLOptions(&options, args)

	return strings.Join(options, ", ")
}

// addBasicSQLOptions adds basic search options to SQL
func (h *Handler) addBasicSQLOptions(options *[]string, args Args) {
	if args.Ranker != "" {
		*options = append(*options, "ranker="+args.Ranker)
	}
	if args.MaxMatches > 0 {
		*options = append(*options, "max_matches="+strconv.Itoa(args.MaxMatches))
	}
	if args.Cutoff > 0 {
		*options = append(*options, "cutoff="+strconv.Itoa(args.Cutoff))
	}
	if args.MaxQueryTime > 0 {
		*options = append(*options, "max_query_time="+strconv.Itoa(args.MaxQueryTime))
	}
	if len(args.FieldWeights) > 0 {
		weights := make([]string, 0, len(args.FieldWeights))
		for field, weight := range args.FieldWeights {
			weights = append(weights, field+"="+strconv.Itoa(weight))
		}
		*options = append(*options, "field_weights=("+strings.Join(weights, ",")+")")
	}
	if args.Comment != "" {
		*options = append(*options, "comment='"+strings.ReplaceAll(args.Comment, "'", "''")+"'")
	}
}

// addAdvancedSQLOptions adds advanced search options to SQL
func (h *Handler) addAdvancedSQLOptions(options *[]string, args Args) {
	if args.NotTermsOnlyAllowed > 0 {
		*options = append(*options, "not_terms_only_allowed="+strconv.Itoa(args.NotTermsOnlyAllowed))
	}
	if args.BooleanSimplify == 0 {
		*options = append(*options, "boolean_simplify=0")
	}
	if args.AccurateAggregation > 0 {
		*options = append(*options, "accurate_aggregation="+strconv.Itoa(args.AccurateAggregation))
	}
	if args.RandSeed > 0 {
		*options = append(*options, "rand_seed="+strconv.Itoa(args.RandSeed))
	}
	if args.Morphology != "" {
		*options = append(*options, "morphology="+args.Morphology)
	}
	if args.TokenFilter != "" {
		*options = append(*options, "token_filter='"+strings.ReplaceAll(args.TokenFilter, "'", "''")+"'")
	}
	if args.MaxPredictedTime > 0 {
		*options = append(*options, "max_predicted_time="+strconv.Itoa(args.MaxPredictedTime))
	}
}

// addAgentSQLOptions adds distributed agent options to SQL
func (h *Handler) addAgentSQLOptions(options *[]string, args Args) {
	if args.AgentQueryTimeout > 0 {
		*options = append(*options, "agent_query_timeout="+strconv.Itoa(args.AgentQueryTimeout))
	}
	if args.RetryCount > 0 {
		*options = append(*options, "retry_count="+strconv.Itoa(args.RetryCount))
	}
	if args.RetryDelay > 0 {
		*options = append(*options, "retry_delay="+strconv.Itoa(args.RetryDelay))
	}
}

// addFuzzySQLOptions adds fuzzy search options to SQL
func (h *Handler) addFuzzySQLOptions(options *[]string, args Args) {
	if args.Fuzzy != nil && args.Fuzzy.Enabled {
		*options = append(*options, "fuzzy=1")
		if args.Fuzzy.Distance > 0 {
			*options = append(*options, "distance="+strconv.Itoa(args.Fuzzy.Distance))
		}
		if args.Fuzzy.Preserve > 0 {
			*options = append(*options, "preserve="+strconv.Itoa(args.Fuzzy.Preserve))
		}
		if len(args.Fuzzy.Layouts) > 0 {
			layouts := strings.Join(args.Fuzzy.Layouts, ",")
			*options = append(*options, "layouts='"+layouts+"'")
		}
	}
}

// buildHighlightFunction constructs HIGHLIGHT() function call
// Syntax: HIGHLIGHT([options], [field_list], [query])
func (h *Handler) buildHighlightFunction(highlight *HighlightOptions, query string) string {
	if !highlight.Enabled {
		return ""
	}

	var parts []string

	// Build options map (optional first parameter)
	var highlightOpts []string

	if highlight.Limit > 0 {
		highlightOpts = append(highlightOpts, "limit="+strconv.Itoa(highlight.Limit))
	}

	if highlight.LimitPerField > 0 {
		highlightOpts = append(highlightOpts, "limit_per_field="+strconv.Itoa(highlight.LimitPerField))
	}

	if highlight.LimitWords > 0 {
		highlightOpts = append(highlightOpts, "limit_words="+strconv.Itoa(highlight.LimitWords))
	}

	if highlight.Around > 0 {
		highlightOpts = append(highlightOpts, "around="+strconv.Itoa(highlight.Around))
	}

	if highlight.StartTag != "" {
		highlightOpts = append(highlightOpts, "before_match='"+strings.ReplaceAll(highlight.StartTag, "'", "''")+"'")
	}

	if highlight.EndTag != "" {
		highlightOpts = append(highlightOpts, "after_match='"+strings.ReplaceAll(highlight.EndTag, "'", "''")+"'")
	}

	// Add options map if any options specified
	if len(highlightOpts) > 0 {
		parts = append(parts, "{"+strings.Join(highlightOpts, ", ")+"}")
	}

	// Add field list (optional second parameter)
	if len(highlight.Fields) > 0 {
		fieldList := strings.Join(highlight.Fields, ",")
		parts = append(parts, "'"+fieldList+"'")
	}

	// Add query (optional third parameter)
	// If no custom query specified, highlight will use the MATCH query automatically
	// According to docs, query parameter is optional and defaults to search query

	return "HIGHLIGHT(" + strings.Join(parts, ", ") + ") AS highlight"
}

// buildTableName constructs table name with cluster prefix if provided
func (h *Handler) buildTableName(cluster, table string) string {
	if cluster != "" {
		return cluster + ":" + table
	}
	return table
}
