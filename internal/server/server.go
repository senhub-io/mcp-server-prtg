package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
	"github.com/matthieu/mcp-server-prtg/internal/config"
	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/handlers"
)

// Server represents the MCP server instance
type Server struct {
	mcpServer *server.MCPServer
	db        *database.DB
	logger    *slog.Logger
}

// New creates a new MCP server instance
func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	// Initialize database connection
	db, err := database.New(cfg.Database.ConnectionString(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"prtg-server",
		"1.0.0",
	)

	// Register tools
	toolHandler := handlers.NewToolHandler(db, logger)
	toolHandler.RegisterTools(mcpServer)

	logger.Info("MCP server initialized",
		"server_name", "prtg-server",
		"version", "1.0.0",
		"tools_count", 6,
	)

	return &Server{
		mcpServer: mcpServer,
		db:        db,
		logger:    logger,
	}, nil
}

// Start starts the MCP server
func (s *Server) Start(_ctx context.Context) error {
	s.logger.Info("starting PRTG MCP server")

	// Run server in stdio mode - this is blocking
	return server.ServeStdio(s.mcpServer)
}

// Close cleanly shuts down the server
func (s *Server) Close() error {
	s.logger.Info("shutting down server")

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("error closing database", "error", err)
			return err
		}
	}

	return nil
}
