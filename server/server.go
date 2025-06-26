package server

import (
	"log/slog"
)

// Server handles MCP protocol communication
type Server struct {
	logger *slog.Logger
}

// New creates a new MCP server
func New(toolHandler interface{}, logger *slog.Logger) *Server {
	return &Server{
		logger: logger,
	}
}

// Run starts the MCP server
func (s *Server) Run() error {
	s.logger.Info("Server starting...")
	return nil
}
