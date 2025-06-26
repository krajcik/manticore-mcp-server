package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryBuilder_BuildHTTPQuery(t *testing.T) {
	qb := NewQueryBuilder("test_cluster", "test_table")

	t.Run("SimpleMatch", func(t *testing.T) {
		args := Args{
			Query: "test search",
			Limit: 10,
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)
		assert.Equal(t, "test_cluster:test_table", result["table"])
		assert.Equal(t, 10, result["limit"])

		query := result["query"].(map[string]interface{})
		match := query["match"].(map[string]interface{})
		assert.Equal(t, "test search", match["*"])
	})

	t.Run("WithClusterAndTable", func(t *testing.T) {
		qb2 := NewQueryBuilder("", "simple_table")
		args := Args{
			Query: "test",
			Limit: 5,
		}

		result, err := qb2.BuildHTTPQuery(args)
		assert.NoError(t, err)
		assert.Equal(t, "simple_table", result["table"])
	})

	t.Run("BoolQuery", func(t *testing.T) {
		boolQuery := BoolQuery{
			Must: []QueryClause{
				NewMatchClause("title", "test", "and"),
				NewRangeClause("price", map[string]interface{}{
					"gte": 100,
					"lte": 500,
				}),
			},
			Should: []QueryClause{
				NewEqualsClause("category", 1),
				NewEqualsClause("category", 2),
			},
			MustNot: []QueryClause{
				NewEqualsClause("status", "disabled"),
			},
		}

		args := Args{
			BoolQuery: &boolQuery,
			Limit:     20,
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)
		assert.Equal(t, 20, result["limit"])

		query := result["query"].(map[string]interface{})
		boolClause := query["bool"].(map[string]interface{})

		// Check must clauses
		must := boolClause["must"].([]map[string]interface{})
		assert.Len(t, must, 2)

		// Check should clauses
		should := boolClause["should"].([]map[string]interface{})
		assert.Len(t, should, 2)

		// Check must_not clauses
		mustNot := boolClause["must_not"].([]map[string]interface{})
		assert.Len(t, mustNot, 1)
	})

	t.Run("WithOptions", func(t *testing.T) {
		args := Args{
			Query:      "test",
			Ranker:     "bm25",
			MaxMatches: 1000,
			FieldWeights: map[string]int{
				"title":   10,
				"content": 5,
			},
			Comment: "test comment",
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)

		options := result["options"].(map[string]interface{})
		assert.Equal(t, "bm25", options["ranker"])
		assert.Equal(t, 1000, options["max_matches"])
		assert.Equal(t, "test comment", options["comment"])

		fieldWeights := options["field_weights"].(map[string]int)
		assert.Equal(t, 10, fieldWeights["title"])
		assert.Equal(t, 5, fieldWeights["content"])
	})

	t.Run("WithHighlight", func(t *testing.T) {
		args := Args{
			Query: "test",
			Highlight: &HighlightOptions{
				Enabled:         true,
				Fields:          []string{"title", "content"},
				Limit:           50,
				StartTag:        "<mark>",
				EndTag:          "</mark>",
				NumberFragments: 3,
			},
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)

		highlight := result["highlight"].(map[string]interface{})
		assert.Equal(t, 50, highlight["limit"])
		assert.Equal(t, "<mark>", highlight["before_match"])
		assert.Equal(t, "</mark>", highlight["after_match"])
		assert.Equal(t, 3, highlight["number_of_fragments"])

		fields := highlight["fields"].([]string)
		assert.Contains(t, fields, "title")
		assert.Contains(t, fields, "content")
	})

	t.Run("WithSorting", func(t *testing.T) {
		args := Args{
			Query:   "test",
			OrderBy: []string{"price DESC", "title ASC", "weight()"},
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)

		sort := result["sort"].([]map[string]string)
		assert.Len(t, sort, 3)
		assert.Equal(t, "desc", sort[0]["price"])
		assert.Equal(t, "asc", sort[1]["title"])
		assert.Equal(t, "asc", sort[2]["weight()"])
	})

	t.Run("WithSourceFields", func(t *testing.T) {
		args := Args{
			Query:  "test",
			Fields: []string{"id", "title", "price"},
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)

		source := result["_source"].([]string)
		assert.Contains(t, source, "id")
		assert.Contains(t, source, "title")
		assert.Contains(t, source, "price")
	})

	t.Run("WithPagination", func(t *testing.T) {
		args := Args{
			Query:  "test",
			Limit:  25,
			Offset: 50,
		}

		result, err := qb.BuildHTTPQuery(args)
		assert.NoError(t, err)
		assert.Equal(t, 25, result["limit"])
		assert.Equal(t, 50, result["offset"])
	})
}

func TestQueryBuilder_BuildQueryClauses(t *testing.T) {
	qb := NewQueryBuilder("", "test_table")

	t.Run("MatchClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewMatchClause("title", "test query", "and"),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		match := result[0]["match"].(map[string]interface{})
		title := match["title"].(map[string]interface{})
		assert.Equal(t, "test query", title["query"])
		assert.Equal(t, "and", title["operator"])
	})

	t.Run("RangeClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewRangeClause("price", map[string]interface{}{
				"gte": 100,
				"lt":  1000,
			}),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		rangeClause := result[0]["range"].(map[string]interface{})
		price := rangeClause["price"].(map[string]interface{})
		assert.Equal(t, 100, price["gte"])
		assert.Equal(t, 1000, price["lt"])
	})

	t.Run("EqualsClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewEqualsClause("category", 1),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		equals := result[0]["equals"].(map[string]interface{})
		assert.Equal(t, 1, equals["category"])
	})

	t.Run("InClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewInClause("category", []interface{}{1, 2, 3}),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		in := result[0]["in"].(map[string]interface{})
		values := in["category"].([]interface{})
		assert.Contains(t, values, 1)
		assert.Contains(t, values, 2)
		assert.Contains(t, values, 3)
	})

	t.Run("GeoDistanceClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewGeoDistanceClause("adaptive", map[string]float64{
				"lat": 52.396,
				"lon": -1.774,
			}, "latitude,longitude", "10000 m"),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		geoDist := result[0]["geo_distance"].(map[string]interface{})
		assert.Equal(t, "adaptive", geoDist["distance_type"])
		assert.Equal(t, "10000 m", geoDist["distance"])

		anchor := geoDist["location_anchor"].(map[string]float64)
		assert.InDelta(t, 52.396, anchor["lat"], 0.001)
		assert.InDelta(t, -1.774, anchor["lon"], 0.001)
	})

	t.Run("QueryStringClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewQueryStringClause("@title hello @body world"),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		queryString := result[0]["query_string"].(string)
		assert.Equal(t, "@title hello @body world", queryString)
	})

	t.Run("MatchAllClause", func(t *testing.T) {
		clauses := []QueryClause{
			NewMatchAllClause(),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		matchAll := result[0]["match_all"].(map[string]interface{})
		assert.Empty(t, matchAll)
	})

	t.Run("NestedBoolClause", func(t *testing.T) {
		nestedBool := BoolQuery{
			Must: []QueryClause{
				NewMatchClause("title", "test", ""),
			},
		}

		clauses := []QueryClause{
			NewBoolClause(nestedBool),
		}

		result, err := qb.buildQueryClauses(clauses)
		assert.NoError(t, err)
		assert.Len(t, result, 1)

		boolQuery := result[0]["bool"].(map[string]interface{})
		must := boolQuery["must"].([]map[string]interface{})
		assert.Len(t, must, 1)
	})

	t.Run("InvalidClauseType", func(t *testing.T) {
		clauses := []QueryClause{
			{
				Type: "invalid_type",
				Data: "invalid_data",
			},
		}

		_, err := qb.buildQueryClauses(clauses)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported query clause type")
	})

	t.Run("InvalidClauseData", func(t *testing.T) {
		clauses := []QueryClause{
			{
				Type: "match",
				Data: "invalid_match_data", // Should be MatchClause
			},
		}

		_, err := qb.buildQueryClauses(clauses)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid match clause data")
	})
}

func TestQueryBuilder_MatchClauseVariations(t *testing.T) {
	qb := NewQueryBuilder("", "test_table")

	t.Run("MatchWithoutOperator", func(t *testing.T) {
		clause := NewMatchClause("title", "test query", "")
		result := qb.buildMatchClause(clause.Data.(MatchClause))

		match := result["match"].(map[string]interface{})
		assert.Equal(t, "test query", match["title"])
	})

	t.Run("MatchWithOperator", func(t *testing.T) {
		clause := NewMatchClause("title", "test query", "or")
		result := qb.buildMatchClause(clause.Data.(MatchClause))

		match := result["match"].(map[string]interface{})
		title := match["title"].(map[string]interface{})
		assert.Equal(t, "test query", title["query"])
		assert.Equal(t, "or", title["operator"])
	})
}

func TestQueryBuilder_BuildTableName(t *testing.T) {
	t.Run("WithCluster", func(t *testing.T) {
		qb := NewQueryBuilder("test_cluster", "test_table")
		tableName := qb.buildTableName()
		assert.Equal(t, "test_cluster:test_table", tableName)
	})

	t.Run("WithoutCluster", func(t *testing.T) {
		qb := NewQueryBuilder("", "test_table")
		tableName := qb.buildTableName()
		assert.Equal(t, "test_table", tableName)
	})
}

func TestQueryBuilder_BuildHTTPOptions(t *testing.T) {
	qb := NewQueryBuilder("", "test_table")

	t.Run("AllOptions", func(t *testing.T) {
		args := Args{
			Ranker:              "bm25",
			MaxMatches:          2000,
			Cutoff:              1000,
			MaxQueryTime:        5000,
			NotTermsOnlyAllowed: 1,
			BooleanSimplify:     0,
			AccurateAggregation: 1,
			RandSeed:            12345,
			Comment:             "test query",
			AgentQueryTimeout:   10000,
			RetryCount:          3,
			RetryDelay:          1000,
			Morphology:          "none",
			TokenFilter:         "mylib.so:plugin:settings",
			MaxPredictedTime:    3000,
			FieldWeights: map[string]int{
				"title":   10,
				"content": 5,
			},
		}

		options := qb.buildHTTPOptions(args)

		assert.Equal(t, "bm25", options["ranker"])
		assert.Equal(t, 2000, options["max_matches"])
		assert.Equal(t, 1000, options["cutoff"])
		assert.Equal(t, 5000, options["max_query_time"])
		assert.Equal(t, 1, options["not_terms_only_allowed"])
		assert.Equal(t, 0, options["boolean_simplify"])
		assert.Equal(t, 1, options["accurate_aggregation"])
		assert.Equal(t, 12345, options["rand_seed"])
		assert.Equal(t, "test query", options["comment"])
		assert.Equal(t, 10000, options["agent_query_timeout"])
		assert.Equal(t, 3, options["retry_count"])
		assert.Equal(t, 1000, options["retry_delay"])
		assert.Equal(t, "none", options["morphology"])
		assert.Equal(t, "mylib.so:plugin:settings", options["token_filter"])
		assert.Equal(t, 3000, options["max_predicted_time"])

		fieldWeights := options["field_weights"].(map[string]int)
		assert.Equal(t, 10, fieldWeights["title"])
		assert.Equal(t, 5, fieldWeights["content"])
	})

	t.Run("NoOptions", func(t *testing.T) {
		args := Args{
			BooleanSimplify: 1, // Default value
		}
		options := qb.buildHTTPOptions(args)
		assert.Empty(t, options)
	})

	t.Run("DefaultBooleanSimplify", func(t *testing.T) {
		args := Args{
			BooleanSimplify: 1, // Default value, should not be included
		}
		options := qb.buildHTTPOptions(args)
		assert.NotContains(t, options, "boolean_simplify")
	})
}

func TestQueryBuilder_BuildHighlightOptions(t *testing.T) {
	qb := NewQueryBuilder("", "test_table")

	t.Run("AllHighlightOptions", func(t *testing.T) {
		highlight := HighlightOptions{
			Fields:          []string{"title", "content"},
			Limit:           50,
			LimitPerField:   10,
			LimitWords:      100,
			Around:          3,
			StartTag:        "<mark>",
			EndTag:          "</mark>",
			NumberFragments: 5,
		}

		result := qb.buildHighlightOptions(highlight)

		assert.Equal(t, 50, result["limit"])
		assert.Equal(t, 10, result["limit_per_field"])
		assert.Equal(t, 100, result["limit_words"])
		assert.Equal(t, 3, result["around"])
		assert.Equal(t, "<mark>", result["before_match"])
		assert.Equal(t, "</mark>", result["after_match"])
		assert.Equal(t, 5, result["number_of_fragments"])

		fields := result["fields"].([]string)
		assert.Contains(t, fields, "title")
		assert.Contains(t, fields, "content")
	})

	t.Run("MinimalHighlightOptions", func(t *testing.T) {
		highlight := HighlightOptions{}
		result := qb.buildHighlightOptions(highlight)
		assert.Empty(t, result)
	})
}
