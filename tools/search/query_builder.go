package search

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidMatchClauseData       = errors.New("invalid match clause data")
	ErrInvalidRangeClauseData       = errors.New("invalid range clause data")
	ErrInvalidEqualsClauseData      = errors.New("invalid equals clause data")
	ErrInvalidInClauseData          = errors.New("invalid in clause data")
	ErrInvalidGeoDistanceClauseData = errors.New("invalid geo_distance clause data")
	ErrInvalidQueryStringClauseData = errors.New("invalid query_string clause data")
	ErrInvalidBoolClauseData        = errors.New("invalid bool clause data")
	ErrUnsupportedClauseType        = errors.New("unsupported query clause type")
)

// QueryBuilder helps construct complex search queries
type QueryBuilder struct {
	cluster string
	table   string
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(cluster, table string) *QueryBuilder {
	return &QueryBuilder{
		cluster: cluster,
		table:   table,
	}
}

// BoolQuery represents a boolean query with must, should, must_not clauses
type BoolQuery struct {
	Must    []QueryClause `json:"must,omitempty"`
	Should  []QueryClause `json:"should,omitempty"`
	MustNot []QueryClause `json:"must_not,omitempty"`
}

// QueryClause represents different types of query clauses
type QueryClause struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// MatchClause represents a match query
type MatchClause struct {
	Field    string `json:"field"`
	Query    string `json:"query"`
	Operator string `json:"operator,omitempty"` // "and" or "or"
}

// RangeClause represents a range query
type RangeClause struct {
	Field  string                 `json:"field"`
	Ranges map[string]interface{} `json:"ranges"` // gte, lte, gt, lt
}

// EqualsClause represents an equals query
type EqualsClause struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// InClause represents an IN query
type InClause struct {
	Field  string        `json:"field"`
	Values []interface{} `json:"values"`
}

// GeoDistanceClause represents a geo distance query
type GeoDistanceClause struct {
	DistanceType   string             `json:"distance_type"`
	LocationAnchor map[string]float64 `json:"location_anchor"`
	LocationSource string             `json:"location_source"`
	Distance       string             `json:"distance"`
}

// QueryStringClause represents a query_string query
type QueryStringClause struct {
	Query string `json:"query"`
}

// MatchAllClause represents a match_all query
type MatchAllClause struct{}

// BuildHTTPQuery constructs HTTP JSON query from complex search arguments
func (qb *QueryBuilder) BuildHTTPQuery(args Args) (map[string]interface{}, error) {
	query := make(map[string]interface{})
	query["table"] = qb.buildTableName()

	// Build main query
	if args.BoolQuery != nil {
		mainQuery, err := qb.buildBoolQuery(*args.BoolQuery)
		if err != nil {
			return nil, err
		}
		query["query"] = mainQuery
	} else if args.Query != "" {
		// Simple match query
		query["query"] = map[string]interface{}{
			"match": map[string]interface{}{
				"*": args.Query,
			},
		}
	} else {
		// Match all
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Add pagination
	if args.Limit > 0 {
		query["limit"] = args.Limit
	}
	if args.Offset > 0 {
		query["offset"] = args.Offset
	}

	// Add source fields
	if len(args.Fields) > 0 {
		query["_source"] = args.Fields
	}

	// Add sorting
	if len(args.OrderBy) > 0 {
		sort := make([]map[string]string, len(args.OrderBy))
		for i, orderExpr := range args.OrderBy {
			parts := strings.Fields(orderExpr)
			if len(parts) >= 2 {
				field := parts[0]
				direction := strings.ToLower(parts[1])
				sort[i] = map[string]string{field: direction}
			} else {
				sort[i] = map[string]string{orderExpr: "asc"}
			}
		}
		query["sort"] = sort
	}

	// Add highlighting
	if args.Highlight != nil && args.Highlight.Enabled {
		highlight := qb.buildHighlightOptions(*args.Highlight)
		query["highlight"] = highlight
	}

	// Add options
	if options := qb.buildHTTPOptions(args); len(options) > 0 {
		query["options"] = options
	}

	return query, nil
}

// buildBoolQuery constructs a boolean query
func (qb *QueryBuilder) buildBoolQuery(boolQuery BoolQuery) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	boolClause := make(map[string]interface{})

	if len(boolQuery.Must) > 0 {
		must, err := qb.buildQueryClauses(boolQuery.Must)
		if err != nil {
			return nil, err
		}
		boolClause["must"] = must
	}

	if len(boolQuery.Should) > 0 {
		should, err := qb.buildQueryClauses(boolQuery.Should)
		if err != nil {
			return nil, err
		}
		boolClause["should"] = should
	}

	if len(boolQuery.MustNot) > 0 {
		mustNot, err := qb.buildQueryClauses(boolQuery.MustNot)
		if err != nil {
			return nil, err
		}
		boolClause["must_not"] = mustNot
	}

	result["bool"] = boolClause
	return result, nil
}

// buildQueryClauses constructs individual query clauses
func (qb *QueryBuilder) buildQueryClauses(clauses []QueryClause) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, len(clauses))

	for i, clause := range clauses {
		clauseData, err := qb.buildSingleClause(clause)
		if err != nil {
			return nil, err
		}
		result[i] = clauseData
	}

	return result, nil
}

// buildSingleClause constructs a single query clause
func (qb *QueryBuilder) buildSingleClause(clause QueryClause) (map[string]interface{}, error) {
	switch clause.Type {
	case "match":
		return qb.buildMatchClauseFromData(clause.Data)
	case "range":
		return qb.buildRangeClauseFromData(clause.Data)
	case "equals":
		return qb.buildEqualsClauseFromData(clause.Data)
	case "in":
		return qb.buildInClauseFromData(clause.Data)
	case "geo_distance":
		return qb.buildGeoDistanceClauseFromData(clause.Data)
	case "query_string":
		return qb.buildQueryStringClauseFromData(clause.Data)
	case "match_all":
		return qb.buildMatchAllClause()
	case "bool":
		return qb.buildBoolClauseFromData(clause.Data)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedClauseType, clause.Type)
	}
}

// buildMatchClauseFromData constructs a match clause from interface data
func (qb *QueryBuilder) buildMatchClauseFromData(data interface{}) (map[string]interface{}, error) {
	matchData, ok := data.(MatchClause)
	if !ok {
		return nil, ErrInvalidMatchClauseData
	}
	return qb.buildMatchClause(matchData), nil
}

// buildRangeClauseFromData constructs a range clause from interface data
func (qb *QueryBuilder) buildRangeClauseFromData(data interface{}) (map[string]interface{}, error) {
	rangeData, ok := data.(RangeClause)
	if !ok {
		return nil, ErrInvalidRangeClauseData
	}
	return qb.buildRangeClause(rangeData), nil
}

// buildEqualsClauseFromData constructs an equals clause from interface data
func (qb *QueryBuilder) buildEqualsClauseFromData(data interface{}) (map[string]interface{}, error) {
	equalsData, ok := data.(EqualsClause)
	if !ok {
		return nil, ErrInvalidEqualsClauseData
	}
	return qb.buildEqualsClause(equalsData), nil
}

// buildInClauseFromData constructs an in clause from interface data
func (qb *QueryBuilder) buildInClauseFromData(data interface{}) (map[string]interface{}, error) {
	inData, ok := data.(InClause)
	if !ok {
		return nil, ErrInvalidInClauseData
	}
	return qb.buildInClause(inData), nil
}

// buildGeoDistanceClauseFromData constructs a geo_distance clause from interface data
func (qb *QueryBuilder) buildGeoDistanceClauseFromData(data interface{}) (map[string]interface{}, error) {
	geoData, ok := data.(GeoDistanceClause)
	if !ok {
		return nil, ErrInvalidGeoDistanceClauseData
	}
	return qb.buildGeoDistanceClause(geoData), nil
}

// buildQueryStringClauseFromData constructs a query_string clause from interface data
func (qb *QueryBuilder) buildQueryStringClauseFromData(data interface{}) (map[string]interface{}, error) {
	queryStringData, ok := data.(QueryStringClause)
	if !ok {
		return nil, ErrInvalidQueryStringClauseData
	}
	return qb.buildQueryStringClause(queryStringData), nil
}

// buildMatchAllClause constructs a match_all clause
func (qb *QueryBuilder) buildMatchAllClause() (map[string]interface{}, error) {
	return map[string]interface{}{
		"match_all": map[string]interface{}{},
	}, nil
}

// buildBoolClauseFromData constructs a bool clause from interface data
func (qb *QueryBuilder) buildBoolClauseFromData(data interface{}) (map[string]interface{}, error) {
	boolData, ok := data.(BoolQuery)
	if !ok {
		return nil, ErrInvalidBoolClauseData
	}
	return qb.buildBoolQuery(boolData)
}

// buildMatchClause constructs a match clause
func (qb *QueryBuilder) buildMatchClause(match MatchClause) map[string]interface{} {
	if match.Operator != "" {
		return map[string]interface{}{
			"match": map[string]interface{}{
				match.Field: map[string]interface{}{
					"query":    match.Query,
					"operator": match.Operator,
				},
			},
		}
	}
	return map[string]interface{}{
		"match": map[string]interface{}{
			match.Field: match.Query,
		},
	}
}

// buildRangeClause constructs a range clause
func (qb *QueryBuilder) buildRangeClause(rangeClause RangeClause) map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			rangeClause.Field: rangeClause.Ranges,
		},
	}
}

// buildEqualsClause constructs an equals clause
func (qb *QueryBuilder) buildEqualsClause(equals EqualsClause) map[string]interface{} {
	return map[string]interface{}{
		"equals": map[string]interface{}{
			equals.Field: equals.Value,
		},
	}
}

// buildInClause constructs an in clause
func (qb *QueryBuilder) buildInClause(in InClause) map[string]interface{} {
	return map[string]interface{}{
		"in": map[string]interface{}{
			in.Field: in.Values,
		},
	}
}

// buildGeoDistanceClause constructs a geo_distance clause
func (qb *QueryBuilder) buildGeoDistanceClause(geo GeoDistanceClause) map[string]interface{} {
	return map[string]interface{}{
		"geo_distance": map[string]interface{}{
			"distance_type":   geo.DistanceType,
			"location_anchor": geo.LocationAnchor,
			"location_source": geo.LocationSource,
			"distance":        geo.Distance,
		},
	}
}

// buildQueryStringClause constructs a query_string clause
func (qb *QueryBuilder) buildQueryStringClause(qs QueryStringClause) map[string]interface{} {
	return map[string]interface{}{
		"query_string": qs.Query,
	}
}

// buildHighlightOptions constructs highlight options
func (qb *QueryBuilder) buildHighlightOptions(highlight HighlightOptions) map[string]interface{} {
	result := make(map[string]interface{})

	if len(highlight.Fields) > 0 {
		result["fields"] = highlight.Fields
	}

	if highlight.Limit > 0 {
		result["limit"] = highlight.Limit
	}

	if highlight.LimitPerField > 0 {
		result["limit_per_field"] = highlight.LimitPerField
	}

	if highlight.LimitWords > 0 {
		result["limit_words"] = highlight.LimitWords
	}

	if highlight.Around > 0 {
		result["around"] = highlight.Around
	}

	if highlight.StartTag != "" {
		result["before_match"] = highlight.StartTag
	}

	if highlight.EndTag != "" {
		result["after_match"] = highlight.EndTag
	}

	if highlight.NumberFragments > 0 {
		result["number_of_fragments"] = highlight.NumberFragments
	}

	return result
}

// buildHTTPOptions constructs options for HTTP query
func (qb *QueryBuilder) buildHTTPOptions(args Args) map[string]interface{} {
	options := make(map[string]interface{})

	qb.addBasicOptions(options, args)
	qb.addAdvancedOptions(options, args)
	qb.addAgentOptions(options, args)

	return options
}

// addBasicOptions adds basic search options
func (qb *QueryBuilder) addBasicOptions(options map[string]interface{}, args Args) {
	if args.Ranker != "" {
		options["ranker"] = args.Ranker
	}
	if args.MaxMatches > 0 {
		options["max_matches"] = args.MaxMatches
	}
	if args.Cutoff > 0 {
		options["cutoff"] = args.Cutoff
	}
	if args.MaxQueryTime > 0 {
		options["max_query_time"] = args.MaxQueryTime
	}
	if len(args.FieldWeights) > 0 {
		options["field_weights"] = args.FieldWeights
	}
	if args.Comment != "" {
		options["comment"] = args.Comment
	}
}

// addAdvancedOptions adds advanced search options
func (qb *QueryBuilder) addAdvancedOptions(options map[string]interface{}, args Args) {
	if args.NotTermsOnlyAllowed > 0 {
		options["not_terms_only_allowed"] = args.NotTermsOnlyAllowed
	}
	if args.BooleanSimplify == 0 {
		options["boolean_simplify"] = 0
	}
	if args.AccurateAggregation > 0 {
		options["accurate_aggregation"] = args.AccurateAggregation
	}
	if args.RandSeed > 0 {
		options["rand_seed"] = args.RandSeed
	}
	if args.Morphology != "" {
		options["morphology"] = args.Morphology
	}
	if args.TokenFilter != "" {
		options["token_filter"] = args.TokenFilter
	}
	if args.MaxPredictedTime > 0 {
		options["max_predicted_time"] = args.MaxPredictedTime
	}
}

// addAgentOptions adds distributed agent options
func (qb *QueryBuilder) addAgentOptions(options map[string]interface{}, args Args) {
	if args.AgentQueryTimeout > 0 {
		options["agent_query_timeout"] = args.AgentQueryTimeout
	}
	if args.RetryCount > 0 {
		options["retry_count"] = args.RetryCount
	}
	if args.RetryDelay > 0 {
		options["retry_delay"] = args.RetryDelay
	}
}

// buildTableName constructs table name with cluster prefix if provided
func (qb *QueryBuilder) buildTableName() string {
	if qb.cluster != "" {
		return qb.cluster + ":" + qb.table
	}
	return qb.table
}

// Helper functions for creating query clauses

// NewMatchClause creates a new match clause
func NewMatchClause(field, query, operator string) QueryClause {
	return QueryClause{
		Type: "match",
		Data: MatchClause{
			Field:    field,
			Query:    query,
			Operator: operator,
		},
	}
}

// NewRangeClause creates a new range clause
func NewRangeClause(field string, ranges map[string]interface{}) QueryClause {
	return QueryClause{
		Type: "range",
		Data: RangeClause{
			Field:  field,
			Ranges: ranges,
		},
	}
}

// NewEqualsClause creates a new equals clause
func NewEqualsClause(field string, value interface{}) QueryClause {
	return QueryClause{
		Type: "equals",
		Data: EqualsClause{
			Field: field,
			Value: value,
		},
	}
}

// NewInClause creates a new in clause
func NewInClause(field string, values []interface{}) QueryClause {
	return QueryClause{
		Type: "in",
		Data: InClause{
			Field:  field,
			Values: values,
		},
	}
}

// NewGeoDistanceClause creates a new geo_distance clause
func NewGeoDistanceClause(distanceType string, anchor map[string]float64, source, distance string) QueryClause {
	return QueryClause{
		Type: "geo_distance",
		Data: GeoDistanceClause{
			DistanceType:   distanceType,
			LocationAnchor: anchor,
			LocationSource: source,
			Distance:       distance,
		},
	}
}

// NewQueryStringClause creates a new query_string clause
func NewQueryStringClause(query string) QueryClause {
	return QueryClause{
		Type: "query_string",
		Data: QueryStringClause{
			Query: query,
		},
	}
}

// NewMatchAllClause creates a new match_all clause
func NewMatchAllClause() QueryClause {
	return QueryClause{
		Type: "match_all",
		Data: MatchAllClause{},
	}
}

// NewBoolClause creates a new bool clause
func NewBoolClause(boolQuery BoolQuery) QueryClause {
	return QueryClause{
		Type: "bool",
		Data: boolQuery,
	}
}
