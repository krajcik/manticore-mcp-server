package clusters

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

type ClustersTestSuite struct {
	suite.Suite
	handler *Handler
	client  client.ManticoreClient
	cfg     *config.Config
}

func (s *ClustersTestSuite) SetupSuite() {
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
}

func (s *ClustersTestSuite) TearDownSuite() {
	// Clean up any test clusters
	ctx := context.Background()

	// Check what clusters exist first
	statusResult, err := s.client.ExecuteSQL(ctx, "SHOW STATUS LIKE 'cluster%'")
	if err == nil && len(statusResult) > 0 {
		// Only delete clusters that actually exist
		for _, row := range statusResult {
			if varName, ok := row["Variable_name"].(string); ok {
				if varName == "cluster_test_cluster_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster")
				}
				if varName == "cluster_test_cluster_with_path_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster_with_path")
				}
				if varName == "cluster_test_cluster_with_nodes_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster_with_nodes")
				}
			}
		}
	}
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_cluster_table")
}

func (s *ClustersTestSuite) SetupTest() {
	// Clean up any existing test clusters before each test
	ctx := context.Background()

	// Drop test table first
	s.client.ExecuteSQL(ctx, "DROP TABLE IF EXISTS test_cluster_table")

	// Check what clusters exist first
	statusResult, err := s.client.ExecuteSQL(ctx, "SHOW STATUS LIKE 'cluster%'")
	if err == nil && len(statusResult) > 0 {
		// Only delete clusters that actually exist
		for _, row := range statusResult {
			if varName, ok := row["Variable_name"].(string); ok {
				if varName == "cluster_test_cluster_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster")
				}
				if varName == "cluster_test_cluster_with_path_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster_with_path")
				}
				if varName == "cluster_test_cluster_with_nodes_nodes_set" {
					s.client.ExecuteSQL(ctx, "DELETE CLUSTER test_cluster_with_nodes")
				}
			}
		}
	}
}

func (s *ClustersTestSuite) TestShowClusterStatus() {
	ctx := context.Background()

	args := ShowClusterStatusArgs{}
	result, err := s.handler.ShowClusterStatus(ctx, args)

	s.NoError(err)
	s.NotNil(result)

	// Check that we get status information
	// Note: Even without clusters, SHOW STATUS should return server status
	s.NotEmpty(result)
}

func (s *ClustersTestSuite) TestShowClusterStatusWithPattern() {
	ctx := context.Background()

	// Test with pattern to filter status variables
	args := ShowClusterStatusArgs{
		Pattern: "cluster%",
	}
	result, err := s.handler.ShowClusterStatus(ctx, args)

	s.NoError(err)
	s.NotNil(result)

	// Filter should work even if no cluster variables exist
}

func (s *ClustersTestSuite) TestCreateClusterBasic() {
	ctx := context.Background()

	args := CreateClusterArgs{
		Name: "test_cluster",
	}

	result, err := s.handler.CreateCluster(ctx, args)

	// Note: This might fail in single-node setup without replication support
	// We'll check both success and expected failure cases
	if err != nil {
		// Expected errors for single-node setups
		s.Contains(err.Error(), "cluster", "Error should mention cluster")
	} else {
		s.NotNil(result)

		// If successful, verify cluster was created by checking status
		statusResult, statusErr := s.handler.ShowClusterStatus(ctx, ShowClusterStatusArgs{
			Pattern: "cluster%",
		})
		s.NoError(statusErr)
		s.NotNil(statusResult)
	}
}

func (s *ClustersTestSuite) TestCreateClusterWithPath() {
	ctx := context.Background()

	args := CreateClusterArgs{
		Name: "test_cluster_with_path",
		Path: "/tmp/test_cluster_data",
	}

	result, err := s.handler.CreateCluster(ctx, args)

	// Note: This might fail in single-node setup
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func (s *ClustersTestSuite) TestCreateClusterWithNodes() {
	ctx := context.Background()

	args := CreateClusterArgs{
		Name:  "test_cluster_with_nodes",
		Path:  "/tmp/test_cluster_nodes",
		Nodes: []string{"localhost:9312", "localhost:9313"},
	}

	result, err := s.handler.CreateCluster(ctx, args)

	// Note: This might fail in single-node setup
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func (s *ClustersTestSuite) TestJoinClusterBasic() {
	ctx := context.Background()

	args := JoinClusterArgs{
		Name: "test_cluster",
		At:   "localhost:9312",
	}

	_, err := s.handler.JoinCluster(ctx, args)

	// This will fail if cluster doesn't exist or no replication support
	s.Error(err, "Should error when trying to join non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestJoinClusterWithNodes() {
	ctx := context.Background()

	args := JoinClusterArgs{
		Name:  "test_cluster",
		Nodes: []string{"localhost:9312", "localhost:9313"},
	}

	_, err := s.handler.JoinCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to join non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestJoinClusterWithPath() {
	ctx := context.Background()

	args := JoinClusterArgs{
		Name: "test_cluster",
		At:   "localhost:9312",
		Path: "/tmp/test_join_path",
	}

	_, err := s.handler.JoinCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to join non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestAlterClusterAddTable() {
	ctx := context.Background()

	// First create a test table
	createTableSQL := `CREATE TABLE test_cluster_table (
		id bigint,
		title text,
		content text
	)`
	_, err := s.client.ExecuteSQL(ctx, createTableSQL)
	s.NoError(err)

	args := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "add",
		Table:     "test_cluster_table",
	}

	_, err = s.handler.AlterCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to alter non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestAlterClusterDropTable() {
	ctx := context.Background()

	args := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "drop",
		Table:     "test_cluster_table",
	}

	_, err := s.handler.AlterCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to alter non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestAlterClusterUpdateNodes() {
	ctx := context.Background()

	args := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "update_nodes",
	}

	_, err := s.handler.AlterCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to alter non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestDeleteCluster() {
	ctx := context.Background()

	args := DeleteClusterArgs{
		Name: "test_cluster",
	}

	result, err := s.handler.DeleteCluster(ctx, args)

	// This will fail if cluster doesn't exist, but should not crash
	if err != nil {
		s.Contains(err.Error(), "cluster")
	} else {
		s.NotNil(result)
	}
}

func (s *ClustersTestSuite) TestSetClusterGlobal() {
	ctx := context.Background()

	args := SetClusterArgs{
		Name:     "test_cluster",
		Variable: "pc.bootstrap",
		Value:    "1",
		Global:   true,
	}

	_, err := s.handler.SetCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to set variable on non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestSetClusterLocal() {
	ctx := context.Background()

	args := SetClusterArgs{
		Name:     "test_cluster",
		Variable: "pc.weight",
		Value:    "100",
		Global:   false,
	}

	_, err := s.handler.SetCluster(ctx, args)

	// This will fail if cluster doesn't exist
	s.Error(err, "Should error when trying to set variable on non-existent cluster")
	s.Contains(err.Error(), "cluster")
}

func (s *ClustersTestSuite) TestClusterErrors() {
	ctx := context.Background()

	// Test missing cluster name for create
	args1 := CreateClusterArgs{}
	_, err := s.handler.CreateCluster(ctx, args1)
	s.Error(err, "Should error when cluster name is missing")

	// Test missing cluster name for join
	args2 := JoinClusterArgs{
		At: "localhost:9312",
	}
	_, err = s.handler.JoinCluster(ctx, args2)
	s.Error(err, "Should error when cluster name is missing")

	// Test missing at and nodes for join
	args3 := JoinClusterArgs{
		Name: "test_cluster",
	}
	_, err = s.handler.JoinCluster(ctx, args3)
	s.Error(err, "Should error when both 'at' and 'nodes' are missing")

	// Test missing operation for alter
	args4 := AlterClusterArgs{
		Name: "test_cluster",
	}
	_, err = s.handler.AlterCluster(ctx, args4)
	s.Error(err, "Should error when operation is missing")

	// Test unsupported operation for alter
	args5 := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "invalid_operation",
	}
	_, err = s.handler.AlterCluster(ctx, args5)
	s.Error(err, "Should error for unsupported operation")

	// Test missing table for add operation
	args6 := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "add",
	}
	_, err = s.handler.AlterCluster(ctx, args6)
	s.Error(err, "Should error when table name is missing for add operation")

	// Test missing table for drop operation
	args7 := AlterClusterArgs{
		Name:      "test_cluster",
		Operation: "drop",
	}
	_, err = s.handler.AlterCluster(ctx, args7)
	s.Error(err, "Should error when table name is missing for drop operation")

	// Test missing cluster name for delete
	args8 := DeleteClusterArgs{}
	_, err = s.handler.DeleteCluster(ctx, args8)
	s.Error(err, "Should error when cluster name is missing for delete")

	// Test missing cluster name for set
	args9 := SetClusterArgs{
		Variable: "test_var",
		Value:    "test_value",
	}
	_, err = s.handler.SetCluster(ctx, args9)
	s.Error(err, "Should error when cluster name is missing for set")

	// Test missing variable for set
	args10 := SetClusterArgs{
		Name:  "test_cluster",
		Value: "test_value",
	}
	_, err = s.handler.SetCluster(ctx, args10)
	s.Error(err, "Should error when variable name is missing for set")

	// Test missing value for set
	args11 := SetClusterArgs{
		Name:     "test_cluster",
		Variable: "test_var",
	}
	_, err = s.handler.SetCluster(ctx, args11)
	s.Error(err, "Should error when variable value is missing for set")
}

func (s *ClustersTestSuite) TestSpecialCharacters() {
	ctx := context.Background()

	// Test cluster name with special characters (should be handled safely)
	args1 := CreateClusterArgs{
		Name: "test'; DROP TABLE test; --",
	}
	_, err := s.handler.CreateCluster(ctx, args1)
	// Should either error (invalid name) or be handled safely
	if err != nil {
		s.Contains(err.Error(), "cluster")
	}

	// Test path with special characters
	args2 := CreateClusterArgs{
		Name: "test_cluster",
		Path: "/tmp/test'path",
	}
	_, err = s.handler.CreateCluster(ctx, args2)
	// Should handle escaping properly
	if err != nil {
		s.Contains(err.Error(), "cluster")
	}

	// Test variable with special characters
	args3 := SetClusterArgs{
		Name:     "test_cluster",
		Variable: "test'var",
		Value:    "test'value",
	}
	_, err = s.handler.SetCluster(ctx, args3)
	// Should handle escaping properly
	if err != nil {
		s.Contains(err.Error(), "cluster")
	}
}

func (s *ClustersTestSuite) TestNumericValues() {
	// Test numeric value detection
	s.True(s.handler.isNumeric("123"))
	s.True(s.handler.isNumeric("123.45"))
	s.True(s.handler.isNumeric("-123"))
	s.True(s.handler.isNumeric("-123.45"))
	s.False(s.handler.isNumeric("abc"))
	s.False(s.handler.isNumeric("123abc"))
	s.False(s.handler.isNumeric(""))
	s.False(s.handler.isNumeric("12.34.56"))
}

func TestClustersSuite(t *testing.T) {
	suite.Run(t, new(ClustersTestSuite))
}
