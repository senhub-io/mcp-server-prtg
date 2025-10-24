package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/handlers"
	"github.com/matthieu/mcp-server-prtg/internal/server"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Agent represents the main application orchestrator.
type Agent struct {
	config    *configuration.Configuration
	logger    *logger.Logger
	db        *database.DB
	sseServer *server.SSEServer
	args      *cliArgs.ParsedArgs
}

// NewAgent creates a new agent instance.
func NewAgent(args *cliArgs.ParsedArgs) (*Agent, error) {
	// Initialize logger
	baseLogger := logger.NewLogger(args)

	moduleLogger := logger.NewModuleLogger(baseLogger, "agent")
	moduleLogger.Info().Msg("Initializing MCP Server PRTG Agent")

	// Load or create configuration
	config, err := configuration.NewConfiguration(args, baseLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}

	moduleLogger.Info().
		Str("config_path", args.ConfigPath).
		Str("api_key_preview", maskKey(config.GetAPIKey())).
		Msg("Configuration loaded")

	// Initialize database
	dbLogger := logger.NewModuleLogger(baseLogger, logger.ModuleDatabase)
	db, err := database.New(config.GetDatabaseConnectionString(), dbLogger.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	moduleLogger.Info().Msg("Database connection established")

	// Create MCP server
	mcpServer := mcpserver.NewMCPServer(
		"prtg-server",
		"1.0.0",
	)

	// Register MCP tools
	toolHandler := handlers.NewToolHandler(db, baseLogger)
	toolHandler.RegisterTools(mcpServer)

	moduleLogger.Info().
		Int("tools_count", 6).
		Msg("MCP tools registered")

	// Create SSE server
	sseServer := server.NewSSEServer(mcpServer, config, baseLogger)

	return &Agent{
		config:    config,
		logger:    baseLogger,
		db:        db,
		sseServer: sseServer,
		args:      args,
	}, nil
}

// Start starts the agent.
func (a *Agent) Start() error {
	moduleLogger := logger.NewModuleLogger(a.logger, "agent")
	moduleLogger.Info().Msg("Starting agent")

	// Start SSE server
	ctx := context.Background()
	if err := a.sseServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SSE server: %w", err)
	}

	// Block forever (server runs in goroutine)
	select {}
}

// Shutdown gracefully shuts down the agent.
func (a *Agent) Shutdown(ctx context.Context) error {
	moduleLogger := logger.NewModuleLogger(a.logger, "agent")
	moduleLogger.Info().Msg("Shutting down agent")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown SSE server
	if a.sseServer != nil {
		if err := a.sseServer.Shutdown(shutdownCtx); err != nil {
			moduleLogger.Error().Err(err).Msg("Error shutting down SSE server")
		}
	}

	// Close database
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			moduleLogger.Error().Err(err).Msg("Error closing database")
		}
	}

	// Shutdown configuration watcher
	if a.config != nil {
		if err := a.config.Shutdown(shutdownCtx); err != nil {
			moduleLogger.Error().Err(err).Msg("Error shutting down configuration")
		}
	}

	moduleLogger.Info().Msg("Agent shut down successfully")

	return nil
}

// maskKey masks an API key for logging.
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}

	return key[:4] + "..." + key[len(key)-4:]
}
