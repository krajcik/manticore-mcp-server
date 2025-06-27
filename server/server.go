package server

import (
	"log/slog"
	"manticore-mcp-server/config"
	"manticore-mcp-server/mcp"
	"manticore-mcp-server/tools"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// Server handles MCP protocol communication
type Server struct {
	toolHandler *tools.Handler
	config      *config.Config
	logger      *slog.Logger
}

// New creates a new MCP server
func New(toolHandler *tools.Handler, cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{
		toolHandler: toolHandler,
		config:      cfg,
		logger:      logger,
	}
}

// Run starts the MCP server
func (s *Server) Run() error {
	s.logger.Info("Starting Manticore Search MCP Server...")

	// Create stdio transport for Claude Code
	transport := stdio.NewStdioServerTransport()

	// Create MCP server
	server := mcp_golang.NewServer(transport)

	// Create MCP registry
	registry := mcp.NewRegistry(s.toolHandler, s.config, s.logger)

	// Register all tools
	if err := registry.RegisterAll(server); err != nil {
		s.logger.Error("Failed to register tools", "error", err)
		return err
	}

	s.logger.Info("Tools registered successfully")

	// Start server
	if err := server.Serve(); err != nil {
		s.logger.Error("MCP server error", "error", err)
		return err
	}

	return nil
}
