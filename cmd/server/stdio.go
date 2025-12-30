package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/matthieu/mcp-server-prtg/internal/handlers"
	"github.com/matthieu/mcp-server-prtg/internal/prtg"
	"github.com/matthieu/mcp-server-prtg/internal/version"
)

// runStdioMode starts the MCP server in stdio mode for use as a local plugin.
// This mode is used by MCP clients (Claude Desktop, Cursor, etc.) to run
// the server as a child process communicating via stdin/stdout.
func runStdioMode() error {
	// Initialize logger (CRITICAL: write to stderr, not stdout)
	// stdout is reserved for MCP protocol communication
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("mode", "stdio").
		Logger()

	logger.Info().Msg("Starting MCP Server PRTG in stdio mode")

	// Load configuration from environment variables
	config, err := loadConfigFromEnv(&logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load configuration")
		return err
	}

	// Validate required configuration
	if config.PRTGURL == "" {
		logger.Error().Msg("PRTG_URL environment variable is required")
		return fmt.Errorf("PRTG_URL is required")
	}
	if config.PRTGToken == "" {
		logger.Error().Msg("PRTG_API_TOKEN environment variable is required")
		return fmt.Errorf("PRTG_API_TOKEN is required")
	}

	logger.Info().
		Str("prtg_url", config.PRTGURL).
		Bool("verify_ssl", config.VerifySSL).
		Int("timeout", config.Timeout).
		Msg("Configuration loaded from environment")

	// Initialize PRTG client
	prtgClient, err := prtg.NewClient(prtg.ClientConfig{
		BaseURL:   config.PRTGURL,
		Token:     config.PRTGToken,
		Timeout:   time.Duration(config.Timeout) * time.Second,
		VerifySSL: config.VerifySSL,
		Logger:    &logger,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create PRTG client")
		return err
	}

	// Test PRTG connectivity (optional, quick check)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := prtgClient.Ping(ctx); err != nil {
		logger.Warn().Err(err).Msg("PRTG connectivity check failed - will retry on first request")
	} else {
		logger.Info().Msg("PRTG API connectivity verified")
	}

	// Create MCP server
	mcpServer := mcpserver.NewMCPServer(
		"mcp-server-prtg",
		version.Get(),
		mcpserver.WithLogging(),
	)

	logger.Info().
		Str("name", "mcp-server-prtg").
		Str("version", version.Get()).
		Msg("MCP server created")

	// Initialize tool handler (no database in stdio mode, API v2 only)
	toolHandler := handlers.NewToolHandler(nil, nil, &logger)

	// Register database-free tools (if any)
	// For now, only metrics tools work in stdio mode
	// TODO: Migrate all tools to API v2 to work without database
	toolHandler.RegisterTools(mcpServer)

	toolsCount := 0 // Will be updated as we register tools

	// Initialize metrics tools (PRTG API v2)
	if config.PRTGEnabled {
		metricsHandler := handlers.NewMetricsToolHandler(prtgClient, toolHandler)
		metricsHandler.RegisterMetricsTools(mcpServer)
		toolsCount += 3 // prtg_get_sensor_timeseries, prtg_get_sensor_history_custom, prtg_get_channel_current_values

		logger.Info().
			Int("tools_count", 3).
			Msg("PRTG API v2 metrics tools registered")
	}

	logger.Info().
		Int("total_tools", toolsCount).
		Msg("All MCP tools registered")

	// Setup context with cancellation
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")
		cancel()
	}()

	// Start MCP server in stdio mode
	// The mcp-go library handles the stdio transport automatically
	logger.Info().Msg("MCP stdio server starting")

	if err := mcpserver.ServeStdio(mcpServer); err != nil {
		logger.Error().Err(err).Msg("MCP stdio server error")
		return err
	}

	logger.Info().Msg("MCP stdio server stopped")
	return nil
}

// Config holds stdio mode configuration loaded from environment variables.
type Config struct {
	PRTGURL     string
	PRTGToken   string
	PRTGEnabled bool
	Timeout     int
	VerifySSL   bool
	LogLevel    string
}

// loadConfigFromEnv loads configuration from environment variables.
// This replaces the YAML config file for stdio mode.
func loadConfigFromEnv(logger *zerolog.Logger) (*Config, error) {
	config := &Config{
		PRTGURL:     getEnv("PRTG_URL", ""),
		PRTGToken:   getEnv("PRTG_API_TOKEN", ""),
		PRTGEnabled: getEnvBool("PRTG_ENABLED", true),
		Timeout:     getEnvInt("PRTG_TIMEOUT", 30, logger),
		VerifySSL:   getEnvBool("PRTG_VERIFY_SSL", true),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	return config, nil
}

// getEnv returns an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns an environment variable as int or a default value.
// Logs a warning to stderr if the value cannot be parsed as an integer.
func getEnvInt(key string, defaultValue int, logger *zerolog.Logger) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
		// Log warning for invalid value
		if logger != nil {
			logger.Warn().
				Str("key", key).
				Str("value", value).
				Int("default", defaultValue).
				Msg("Invalid integer value for environment variable, using default")
		}
	}
	return defaultValue
}

// getEnvBool returns an environment variable as bool or a default value.
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	switch value {
	case "true", "TRUE", "1", "yes", "YES":
		return true
	case "false", "FALSE", "0", "no", "NO":
		return false
	default:
		return defaultValue
	}
}
