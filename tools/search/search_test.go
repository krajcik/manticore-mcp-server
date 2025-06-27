package search

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"manticore-mcp-server/client"
	"manticore-mcp-server/config"
	"manticore-mcp-server/testutils"
)

type SearchTestSuite struct {
	suite.Suite
	handler *Handler
	client  client.ManticoreClient
	cfg     *config.Config
}

func (s *SearchTestSuite) SetupSuite() {
	s.cfg = testutils.LoadTestConfig()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	s.client = client.New(s.cfg, logger)
	s.handler = NewHandler(s.client, logger)

	// Wait for Manticore to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 30; i++ {
		if err := s.client.Ping(ctx); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		if i == 29 {
			s.T().Fatal("Failed to connect to Manticore after 30 seconds")
		}
	}

	// Create test table
	s.createTestTable()
}

func (s *SearchTestSuite) TearDownSuite() {
	// Clean up test table
	ctx := context.Background()
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_search_table")
}

func (s *SearchTestSuite) createTestTable() {
	s.T().Helper()

	ctx := context.Background()

	// Drop if exists
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_search_table")

	// Create table
	createSQL := `CREATE TABLE test_search_table (
		id bigint,
		title text,
		content text,
		price float,
		category int
	)`
	_, err := s.client.ExecuteSQL(ctx, createSQL)
	s.Require().NoError(err)

	// Insert test data
	testData := []string{
		"INSERT INTO test_search_table (id, title, content, price, category) VALUES (1, 'laptop computer', 'high performance gaming laptop', 1500.99, 1)",
		"INSERT INTO test_search_table (id, title, content, price, category) VALUES (2, 'smartphone mobile', 'latest android smartphone device', 899.50, 2)",
		"INSERT INTO test_search_table (id, title, content, price, category) VALUES (3, 'tablet device', 'portable tablet for reading books', 299.99, 2)",
		"INSERT INTO test_search_table (id, title, content, price, category) VALUES (4, 'desktop computer', 'powerful desktop workstation', 2500.00, 1)",
		"INSERT INTO test_search_table (id, title, content, price, category) VALUES (5, 'gaming laptop', 'professional gaming laptop computer', 1899.99, 1)",
	}

	for _, insertSQL := range testData {
		_, err := s.client.ExecuteSQL(ctx, insertSQL)
		s.Require().NoError(err)
	}
}

func (s *SearchTestSuite) TestSimpleSearch() {
	ctx := context.Background()

	args := Args{
		Query: "laptop",
		Table: "test_search_table",
		Limit: 10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.NotEmpty(result)

	// Should find at least 2 laptops
	s.GreaterOrEqual(len(result), 2)

	// Check that results contain laptop-related content
	foundLaptop := false
	for _, row := range result {
		if title, ok := row["title"].(string); ok {
			if title == "laptop computer" || title == "gaming laptop" {
				foundLaptop = true
				break
			}
		}
	}
	s.True(foundLaptop, "Should find laptop in results")
}

func (s *SearchTestSuite) TestSearchWithFieldWeights() {
	ctx := context.Background()

	args := Args{
		Query: "computer",
		Table: "test_search_table",
		FieldWeights: map[string]int{
			"title":   10,
			"content": 3,
		},
		Ranker: "bm25",
		Limit:  5,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.NotEmpty(result)
}

func (s *SearchTestSuite) TestSearchWithHighlighting() {
	ctx := context.Background()

	args := Args{
		Query: "gaming",
		Table: "test_search_table",
		Highlight: &HighlightOptions{
			Enabled:         true,
			Fields:          []string{"title", "content"},
			StartTag:        "<mark>",
			EndTag:          "</mark>",
			Around:          3,
			NumberFragments: 2,
		},
		Limit: 10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.NotEmpty(result)

	// Check for highlight in results
	foundHighlight := false
	for _, row := range result {
		if highlight, ok := row["highlight"].(string); ok && highlight != "" {
			foundHighlight = true
			s.Contains(highlight, "<mark>")
			s.Contains(highlight, "</mark>")
			break
		}
	}
	s.True(foundHighlight, "Should find highlighted results")
}

func (s *SearchTestSuite) TestSearchWithFuzzy() {
	ctx := context.Background()

	args := Args{
		Query: "compueter", // Intentional typo
		Table: "test_search_table",
		Fuzzy: &FuzzyOptions{
			Enabled:  true,
			Distance: 2,
		},
		Limit: 5,
	}

	_, err := s.handler.Execute(ctx, args)
	// Fuzzy search requires min_infix_len setting, which we don't have in test table
	// So we expect this to fail
	s.Error(err, "Fuzzy search should fail without min_infix_len setting")
	s.Contains(err.Error(), "min_infix_len")
}

func (s *SearchTestSuite) TestSearchWithFuzzyPositive() {
	ctx := context.Background()

	// Create table with fuzzy search support
	createFuzzyTableSQL := `CREATE TABLE test_fuzzy_table (
		id bigint,
		title text,
		content text
	) min_infix_len='2'`

	_, err := s.client.ExecuteSQL(ctx, createFuzzyTableSQL)
	s.Require().NoError(err)

	// Insert test data with intentional variations
	testData := []string{
		"INSERT INTO test_fuzzy_table (id, title, content) VALUES (1, 'computer laptop', 'high performance computing device')",
		"INSERT INTO test_fuzzy_table (id, title, content) VALUES (2, 'smartphone mobile', 'mobile communication device')",
		"INSERT INTO test_fuzzy_table (id, title, content) VALUES (3, 'tablet device', 'portable computing tablet')",
	}

	for _, insertSQL := range testData {
		_, err := s.client.ExecuteSQL(ctx, insertSQL)
		s.Require().NoError(err)
	}

	// Test fuzzy search with typo
	args := Args{
		Query: "compueter", // Intentional typo for "computer"
		Table: "test_fuzzy_table",
		Fuzzy: &FuzzyOptions{
			Enabled:  true,
			Distance: 2,
		},
		Limit: 10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err, "Fuzzy search should work with min_infix_len setting")
	s.NotEmpty(result, "Should find fuzzy matches")

	// Check that we found the computer-related document
	foundComputer := false
	for _, row := range result {
		if title, ok := row["title"].(string); ok {
			if title == "computer laptop" {
				foundComputer = true
				break
			}
		}
	}
	s.True(foundComputer, "Should find 'computer laptop' with fuzzy search for 'compueter'")

	// Clean up
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_fuzzy_table")
}

func (s *SearchTestSuite) TestSearchWithOrdering() {
	ctx := context.Background()

	args := Args{
		Query:   "laptop computer",
		Table:   "test_search_table",
		OrderBy: []string{"price DESC", "id ASC"},
		Limit:   10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.NotEmpty(result)

	// Check ordering by price DESC
	if len(result) >= 2 {
		for i := 0; i < len(result)-1; i++ {
			price1, ok1 := result[i]["price"].(float64)
			price2, ok2 := result[i+1]["price"].(float64)
			if ok1 && ok2 {
				s.GreaterOrEqual(price1, price2, "Results should be ordered by price DESC")
			}
		}
	}
}

func (s *SearchTestSuite) TestSearchWithFilters() {
	ctx := context.Background()

	args := Args{
		Query: "laptop",
		Table: "test_search_table",
		Where: []string{"price > 1000", "category = 1"},
		Limit: 10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)

	// All results should meet the filter criteria
	for _, row := range result {
		if price, ok := row["price"].(float64); ok {
			s.Greater(price, 1000.0, "Price should be greater than 1000")
		}
		if category, ok := row["category"].(float64); ok { // JSON numbers come as float64
			s.InDelta(1.0, category, 0.01, "Category should be 1")
		}
	}
}

func (s *SearchTestSuite) TestSearchWithPagination() {
	ctx := context.Background()

	// First page
	args1 := Args{
		Query:  "laptop computer",
		Table:  "test_search_table",
		Limit:  2,
		Offset: 0,
	}

	result1, err := s.handler.Execute(ctx, args1)
	s.NoError(err)
	s.LessOrEqual(len(result1), 2)

	// Second page
	args2 := Args{
		Query:  "laptop computer",
		Table:  "test_search_table",
		Limit:  2,
		Offset: 2,
	}

	result2, err := s.handler.Execute(ctx, args2)
	s.NoError(err)

	// Results should be different between pages
	if len(result1) > 0 && len(result2) > 0 {
		id1, _ := result1[0]["id"].(float64)
		id2, _ := result2[0]["id"].(float64)
		s.NotEqual(id1, id2, "Different pages should have different results")
	}
}

func (s *SearchTestSuite) TestSearchWithMaxMatches() {
	ctx := context.Background()

	args := Args{
		Query:      "laptop computer device",
		Table:      "test_search_table",
		MaxMatches: 2,
		Limit:      10,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.LessOrEqual(len(result), 2, "Should respect max_matches limit")
}

func (s *SearchTestSuite) TestSearchWithComment() {
	ctx := context.Background()

	args := Args{
		Query:   "laptop",
		Table:   "test_search_table",
		Comment: "test search with comment",
		Limit:   5,
	}

	result, err := s.handler.Execute(ctx, args)
	s.NoError(err)
	s.NotNil(result)
}

func (s *SearchTestSuite) TestSearchWithRanker() {
	ctx := context.Background()

	rankers := []string{"bm25", "proximity_bm25", "wordcount", "proximity"}

	for _, ranker := range rankers {
		args := Args{
			Query:  "laptop computer",
			Table:  "test_search_table",
			Ranker: ranker,
			Limit:  5,
		}

		result, err := s.handler.Execute(ctx, args)
		s.NoError(err, "Ranker %s should work", ranker)
		s.NotNil(result)
	}
}

func (s *SearchTestSuite) TestSearchErrors() {
	ctx := context.Background()

	// Test missing query
	args1 := Args{
		Table: "test_search_table",
	}
	_, err := s.handler.Execute(ctx, args1)
	s.Error(err, "Should error when query is missing")

	// Test missing table
	args2 := Args{
		Query: "test",
	}
	_, err = s.handler.Execute(ctx, args2)
	s.Error(err, "Should error when table is missing")

	// Test non-existent table
	args3 := Args{
		Query: "test",
		Table: "non_existent_table",
	}
	_, err = s.handler.Execute(ctx, args3)
	s.Error(err, "Should error for non-existent table")
}

func (s *SearchTestSuite) TestSearchWithCluster() {
	ctx := context.Background()

	// Test with cluster prefix (will fail if no cluster, but should handle gracefully)
	args := Args{
		Query:   "laptop",
		Table:   "test_search_table",
		Cluster: "test_cluster",
		Limit:   5,
	}

	result, err := s.handler.Execute(ctx, args)
	// This might error if cluster doesn't exist, which is expected
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func TestSearchSuite(t *testing.T) {
	suite.Run(t, new(SearchTestSuite))
}
