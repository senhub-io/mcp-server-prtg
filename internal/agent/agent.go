package agent

import (
	"context"
	"fmt"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/handlers"
	"github.com/matthieu/mcp-server-prtg/internal/server"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
)

// Agent represents the main application orchestrator.
type Agent struct {
	config     *configuration.Configuration
	logger     *logger.Logger
	db         *database.DB
	httpServer *server.StreamableHTTPServer
	args       *cliArgs.ParsedArgs
	shutdownCh chan struct{} // Channel to signal shutdown
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

	// Initialize database (optional - server can start without database)
	dbLogger := logger.NewModuleLogger(baseLogger, logger.ModuleDatabase)
	connStr := config.GetDatabaseConnectionString()

	// Log connection attempt (with masked password)
	moduleLogger.Debug().
		Str("host", config.GetDatabaseHost()).
		Int("port", config.GetDatabasePort()).
		Str("database", config.GetDatabaseName()).
		Str("user", config.GetDatabaseUser()).
		Str("sslmode", config.GetDatabaseSSLMode()).
		Msg("Attempting database connection")

	db, err := database.New(connStr, dbLogger.Logger)
	if err != nil {
		moduleLogger.Warn().
			Err(err).
			Str("sslmode", config.GetDatabaseSSLMode()).
			Msg("Failed to initialize database - server will start but tools will not work")
		db = nil
	} else {
		moduleLogger.Info().Msg("Database connection established")
	}

	// Create MCP server
	mcpServer := mcpserver.NewMCPServer(
		"prtg-server",
		"1.0.0",
	)

	// Register MCP tools
	toolHandler := handlers.NewToolHandler(db, config, baseLogger)
	toolHandler.RegisterTools(mcpServer)

	moduleLogger.Info().
		Int("tools_count", 6).
		Msg("MCP tools registered")

	// Create Streamable HTTP server (modern MCP transport)
	httpServer := server.NewStreamableHTTPServer(mcpServer, db, config, baseLogger)

	return &Agent{
		config:     config,
		logger:     baseLogger,
		db:         db,
		httpServer: httpServer,
		args:       args,
		shutdownCh: make(chan struct{}),
	}, nil
}

// Start starts the agent.
func (a *Agent) Start() error {
	moduleLogger := logger.NewModuleLogger(a.logger, "agent")
	moduleLogger.Info().Msg("Starting agent")

	// Start Streamable HTTP server
	ctx := context.Background()
	if err := a.httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Wait for shutdown signal (server runs in goroutine)
	<-a.shutdownCh
	moduleLogger.Info().Msg("Shutdown signal received, agent stopping")

	return nil
}

// Shutdown gracefully shuts down the agent.
func (a *Agent) Shutdown(ctx context.Context) error {
	moduleLogger := logger.NewModuleLogger(a.logger, "agent")
	moduleLogger.Info().Msg("Shutting down agent")

	// Signal shutdown to Start() (close channel only once)
	select {
	case <-a.shutdownCh:
		// Already closed
	default:
		close(a.shutdownCh)
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			moduleLogger.Error().Err(err).Msg("Error shutting down HTTP server")
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
