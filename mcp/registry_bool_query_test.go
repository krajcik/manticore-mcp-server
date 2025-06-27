package mcp

import (
	"testing"

	"manticore-mcp-server/config"
	"manticore-mcp-server/tools/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_mapToSearchArgs_BoolQuery(t *testing.T) {
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
			name: "with boolean query",
			args: map[string]interface{}{
				"table": "documents",
				"bool_query": map[string]interface{}{
					"must": []interface{}{
						map[string]interface{}{
							"type": "match",
							"data": map[string]interface{}{
								"field": "content",
								"query": "important",
							},
						},
					},
					"should": []interface{}{
						map[string]interface{}{
							"type": "match",
							"data": map[string]interface{}{
								"field": "title",
								"query": "news",
							},
						},
					},
				},
			},
			expected: &search.Args{
				Table: "documents",
				BoolQuery: &search.BoolQuery{
					Must: []search.QueryClause{
						{
							Type: "match",
							Data: map[string]interface{}{
								"field": "content",
								"query": "important",
							},
						},
					},
					Should: []search.QueryClause{
						{
							Type: "match",
							Data: map[string]interface{}{
								"field": "title",
								"query": "news",
							},
						},
					},
				},
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

func TestRegistry_mapToBoolQuery(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		boolMap  map[string]interface{}
		expected *search.BoolQuery
		wantErr  bool
	}{
		{
			name: "complete bool query",
			boolMap: map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"type": "match",
						"data": map[string]interface{}{"field": "title", "query": "test"},
					},
				},
				"should": []interface{}{
					map[string]interface{}{
						"type": "range",
						"data": map[string]interface{}{"field": "price", "gte": 100},
					},
				},
				"must_not": []interface{}{
					map[string]interface{}{
						"type": "term",
						"data": map[string]interface{}{"field": "status", "value": "deleted"},
					},
				},
			},
			expected: &search.BoolQuery{
				Must: []search.QueryClause{
					{
						Type: "match",
						Data: map[string]interface{}{"field": "title", "query": "test"},
					},
				},
				Should: []search.QueryClause{
					{
						Type: "range",
						Data: map[string]interface{}{"field": "price", "gte": 100},
					},
				},
				MustNot: []search.QueryClause{
					{
						Type: "term",
						Data: map[string]interface{}{"field": "status", "value": "deleted"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "only must clause",
			boolMap: map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"type": "match",
						"data": "test data",
					},
				},
			},
			expected: &search.BoolQuery{
				Must: []search.QueryClause{
					{
						Type: "match",
						Data: "test data",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "empty bool query",
			boolMap:  map[string]interface{}{},
			expected: &search.BoolQuery{},
			wantErr:  false,
		},
		{
			name: "invalid must clause format",
			boolMap: map[string]interface{}{
				"must": "not a slice",
			},
			expected: &search.BoolQuery{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.mapToBoolQuery(tt.boolMap)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRegistry_mapToQueryClauses(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name        string
		clauseSlice []interface{}
		expected    []search.QueryClause
		wantErr     bool
	}{
		{
			name: "valid clauses",
			clauseSlice: []interface{}{
				map[string]interface{}{
					"type": "match",
					"data": map[string]interface{}{"field": "content", "query": "search"},
				},
				map[string]interface{}{
					"type": "range",
					"data": map[string]interface{}{"field": "date", "gte": "2023-01-01"},
				},
			},
			expected: []search.QueryClause{
				{
					Type: "match",
					Data: map[string]interface{}{"field": "content", "query": "search"},
				},
				{
					Type: "range",
					Data: map[string]interface{}{"field": "date", "gte": "2023-01-01"},
				},
			},
			wantErr: false,
		},
		{
			name: "clause without type",
			clauseSlice: []interface{}{
				map[string]interface{}{
					"data": "some data",
				},
			},
			expected: []search.QueryClause{
				{
					Type: "",
					Data: "some data",
				},
			},
			wantErr: false,
		},
		{
			name: "clause without data",
			clauseSlice: []interface{}{
				map[string]interface{}{
					"type": "match",
				},
			},
			expected: []search.QueryClause{
				{
					Type: "match",
					Data: nil,
				},
			},
			wantErr: false,
		},
		{
			name:        "empty slice",
			clauseSlice: []interface{}{},
			expected:    []search.QueryClause{},
			wantErr:     false,
		},
		{
			name: "invalid clause format",
			clauseSlice: []interface{}{
				"not a map",
				map[string]interface{}{
					"type": "valid",
					"data": "valid data",
				},
			},
			expected: []search.QueryClause{
				{
					Type: "valid",
					Data: "valid data",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.mapToQueryClauses(tt.clauseSlice)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
