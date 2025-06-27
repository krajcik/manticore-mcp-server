package mcp

import (
	"testing"

	"manticore-mcp-server/config"
	"manticore-mcp-server/tools/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_mapToSearchArgs(t *testing.T) {
	cfg := &config.Config{
		MaxResultsPerQuery: 100,
	}
	registry := &Registry{config: cfg}

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected *search.Args
		wantErr  bool
		errMsg   string
	}{
		{
			name: "basic search args",
			args: map[string]interface{}{
				"query":   "test query",
				"table":   "articles",
				"cluster": "main",
				"limit":   10,
				"offset":  5,
			},
			expected: &search.Args{
				Query:   "test query",
				Table:   "articles",
				Cluster: "main",
				Limit:   10,
				Offset:  5,
			},
			wantErr: false,
		},
		{
			name: "missing table parameter",
			args: map[string]interface{}{
				"query": "test query",
			},
			expected: nil,
			wantErr:  true,
			errMsg:   "table parameter is required",
		},
		{
			name: "with field selection",
			args: map[string]interface{}{
				"table":  "products",
				"fields": []interface{}{"title", "description", "price"},
			},
			expected: &search.Args{
				Table:  "products",
				Fields: []string{"title", "description", "price"},
			},
			wantErr: false,
		},
		{
			name: "with search options",
			args: map[string]interface{}{
				"table":                "docs",
				"ranker":               "bm25",
				"match_mode":           "extended",
				"max_matches":          1000,
				"cutoff":               500,
				"max_query_time":       3000,
				"boolean_simplify":     1,
				"accurate_aggregation": 0,
			},
			expected: &search.Args{
				Table:               "docs",
				Ranker:              "bm25",
				MatchMode:           "extended",
				MaxMatches:          1000,
				Cutoff:              500,
				MaxQueryTime:        3000,
				BooleanSimplify:     1,
				AccurateAggregation: 0,
			},
			wantErr: false,
		},
		{
			name: "float64 to int conversion",
			args: map[string]interface{}{
				"table":       "test",
				"limit":       10.0,
				"offset":      5.5,
				"max_matches": 1000.0,
			},
			expected: &search.Args{
				Table:      "test",
				Limit:      10,
				Offset:     5,
				MaxMatches: 1000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.mapToSearchArgs(tt.args)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRegistry_mapToSearchArgs_ComplexOptions(t *testing.T) {
	cfg := &config.Config{
		MaxResultsPerQuery: 100,
	}
	registry := &Registry{config: cfg}

	tests := []struct {
		name     string
		args     map[string]interface{}
		expected *search.Args
		wantErr  bool
	}{
		{
			name: "with field weights",
			args: map[string]interface{}{
				"table": "articles",
				"field_weights": map[string]interface{}{
					"title":   10,
					"content": 5.0,
				},
			},
			expected: &search.Args{
				Table: "articles",
				FieldWeights: map[string]int{
					"title":   10,
					"content": 5,
				},
			},
			wantErr: false,
		},
		{
			name: "with highlighting options",
			args: map[string]interface{}{
				"table": "posts",
				"highlight": map[string]interface{}{
					"enabled":             true,
					"fields":              []interface{}{"title", "body"},
					"limit":               3,
					"around":              10,
					"start_tag":           "<mark>",
					"end_tag":             "</mark>",
					"number_of_fragments": 2,
				},
			},
			expected: &search.Args{
				Table: "posts",
				Highlight: &search.HighlightOptions{
					Enabled:         true,
					Fields:          []string{"title", "body"},
					Limit:           3,
					Around:          10,
					StartTag:        "<mark>",
					EndTag:          "</mark>",
					NumberFragments: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "with fuzzy options",
			args: map[string]interface{}{
				"table": "search_index",
				"fuzzy": map[string]interface{}{
					"enabled":  true,
					"distance": 2,
					"preserve": 1,
					"layouts":  []interface{}{"qwerty", "dvorak"},
				},
			},
			expected: &search.Args{
				Table: "search_index",
				Fuzzy: &search.FuzzyOptions{
					Enabled:  true,
					Distance: 2,
					Preserve: 1,
					Layouts:  []string{"qwerty", "dvorak"},
				},
			},
			wantErr: false,
		},
		{
			name: "with ordering and grouping",
			args: map[string]interface{}{
				"table":      "products",
				"order_by":   []interface{}{"weight() DESC", "id ASC"},
				"group_by":   []interface{}{"category_id"},
				"group_sort": "count(*) DESC",
			},
			expected: &search.Args{
				Table:     "products",
				OrderBy:   []string{"weight() DESC", "id ASC"},
				GroupBy:   []string{"category_id"},
				GroupSort: "count(*) DESC",
			},
			wantErr: false,
		},
		{
			name: "with where conditions",
			args: map[string]interface{}{
				"table": "items",
				"where": []interface{}{"price > 100", "category_id IN (1,2,3)"},
			},
			expected: &search.Args{
				Table: "items",
				Where: []string{"price > 100", "category_id IN (1,2,3)"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.mapToSearchArgs(tt.args)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
