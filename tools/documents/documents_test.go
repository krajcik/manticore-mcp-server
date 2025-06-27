package documents

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

type DocumentsTestSuite struct {
	suite.Suite
	handler *Handler
	client  client.ManticoreClient
	cfg     *config.Config
}

func (s *DocumentsTestSuite) SetupSuite() {
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

func (s *DocumentsTestSuite) TearDownSuite() {
	// Clean up test table
	ctx := context.Background()
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_documents_table")
}

func (s *DocumentsTestSuite) createTestTable() {
	s.T().Helper()

	ctx := context.Background()

	// Drop if exists
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_documents_table")

	// Create table
	createSQL := `CREATE TABLE test_documents_table (
		id bigint,
		title text,
		content text,
		price float,
		category int,
		active bool,
		tags string
	)`
	_, err := s.client.ExecuteSQL(ctx, createSQL)
	s.Require().NoError(err)
}

func (s *DocumentsTestSuite) SetupTest() {
	// Clear table before each test
	ctx := context.Background()
	s.client.ExecuteSQL(ctx, "TRUNCATE RTINDEX test_documents_table")
}

func (s *DocumentsTestSuite) TestInsertDocument() {
	ctx := context.Background()

	id := int64(1)
	args := InsertDocumentArgs{
		Table: "test_documents_table",
		ID:    &id,
		Document: map[string]interface{}{
			"title":    "Test Product",
			"content":  "This is a test product description",
			"price":    99.99,
			"category": 1,
			"active":   true,
			"tags":     "test,product",
		},
	}

	result, err := s.handler.InsertDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was inserted
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Len(checkResult, 1)
	s.Equal("Test Product", checkResult[0]["title"])
	s.InDelta(99.99, checkResult[0]["price"], 0.01)
}

func (s *DocumentsTestSuite) TestInsertDocumentAutoID() {
	ctx := context.Background()

	args := InsertDocumentArgs{
		Table: "test_documents_table",
		Document: map[string]interface{}{
			"title":   "Auto ID Product",
			"content": "Product with auto-generated ID",
			"price":   49.99,
		},
	}

	result, err := s.handler.InsertDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was inserted (ID should be auto-generated)
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE MATCH('Auto ID Product')")
	s.NoError(err)
	s.Len(checkResult, 1)
	s.Equal("Auto ID Product", checkResult[0]["title"])
}

func (s *DocumentsTestSuite) TestReplaceDocument() {
	ctx := context.Background()

	// First insert
	id := int64(1)
	args1 := InsertDocumentArgs{
		Table: "test_documents_table",
		ID:    &id,
		Document: map[string]interface{}{
			"title": "Original Product",
			"price": 99.99,
		},
	}
	_, err := s.handler.InsertDocument(ctx, args1)
	s.NoError(err)

	// Replace with new data
	args2 := InsertDocumentArgs{
		Table:   "test_documents_table",
		ID:      &id,
		Replace: true,
		Document: map[string]interface{}{
			"title": "Replaced Product",
			"price": 149.99,
		},
	}

	result, err := s.handler.InsertDocument(ctx, args2)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was replaced
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Len(checkResult, 1)
	s.Equal("Replaced Product", checkResult[0]["title"])
	s.InDelta(149.99, checkResult[0]["price"], 0.01)
}

func (s *DocumentsTestSuite) TestUpdateDocument() {
	ctx := context.Background()

	// First insert a document
	id := int64(1)
	args1 := InsertDocumentArgs{
		Table: "test_documents_table",
		ID:    &id,
		Document: map[string]interface{}{
			"title":    "Original Product",
			"content":  "Original content",
			"price":    99.99,
			"category": 1,
			"active":   true,
		},
	}
	_, err := s.handler.InsertDocument(ctx, args1)
	s.NoError(err)

	// Update the document (only attributes, not text fields)
	args2 := UpdateDocumentArgs{
		Table: "test_documents_table",
		ID:    1,
		Document: map[string]interface{}{
			"price":    149.99,
			"category": 2,
		},
	}

	result, err := s.handler.UpdateDocument(ctx, args2)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was updated
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Len(checkResult, 1)
	s.Equal("Original Product", checkResult[0]["title"]) // Text fields can't be updated
	s.InDelta(149.99, checkResult[0]["price"], 0.01)
	s.InDelta(2.0, checkResult[0]["category"], 0.01)       // Should be updated
	s.Equal("Original content", checkResult[0]["content"]) // Should remain unchanged
}

func (s *DocumentsTestSuite) TestUpdateDocumentWithCondition() {
	ctx := context.Background()

	// Insert test documents
	id1, id2 := int64(1), int64(2)
	docs := []InsertDocumentArgs{
		{
			Table: "test_documents_table",
			ID:    &id1,
			Document: map[string]interface{}{
				"title":    "Product 1",
				"price":    99.99,
				"category": 1,
			},
		},
		{
			Table: "test_documents_table",
			ID:    &id2,
			Document: map[string]interface{}{
				"title":    "Product 2",
				"price":    199.99,
				"category": 2,
			},
		},
	}

	for _, doc := range docs {
		_, err := s.handler.InsertDocument(ctx, doc)
		s.NoError(err)
	}

	// Update with additional condition (only attributes)
	args := UpdateDocumentArgs{
		Table:     "test_documents_table",
		ID:        1,
		Condition: "category = 1",
		Document: map[string]interface{}{
			"price": 299.99,
		},
	}

	result, err := s.handler.UpdateDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify only the matching document was updated
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Len(checkResult, 1)
	s.InDelta(299.99, checkResult[0]["price"], 0.01)
}

func (s *DocumentsTestSuite) TestDeleteDocumentByID() {
	ctx := context.Background()

	// Insert test documents
	id1, id2 := int64(1), int64(2)
	docs := []InsertDocumentArgs{
		{
			Table: "test_documents_table",
			ID:    &id1,
			Document: map[string]interface{}{
				"title": "Product 1",
				"price": 99.99,
			},
		},
		{
			Table: "test_documents_table",
			ID:    &id2,
			Document: map[string]interface{}{
				"title": "Product 2",
				"price": 199.99,
			},
		},
	}

	for _, doc := range docs {
		_, err := s.handler.InsertDocument(ctx, doc)
		s.NoError(err)
	}

	// Delete document by ID
	deleteID := int64(1)
	args := DeleteDocumentArgs{
		Table: "test_documents_table",
		ID:    &deleteID,
	}

	result, err := s.handler.DeleteDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was deleted
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Empty(checkResult)

	// Verify other document still exists
	checkResult2, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 2")
	s.NoError(err)
	s.Len(checkResult2, 1)
}

func (s *DocumentsTestSuite) TestDeleteDocumentByCondition() {
	ctx := context.Background()

	// Insert test documents
	id1, id2, id3 := int64(1), int64(2), int64(3)
	docs := []InsertDocumentArgs{
		{
			Table: "test_documents_table",
			ID:    &id1,
			Document: map[string]interface{}{
				"title":    "Product 1",
				"price":    99.99,
				"category": 1,
			},
		},
		{
			Table: "test_documents_table",
			ID:    &id2,
			Document: map[string]interface{}{
				"title":    "Product 2",
				"price":    199.99,
				"category": 1,
			},
		},
		{
			Table: "test_documents_table",
			ID:    &id3,
			Document: map[string]interface{}{
				"title":    "Product 3",
				"price":    299.99,
				"category": 2,
			},
		},
	}

	for _, doc := range docs {
		_, err := s.handler.InsertDocument(ctx, doc)
		s.NoError(err)
	}

	// Delete documents by condition
	args := DeleteDocumentArgs{
		Table:     "test_documents_table",
		Condition: "category = 1",
	}

	result, err := s.handler.DeleteDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify documents with category 1 were deleted
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE category = 1")
	s.NoError(err)
	s.Empty(checkResult)

	// Verify document with category 2 still exists
	checkResult2, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE category = 2")
	s.NoError(err)
	s.Len(checkResult2, 1)
}

func (s *DocumentsTestSuite) TestDeleteDocumentByIDAndCondition() {
	ctx := context.Background()

	// Insert test documents
	id1, id2 := int64(1), int64(2)
	docs := []InsertDocumentArgs{
		{
			Table: "test_documents_table",
			ID:    &id1,
			Document: map[string]interface{}{
				"title":    "Product 1",
				"category": 1,
			},
		},
		{
			Table: "test_documents_table",
			ID:    &id2,
			Document: map[string]interface{}{
				"title":    "Product 2",
				"category": 2,
			},
		},
	}

	for _, doc := range docs {
		_, err := s.handler.InsertDocument(ctx, doc)
		s.NoError(err)
	}

	// Delete by both ID and condition
	deleteID := int64(1)
	args := DeleteDocumentArgs{
		Table:     "test_documents_table",
		ID:        &deleteID,
		Condition: "category = 1",
	}

	result, err := s.handler.DeleteDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify specific document was deleted
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Empty(checkResult)

	// Verify other document still exists
	checkResult2, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 2")
	s.NoError(err)
	s.Len(checkResult2, 1)
}

func (s *DocumentsTestSuite) TestDocumentValueTypes() {
	ctx := context.Background()

	// Test various data types
	id := int64(1)
	args := InsertDocumentArgs{
		Table: "test_documents_table",
		ID:    &id,
		Document: map[string]interface{}{
			"title":    "Type Test",
			"content":  "String with 'quotes' and \"double quotes\"",
			"price":    123.45,
			"category": 42,
			"active":   true,
			"tags":     "",
		},
	}

	result, err := s.handler.InsertDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)

	// Verify document was inserted with correct types
	checkResult, err := s.client.ExecuteSQL(ctx, "SELECT * FROM test_documents_table WHERE id = 1")
	s.NoError(err)
	s.Len(checkResult, 1)

	row := checkResult[0]
	s.Equal("Type Test", row["title"])
	s.Equal("String with 'quotes' and \"double quotes\"", row["content"]) // String preserved as-is
	s.InDelta(123.45, row["price"], 0.01)
	s.InDelta(42.0, row["category"], 0.01) // JSON numbers come as float64
	s.InDelta(1.0, row["active"], 0.01)    // Boolean true becomes 1
}

func (s *DocumentsTestSuite) TestDocumentWithCluster() {
	ctx := context.Background()

	// Test with cluster prefix (will fail if no cluster, but should handle gracefully)
	id := int64(1)
	args := InsertDocumentArgs{
		Table:   "test_documents_table",
		Cluster: "test_cluster",
		ID:      &id,
		Document: map[string]interface{}{
			"title": "Cluster Test",
			"price": 99.99,
		},
	}

	result, err := s.handler.InsertDocument(ctx, args)
	// This might error if cluster doesn't exist, which is expected
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func (s *DocumentsTestSuite) TestDocumentErrors() {
	ctx := context.Background()

	// Test missing table
	args1 := InsertDocumentArgs{
		Document: map[string]interface{}{
			"title": "Test",
		},
	}
	_, err := s.handler.InsertDocument(ctx, args1)
	s.Error(err, "Should error when table is missing")

	// Test empty document
	args2 := InsertDocumentArgs{
		Table:    "test_documents_table",
		Document: map[string]interface{}{},
	}
	_, err = s.handler.InsertDocument(ctx, args2)
	s.Error(err, "Should error when document is empty")

	// Test update missing table
	args3 := UpdateDocumentArgs{
		ID: 1,
		Document: map[string]interface{}{
			"title": "Test",
		},
	}
	_, err = s.handler.UpdateDocument(ctx, args3)
	s.Error(err, "Should error when table is missing for update")

	// Test delete missing table
	args4 := DeleteDocumentArgs{}
	_, err = s.handler.DeleteDocument(ctx, args4)
	s.Error(err, "Should error when table is missing for delete")

	// Test delete without ID or condition
	args5 := DeleteDocumentArgs{
		Table: "test_documents_table",
	}
	_, err = s.handler.DeleteDocument(ctx, args5)
	s.Error(err, "Should error when both ID and condition are missing for delete")
}

func (s *DocumentsTestSuite) TestNilValues() {
	ctx := context.Background()

	id := int64(1)
	args := InsertDocumentArgs{
		Table: "test_documents_table",
		ID:    &id,
		Document: map[string]interface{}{
			"title":   "Nil Test",
			"content": nil,
			"price":   99.99,
		},
	}

	result, err := s.handler.InsertDocument(ctx, args)
	s.NoError(err)
	s.NotNil(result)
}

func TestDocumentsSuite(t *testing.T) {
	suite.Run(t, new(DocumentsTestSuite))
}
