package client

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"manticore-mcp-server/config"
	"manticore-mcp-server/testutils"
)

type ClientTestSuite struct {
	suite.Suite
	client ManticoreClient
	cfg    *config.Config
}

func (s *ClientTestSuite) SetupSuite() {
	s.cfg = testutils.LoadTestConfig()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Reduce noise in tests
	}))
	s.client = New(s.cfg, logger)

	// Wait for Manticore to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to connect with retries
	for i := 0; i < 30; i++ {
		if err := s.client.Ping(ctx); err == nil {
			return
		}
		time.Sleep(1 * time.Second)
	}

	s.T().Fatal("Failed to connect to Manticore after 30 seconds")
}

func (s *ClientTestSuite) TestPing() {
	ctx := context.Background()
	err := s.client.Ping(ctx)
	s.NoError(err)
}

func (s *ClientTestSuite) TestExecuteSQL_ShowTables() {
	ctx := context.Background()

	result, err := s.client.ExecuteSQL(ctx, "SHOW TABLES")
	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestExecuteSQL_ShowStatus() {
	ctx := context.Background()

	result, err := s.client.ExecuteSQL(ctx, "SHOW STATUS")
	s.NoError(err)
	s.NotNil(result)
	s.NotEmpty(result, "SHOW STATUS should return at least one row")
}

func (s *ClientTestSuite) TestExecuteSQL_CreateAndDropTable() {
	ctx := context.Background()
	tableName := "test_table_create_drop"

	// Create table
	createSQL := "CREATE TABLE " + tableName + " (id bigint, title text)"
	_, err := s.client.ExecuteSQL(ctx, createSQL)
	s.NoError(err)

	// Verify table exists
	result, err := s.client.ExecuteSQL(ctx, "SHOW TABLES LIKE '"+tableName+"'")
	s.NoError(err)
	s.Len(result, 1)
	// Check if table exists in result (field name may vary)
	found := false
	for _, row := range result {
		for _, value := range row {
			if value == tableName {
				found = true
				break
			}
		}
	}
	s.True(found, "Table should be found in SHOW TABLES result")

	// Drop table
	dropSQL := "DROP TABLE " + tableName
	_, err = s.client.ExecuteSQL(ctx, dropSQL)
	s.NoError(err)

	// Verify table is gone
	result, err = s.client.ExecuteSQL(ctx, "SHOW TABLES LIKE '"+tableName+"'")
	s.NoError(err)
	s.Empty(result)
}

func (s *ClientTestSuite) TestExecuteSQL_InsertAndSelect() {
	ctx := context.Background()
	tableName := "test_insert_select"

	// Create table
	createSQL := "CREATE TABLE " + tableName + " (id bigint, title text, content text)"
	_, err := s.client.ExecuteSQL(ctx, createSQL)
	s.NoError(err)

	defer func() {
		// Cleanup
		s.client.ExecuteSQL(ctx, "DROP TABLE "+tableName)
	}()

	// Insert document
	insertSQL := "INSERT INTO " + tableName + " (id, title, content) VALUES (1, 'Test Title', 'Test content for search')"
	_, err = s.client.ExecuteSQL(ctx, insertSQL)
	s.NoError(err)

	// Select document
	selectSQL := "SELECT * FROM " + tableName + " WHERE id = 1"
	result, err := s.client.ExecuteSQL(ctx, selectSQL)
	s.NoError(err)
	s.Len(result, 1)
	s.InDelta(1.0, result[0]["id"], 0.01) // JSON numbers come as float64
	s.Equal("Test Title", result[0]["title"])
}

func (s *ClientTestSuite) TestExecuteSQL_InvalidQuery() {
	ctx := context.Background()

	_, err := s.client.ExecuteSQL(ctx, "INVALID SQL SYNTAX")
	s.Error(err)
	s.Contains(err.Error(), "SQL request failed")
}

func (s *ClientTestSuite) TestExecuteSQL_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel the context

	_, err := s.client.ExecuteSQL(ctx, "SHOW TABLES")
	if s.Error(err) {
		s.Contains(err.Error(), "context canceled")
	}
}

func TestClientIntegration(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

// Unit tests for edge cases that don't require Manticore
func TestClient_New(t *testing.T) {
	cfg := &config.Config{
		ManticoreURL:   "http://localhost:19308",
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	client := New(cfg, logger)

	assert.NotNil(t, client)

	// Test that client implements interface
	_ = client
}
