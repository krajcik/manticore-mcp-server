package main

import (
	"log/slog"
	"manticore-mcp-server/client"
	"manticore-mcp-server/config"
	"manticore-mcp-server/server"
	"manticore-mcp-server/tools"
	"os"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	manticoreClient := client.New(cfg.ManticoreURL, logger)
	toolHandler := tools.NewHandler(manticoreClient, logger)

	mcpServer := server.New(toolHandler, logger)

	if err := mcpServer.Run(); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
