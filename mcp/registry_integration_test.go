package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"manticore-mcp-server/client"
	"manticore-mcp-server/config"
	"manticore-mcp-server/testutils"
	"manticore-mcp-server/tools"
)

type RegistryIntegrationTestSuite struct {
	suite.Suite
	registry     *Registry
	cfg          *config.Config
	client       client.ManticoreClient
	toolsHandler *tools.Handler
}

func (s *RegistryIntegrationTestSuite) SetupSuite() {
	s.cfg = testutils.LoadTestConfig()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))

	// Initialize client
	s.client = client.New(s.cfg, logger)

	// Wait for Manticore to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to connect with retries
	for i := 0; i < 30; i++ {
		if err := s.client.Ping(ctx); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
		if i == 29 {
			s.T().Fatal("Failed to connect to Manticore after 30 seconds")
		}
	}

	// Initialize tools handler and registry
	s.toolsHandler = tools.NewHandler(s.client, logger)
	s.registry = NewRegistry(s.toolsHandler, s.cfg, logger)

	// Create test table for integration tests
	s.setupTestTable()
}

func (s *RegistryIntegrationTestSuite) setupTestTable() {
	s.T().Helper()

	ctx := context.Background()

	// Drop table if exists (ignore errors)
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_articles")

	// Create test table
	createTableSQL := `
		CREATE TABLE test_articles (
			id bigint,
			title text,
			content text,
			category_id int,
			published_at timestamp,
			tags text
		) engine='columnar'
	`

	_, err := s.client.ExecuteSQL(ctx, createTableSQL)
	s.Require().NoError(err, "Failed to create test table")

	// Insert test data
	testDocs := []string{
		"INSERT INTO test_articles (id, title, content, category_id, published_at, tags) VALUES (1, 'Go Programming Guide', 'Learn Go programming language basics and advanced concepts', 1, '2023-01-15 10:00:00', 'programming,go,tutorial')",
		"INSERT INTO test_articles (id, title, content, category_id, published_at, tags) VALUES (2, 'Manticore Search Tutorial', 'How to use Manticore Search for full-text search', 2, '2023-02-20 14:30:00', 'search,manticore,database')",
		"INSERT INTO test_articles (id, title, content, category_id, published_at, tags) VALUES (3, 'MCP Protocol Overview', 'Understanding Model Context Protocol for AI integration', 3, '2023-03-10 09:15:00', 'mcp,ai,protocol')",
		"INSERT INTO test_articles (id, title, content, category_id, published_at, tags) VALUES (4, 'Advanced Go Techniques', 'Deep dive into Go concurrency patterns and best practices', 1, '2023-04-05 16:20:00', 'programming,go,advanced')",
	}

	for _, sql := range testDocs {
		_, err := s.client.ExecuteSQL(ctx, sql)
		s.Require().NoError(err, "Failed to insert test data")
	}
}

func (s *RegistryIntegrationTestSuite) TearDownSuite() {
	if s.client != nil {
		ctx := context.Background()
		// Clean up test table
		s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_articles")
	}
}

func (s *RegistryIntegrationTestSuite) TestNewRegistry() {
	registry := NewRegistry(s.toolsHandler, s.cfg, s.registry.logger)

	s.NotNil(registry)
	s.Equal(s.toolsHandler, registry.tools)
	s.Equal(s.cfg, registry.config)
}

func (s *RegistryIntegrationTestSuite) TestHandleSearchTool() {
	tests := []struct {
		name            string
		args            map[string]interface{}
		wantErr         bool
		expectedResults int
		shouldContain   []string
		validate        func(*testing.T, map[string]interface{})
	}{
		{
			name: "search for Go programming",
			args: map[string]interface{}{
				"table": "test_articles",
				"query": "Go programming",
				"limit": 10,
			},
			wantErr:         false,
			expectedResults: 2, // Should find 2 articles about Go
			shouldContain:   []string{"Go Programming Guide", "Advanced Go Techniques"},
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				require.Len(t, data, 2, "Should find exactly 2 Go programming articles")

				// Check that results contain expected titles
				foundTitles := make(map[string]bool)
				for _, doc := range data {
					docMap := doc.(map[string]interface{})
					if title, ok := docMap["title"]; ok {
						foundTitles[title.(string)] = true
					}
				}

				require.True(t, foundTitles["Go Programming Guide"], "Should find 'Go Programming Guide'")
				require.True(t, foundTitles["Advanced Go Techniques"], "Should find 'Advanced Go Techniques'")

				meta := result["meta"].(map[string]interface{})
				require.Equal(t, "search", meta["operation"])
				require.Equal(t, "test_articles", meta["table"])
				require.InDelta(t, 2.0, meta["count"], 0.1)
			},
		},
		{
			name: "search for Manticore",
			args: map[string]interface{}{
				"table": "test_articles",
				"query": "Manticore",
				"limit": 5,
			},
			wantErr:         false,
			expectedResults: 1,
			shouldContain:   []string{"Manticore Search Tutorial"},
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				require.Len(t, data, 1, "Should find exactly 1 Manticore article")

				doc := data[0].(map[string]interface{})
				require.Equal(t, "Manticore Search Tutorial", doc["title"])
				require.Contains(t, doc["content"], "Manticore Search")
			},
		},
		{
			name: "search with specific fields",
			args: map[string]interface{}{
				"table":  "test_articles",
				"query":  "protocol",
				"fields": []interface{}{"title", "content", "tags"},
				"limit":  5,
			},
			wantErr:         false,
			expectedResults: 1,
			shouldContain:   []string{"MCP Protocol Overview"},
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				require.Len(t, data, 1, "Should find MCP Protocol article")

				doc := data[0].(map[string]interface{})
				require.Equal(t, "MCP Protocol Overview", doc["title"])
				require.Contains(t, doc["content"], "Protocol")
			},
		},
		{
			name: "search with no results",
			args: map[string]interface{}{
				"table": "test_articles",
				"query": "nonexistent keyword xyz",
				"limit": 10,
			},
			wantErr:         false,
			expectedResults: 0,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				require.Empty(t, data, "Should find no results for nonexistent keyword")

				meta := result["meta"].(map[string]interface{})
				require.Contains(t, meta, "count", "Meta should contain count field")
				require.InDelta(t, 0.0, meta["count"], 0.1)
			},
		},
		{
			name: "search with category filter",
			args: map[string]interface{}{
				"table": "test_articles",
				"query": "", // Empty query to match all for WHERE filtering
				"where": []interface{}{"category_id = 1"},
				"limit": 10,
			},
			wantErr:         false,
			expectedResults: 2, // Should find 2 articles in category 1
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				require.Len(t, data, 2, "Should find exactly 2 articles in category 1")

				// Verify all results have category_id = 1
				for _, doc := range data {
					docMap := doc.(map[string]interface{})
					require.InDelta(t, 1.0, docMap["category_id"], 0.1)
				}
			},
		},
		{
			name: "search without table",
			args: map[string]interface{}{
				"query": "test query",
			},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result, "error")
				require.Contains(t, result["error"].(string), "table parameter is required")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			response, err := s.registry.handleSearchTool(tt.args)

			// Both success and error cases should return valid response without Go error
			s.Require().NoError(err)
			s.Require().NotNil(response)

			// Parse the response content
			s.Require().Len(response.Content, 1)
			textContent := response.Content[0].TextContent.Text

			var result map[string]interface{}
			err = json.Unmarshal([]byte(textContent), &result)
			s.Require().NoError(err)

			tt.validate(s.T(), result)
		})
	}
}

func (s *RegistryIntegrationTestSuite) TestHandleShowTablesTool() {
	tests := []struct {
		name     string
		args     map[string]interface{}
		validate func(*testing.T, map[string]interface{})
	}{
		{
			name: "show all tables",
			args: map[string]interface{}{},
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))
				require.Contains(t, result, "data")

				data := result["data"].([]interface{})
				// Should contain our test table
				found := false
				for _, table := range data {
					tableMap := table.(map[string]interface{})
					if tableMap["Table"] == "test_articles" {
						found = true
						break
					}
				}
				require.True(t, found, "test_articles table should be in results")

				meta := result["meta"].(map[string]interface{})
				require.Equal(t, "show_tables", meta["operation"])
				require.Greater(t, meta["count"], float64(0), "Should have at least one table")
			},
		},
		{
			name: "show tables with pattern",
			args: map[string]interface{}{
				"pattern": "test_%",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				data := result["data"].([]interface{})
				// All returned tables should match pattern
				for _, table := range data {
					tableMap := table.(map[string]interface{})
					if tableName, ok := tableMap["Table"]; ok && tableName != nil {
						name := tableName.(string)
						require.Contains(t, name, "test_", "Table name should match pattern test_%")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			response, err := s.registry.handleShowTablesTool(tt.args)
			s.Require().NoError(err)
			s.Require().NotNil(response)

			textContent := response.Content[0].TextContent.Text
			var result map[string]interface{}
			err = json.Unmarshal([]byte(textContent), &result)
			s.Require().NoError(err)

			tt.validate(s.T(), result)
		})
	}
}

func (s *RegistryIntegrationTestSuite) TestHandleDescribeTableTool() {
	tests := []struct {
		name     string
		args     map[string]interface{}
		wantErr  bool
		validate func(*testing.T, map[string]interface{})
	}{
		{
			name: "describe existing table",
			args: map[string]interface{}{
				"table": "test_articles",
			},
			wantErr: false,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))
				require.Contains(t, result, "data")

				data := result["data"].([]interface{})
				require.NotEmpty(t, data, "Should return table schema")

				// Check that we have expected fields
				fieldNames := make(map[string]bool)
				for _, field := range data {
					fieldMap := field.(map[string]interface{})
					fieldName := fieldMap["Field"].(string)
					fieldNames[fieldName] = true
				}

				expectedFields := []string{"id", "title", "content", "category_id", "published_at", "tags"}
				for _, expected := range expectedFields {
					require.True(t, fieldNames[expected], "Should have field: %s", expected)
				}

				meta := result["meta"].(map[string]interface{})
				require.Equal(t, "describe_table", meta["operation"])
				require.Equal(t, "test_articles", meta["table"])
			},
		},
		{
			name: "describe non-existing table",
			args: map[string]interface{}{
				"table": "nonexistent_table",
			},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result, "error")
				// Should contain some indication that table doesn't exist
			},
		},
		{
			name:    "describe without table",
			args:    map[string]interface{}{},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result["error"].(string), "Table parameter is required")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			response, err := s.registry.handleDescribeTableTool(tt.args)
			s.Require().NoError(err)
			s.Require().NotNil(response)

			textContent := response.Content[0].TextContent.Text
			var result map[string]interface{}
			err = json.Unmarshal([]byte(textContent), &result)
			s.Require().NoError(err)

			tt.validate(s.T(), result)
		})
	}
}

func (s *RegistryIntegrationTestSuite) TestHandleInsertDocumentTool() {
	tests := []struct {
		name     string
		args     map[string]interface{}
		wantErr  bool
		validate func(*testing.T, map[string]interface{})
	}{
		{
			name: "insert valid document",
			args: map[string]interface{}{
				"table": "test_articles",
				"document": map[string]interface{}{
					"id":          100,
					"title":       "Integration Test Article",
					"content":     "This is a test article created during integration testing",
					"category_id": 5,
					"tags":        "test,integration,automated",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))

				meta := result["meta"].(map[string]interface{})
				require.Equal(t, "insert_document", meta["operation"])
				require.Equal(t, "test_articles", meta["table"])

				// Verify document was actually inserted by searching for it
				searchArgs := map[string]interface{}{
					"table": "test_articles",
					"query": "Integration Test Article",
					"limit": 1,
				}
				searchResponse, err := s.registry.handleSearchTool(searchArgs)
				require.NoError(t, err)

				searchContent := searchResponse.Content[0].TextContent.Text
				var searchResult map[string]interface{}
				err = json.Unmarshal([]byte(searchContent), &searchResult)
				require.NoError(t, err)

				require.True(t, searchResult["success"].(bool))
				searchData := searchResult["data"].([]interface{})
				require.Len(t, searchData, 1, "Should find the inserted document")

				doc := searchData[0].(map[string]interface{})
				require.Equal(t, "Integration Test Article", doc["title"])
				require.InDelta(t, 100.0, doc["id"], 0.1)
			},
		},
		{
			name: "insert with replace option",
			args: map[string]interface{}{
				"table": "test_articles",
				"document": map[string]interface{}{
					"id":      101,
					"title":   "Replace Test Article",
					"content": "This will be replaced",
				},
				"replace": true,
			},
			wantErr: false,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.True(t, result["success"].(bool))
				meta := result["meta"].(map[string]interface{})
				require.Equal(t, "insert_document", meta["operation"])
			},
		},
		{
			name: "insert without table",
			args: map[string]interface{}{
				"document": map[string]interface{}{
					"title": "Test",
				},
			},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result["error"].(string), "Table parameter is required")
			},
		},
		{
			name: "insert without document",
			args: map[string]interface{}{
				"table": "test_articles",
			},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result["error"].(string), "Document parameter is required")
			},
		},
		{
			name: "insert invalid document type",
			args: map[string]interface{}{
				"table":    "test_articles",
				"document": "invalid document format",
			},
			wantErr: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				t.Helper()
				require.False(t, result["success"].(bool))
				require.Contains(t, result["error"].(string), "Document must be a valid object")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			response, err := s.registry.handleInsertDocumentTool(tt.args)
			s.Require().NoError(err)
			s.Require().NotNil(response)

			textContent := response.Content[0].TextContent.Text
			var result map[string]interface{}
			err = json.Unmarshal([]byte(textContent), &result)
			s.Require().NoError(err)

			tt.validate(s.T(), result)
		})
	}
}

func (s *RegistryIntegrationTestSuite) TestHandleClusterStatusTool() {
	// Basic cluster status test
	response, err := s.registry.handleClusterStatusTool(map[string]interface{}{})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	textContent := response.Content[0].TextContent.Text
	var result map[string]interface{}
	err = json.Unmarshal([]byte(textContent), &result)
	s.Require().NoError(err)

	// Should return success (even if no clusters configured)
	s.Require().Contains(result, "success")
	meta := result["meta"].(map[string]interface{})
	s.Require().Equal("cluster_status", meta["operation"])

	// Should return some status data
	s.Require().Contains(result, "data")
	data := result["data"].([]interface{})
	// Status data might be empty if no clusters, but should be valid array
	s.Require().NotNil(data)
}

func TestRegistryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryIntegrationTestSuite))
}
