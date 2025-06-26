package tables

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"manticore-mcp-server/client"
	"manticore-mcp-server/config"
)

type TablesTestSuite struct {
	suite.Suite
	handler *Handler
	client  client.ManticoreClient
	cfg     *config.Config
}

func (s *TablesTestSuite) SetupSuite() {
	s.cfg = &config.Config{
		ManticoreURL:   "http://localhost:19308",
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
	}

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

	// Create test tables
	s.createTestTables()
}

func (s *TablesTestSuite) TearDownSuite() {
	// Clean up test tables
	ctx := context.Background()
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_table_products")
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_table_orders")
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS products_archive")
}

func (s *TablesTestSuite) createTestTables() {
	ctx := context.Background()

	// Drop existing tables
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_table_products")
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_table_orders")
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS products_archive")

	// Create test tables with different structures
	tables := []string{
		`CREATE TABLE test_table_products (
			id bigint,
			title text,
			description text,
			price float,
			category_id int,
			created_at timestamp
		)`,
		`CREATE TABLE test_table_orders (
			id bigint,
			customer_name text,
			total_amount float,
			status int
		)`,
		`CREATE TABLE products_archive (
			id bigint,
			title text,
			archived_date timestamp
		)`,
	}

	for _, createSQL := range tables {
		_, err := s.client.ExecuteSQL(ctx, createSQL)
		s.Require().NoError(err)
	}

	// Insert some test data
	insertSQL := []string{
		"INSERT INTO test_table_products (id, title, description, price, category_id) VALUES (1, 'Test Product', 'Test Description', 99.99, 1)",
		"INSERT INTO test_table_orders (id, customer_name, total_amount, status) VALUES (1, 'John Doe', 199.98, 1)",
		"INSERT INTO products_archive (id, title) VALUES (1, 'Archived Product')",
	}

	for _, insertStmt := range insertSQL {
		_, err := s.client.ExecuteSQL(ctx, insertStmt)
		s.Require().NoError(err)
	}
}

func (s *TablesTestSuite) TestShowTables() {
	ctx := context.Background()

	args := ShowTablesArgs{}
	result, err := s.handler.ShowTables(ctx, args)

	s.NoError(err)
	s.NotEmpty(result)

	// Should find our test tables
	tableNames := make([]string, 0)
	for _, row := range result {
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	s.Contains(tableNames, "test_table_products")
	s.Contains(tableNames, "test_table_orders")
	s.Contains(tableNames, "products_archive")
}

func (s *TablesTestSuite) TestShowTablesWithPattern() {
	ctx := context.Background()

	// Test pattern matching for tables starting with "test_table_"
	args := ShowTablesArgs{
		Pattern: "test_table_%",
	}
	result, err := s.handler.ShowTables(ctx, args)

	s.NoError(err)
	s.NotEmpty(result)

	// Extract table names from results
	tableNames := make([]string, 0)
	for _, row := range result {
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	// Should contain tables matching pattern
	s.Contains(tableNames, "test_table_products")
	s.Contains(tableNames, "test_table_orders")

	// Should not contain tables not matching pattern
	s.NotContains(tableNames, "products_archive")
}

func (s *TablesTestSuite) TestShowTablesWithPatternProducts() {
	ctx := context.Background()

	// Test pattern matching for tables containing "products"
	args := ShowTablesArgs{
		Pattern: "%products%",
	}
	result, err := s.handler.ShowTables(ctx, args)

	s.NoError(err)

	// Extract table names from results
	tableNames := make([]string, 0)
	for _, row := range result {
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	// Should contain tables with "products" in name
	expectedTables := []string{"test_table_products", "products_archive"}
	for _, expected := range expectedTables {
		s.Contains(tableNames, expected)
	}

	// Should not contain tables without "products" in name
	s.NotContains(tableNames, "test_table_orders")
}

func (s *TablesTestSuite) TestDescribeTable() {
	ctx := context.Background()

	args := DescribeTableArgs{
		Table: "test_table_products",
	}
	result, err := s.handler.DescribeTable(ctx, args)

	s.NoError(err)
	s.NotEmpty(result)

	// Extract field information
	fields := make(map[string]string)
	for _, row := range result {
		if field, ok := row["Field"].(string); ok {
			if fieldType, ok := row["Type"].(string); ok {
				fields[field] = fieldType
			}
		}
	}

	// Check expected fields
	s.Contains(fields, "id")
	s.Contains(fields, "title")
	s.Contains(fields, "description")
	s.Contains(fields, "price")
	s.Contains(fields, "category_id")

	// Check field types
	s.Contains(fields["id"], "bigint")
	s.Contains(fields["title"], "text")
	s.Contains(fields["price"], "float")
}

func (s *TablesTestSuite) TestDescribeTableOrders() {
	ctx := context.Background()

	args := DescribeTableArgs{
		Table: "test_table_orders",
	}
	result, err := s.handler.DescribeTable(ctx, args)

	s.NoError(err)
	s.NotEmpty(result)

	// Extract field information
	fields := make(map[string]string)
	for _, row := range result {
		if field, ok := row["Field"].(string); ok {
			if fieldType, ok := row["Type"].(string); ok {
				fields[field] = fieldType
			}
		}
	}

	// Check expected fields for orders table
	s.Contains(fields, "id")
	s.Contains(fields, "customer_name")
	s.Contains(fields, "total_amount")
	s.Contains(fields, "status")

	// Check field types
	s.Contains(fields["customer_name"], "text")
	s.Contains(fields["total_amount"], "float")
}

func (s *TablesTestSuite) TestDescribeTableWithCluster() {
	ctx := context.Background()

	// Test with cluster prefix (will fail if no cluster, but should handle gracefully)
	args := DescribeTableArgs{
		Table:   "test_table_products",
		Cluster: "test_cluster",
	}
	result, err := s.handler.DescribeTable(ctx, args)

	// This might error if cluster doesn't exist, which is expected
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func (s *TablesTestSuite) TestDescribeTableErrors() {
	ctx := context.Background()

	// Test missing table name
	args1 := DescribeTableArgs{}
	_, err := s.handler.DescribeTable(ctx, args1)
	s.Error(err, "Should error when table name is missing")

	// Test non-existent table
	args2 := DescribeTableArgs{
		Table: "non_existent_table",
	}
	_, err = s.handler.DescribeTable(ctx, args2)
	s.Error(err, "Should error for non-existent table")
}

func (s *TablesTestSuite) TestShowTablesWithSpecialCharacters() {
	ctx := context.Background()

	// Test pattern with SQL special characters (should be escaped)
	args := ShowTablesArgs{
		Pattern: "test_table'_%", // Include single quote to test escaping
	}
	result, err := s.handler.ShowTables(ctx, args)

	// Should not error even with special characters
	s.NoError(err)
	s.NotNil(result)
}

func (s *TablesTestSuite) TestDescribeTableWithSpecialCharacters() {
	ctx := context.Background()

	// Test table name with SQL injection attempt (should be handled safely)
	args := DescribeTableArgs{
		Table: "test'; DROP TABLE test_table_products; --",
	}
	_, err := s.handler.DescribeTable(ctx, args)

	// Should error but not execute the injection
	s.Error(err)

	// Verify our table still exists
	checkArgs := ShowTablesArgs{
		Pattern: "test_table_products",
	}
	result, err := s.handler.ShowTables(ctx, checkArgs)
	s.NoError(err)
	s.NotEmpty(result, "Original table should still exist")
}

func (s *TablesTestSuite) TestEmptyPattern() {
	ctx := context.Background()

	// Test empty pattern (should show all tables)
	args := ShowTablesArgs{
		Pattern: "",
	}
	result, err := s.handler.ShowTables(ctx, args)

	s.NoError(err)
	s.NotEmpty(result)

	// Should include all our test tables
	tableNames := make([]string, 0)
	for _, row := range result {
		for _, value := range row {
			if tableName, ok := value.(string); ok {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	s.Contains(tableNames, "test_table_products")
	s.Contains(tableNames, "test_table_orders")
	s.Contains(tableNames, "products_archive")
}

func TestTablesSuite(t *testing.T) {
	suite.Run(t, new(TablesTestSuite))
}
