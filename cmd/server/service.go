package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kardianos/service"
	"github.com/matthieu/mcp-server-prtg/internal/agent"
	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
)

// program implements the service.Interface.
type program struct {
	agent     *agent.Agent
	args      *cliArgs.ParsedArgs
	done      chan bool
	appLogger *logger.Logger
}

// Start is called when the service starts.
func (p *program) Start(_ service.Service) error {
	p.appLogger.Info().Msg("MCP Server PRTG service starting...")

	// Start agent initialization and execution in background
	// This allows the service to respond quickly to Windows Service Manager
	p.done = make(chan bool, 1)
	go p.run()

	return nil
}

// run executes the agent (initialization + start).
func (p *program) run() {
	p.appLogger.Info().Msg("Initializing agent...")

	// Initialize agent (may take time due to DB connection attempts)
	var err error
	p.agent, err = agent.NewAgent(p.args)
	if err != nil {
		p.appLogger.Error().Err(err).Msg("Failed to create agent")
		p.done <- true
		return
	}

	p.appLogger.Info().Msg("Agent initialized successfully")

	// Start agent
	p.appLogger.Info().Msg("Starting agent server...")

	if err := p.agent.Start(); err != nil {
		p.appLogger.Error().Err(err).Msg("Agent error")
	}

	p.appLogger.Info().Msg("Agent stopped")
	p.done <- true
}

// Stop is called when the service stops.
func (p *program) Stop(_ service.Service) error {
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if p.agent != nil {
		if err := p.agent.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown agent: %w", err)
		}
	}

	<-p.done
	return nil
}

// installService installs the service.
func installService(args *cliArgs.ParsedArgs) error {
	// Get executable directory
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	workingDir := filepath.Dir(execPath)

	// Create logs directory
	logFile := args.LogFile
	if !filepath.IsAbs(logFile) {
		logFile = filepath.Join(workingDir, logFile)
	}
	logDir := filepath.Dir(logFile)

	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	fmt.Printf("✅ Log directory created: %s\n", logDir)
	fmt.Printf("   Logs will be written to: %s\n", logFile)

	svc, err := createService(args)
	if err != nil {
		return err
	}

	return svc.Install()
}

// uninstallService uninstalls the service.
func uninstallService(args *cliArgs.ParsedArgs) error {
	svc, err := createService(args)
	if err != nil {
		return err
	}

	if err := svc.Uninstall(); err != nil {
		return err
	}

	// Clean up configuration, logs, and certificates
	cleanupFiles(args)

	return nil
}

// startService starts the service.
func startService(args *cliArgs.ParsedArgs) error {
	svc, err := createService(args)
	if err != nil {
		return err
	}

	return svc.Start()
}

// stopService stops the service.
func stopService(args *cliArgs.ParsedArgs) error {
	svc, err := createService(args)
	if err != nil {
		return err
	}

	return svc.Stop()
}

// runService runs the agent via service framework (handles both console and service mode).
// This is the correct way to run the agent in all contexts - exactly like senhub-agent.
func runService(args *cliArgs.ParsedArgs) error {
	// Get executable path and working directory
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	workingDir := args.WorkingDir
	if workingDir == "" {
		workingDir = filepath.Dir(executablePath)
	}

	// Convert relative paths to absolute (based on working directory)
	// IMPORTANT: Modify args directly so logger uses absolute paths
	if args.ConfigPath != "" && !filepath.IsAbs(args.ConfigPath) {
		args.ConfigPath = filepath.Join(workingDir, args.ConfigPath)
	}

	if args.LogFile != "" && !filepath.IsAbs(args.LogFile) {
		args.LogFile = filepath.Join(workingDir, args.LogFile)
	}

	// Create logger early for better logging
	appLogger := logger.NewLogger(args)

	// Service configuration
	svcConfig := &service.Config{
		Name:             args.ServiceName,
		DisplayName:      "MCP Server PRTG",
		Description:      "MCP Server for PRTG monitoring data - provides remote access via SSE transport",
		Executable:       executablePath,
		WorkingDirectory: workingDir,
	}

	// Create program
	prg := &program{
		args:      args,
		appLogger: appLogger,
		done:      make(chan bool, 1),
	}

	// Create service
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Get service logger
	svcLogger, err := svc.Logger(nil)
	if err != nil {
		appLogger.Warn().Err(err).Msg("Failed to create service logger")
	}

	// Interactive mode (console): run with signal handling
	if service.Interactive() {
		appLogger.Info().Msg("Running in interactive/console mode")

		// Start agent directly
		if err := prg.Start(svc); err != nil {
			appLogger.Error().Err(err).Msg("Failed to start agent")
			return fmt.Errorf("failed to start: %w", err)
		}

		// Setup signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			sig := <-sigChan
			appLogger.Info().Msgf("Received signal: %v", sig)
			appLogger.Info().Msg("Shutting down gracefully...")
			if err := prg.Stop(svc); err != nil {
				appLogger.Error().Err(err).Msg("Error stopping service")
			}
		}()

		// Wait for completion
		<-prg.done
		appLogger.Info().Msg("Agent stopped")
		return nil
	}

	// Service mode: let the service framework handle everything
	if svcLogger != nil {
		if err := svcLogger.Info("Starting service"); err != nil {
			appLogger.Warn().Err(err).Msg("Failed to log service start message")
		}
	}

	if err := svc.Run(); err != nil {
		if svcLogger != nil {
			if logErr := svcLogger.Error(fmt.Sprintf("Error running service: %v", err)); logErr != nil {
				appLogger.Warn().Err(logErr).Msg("Failed to log service error")
			}
		}
		return fmt.Errorf("service failed to run: %w", err)
	}

	return nil
}

// createService creates a service instance.
func createService(args *cliArgs.ParsedArgs) (service.Service, error) {
	// Get executable path
	executablePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Determine working directory
	workingDir := args.WorkingDir
	if workingDir == "" {
		workingDir = filepath.Dir(executablePath)
	}

	// Convert relative paths to absolute (based on working directory)
	// IMPORTANT: Modify args directly so logger uses absolute paths
	if args.ConfigPath != "" && !filepath.IsAbs(args.ConfigPath) {
		args.ConfigPath = filepath.Join(workingDir, args.ConfigPath)
	}

	if args.LogFile != "" && !filepath.IsAbs(args.LogFile) {
		args.LogFile = filepath.Join(workingDir, args.LogFile)
	}

	// Service arguments
	// Note: Log level is controlled by config file, not CLI flags
	serviceArgs := []string{"run"}
	if args.ConfigPath != "" {
		serviceArgs = append(serviceArgs, "--config", args.ConfigPath)
	}
	if args.LogFile != "" {
		serviceArgs = append(serviceArgs, "--log-file", args.LogFile)
	}

	// Service configuration
	svcConfig := &service.Config{
		Name:             args.ServiceName,
		DisplayName:      "MCP Server PRTG",
		Description:      "MCP Server for PRTG monitoring data - provides remote access via SSE transport",
		Executable:       executablePath,
		Arguments:        serviceArgs,
		WorkingDirectory: workingDir,
		Option: service.KeyValue{
			"LogOutput":             true,
			"User":                  "root",
			"ServiceName":           args.ServiceName + ".service",
			"SystemdScript":         true,
			"Restart":               "always",
			"RestartSec":            "10",
			"StartLimitIntervalSec": "0",
			"StartLimitBurst":       "0",
		},
	}

	// Create logger early (before service starts)
	// args.LogFile is now absolute, so logger will write to correct location
	appLogger := logger.NewLogger(args)

	// Create program
	prg := &program{
		args:      args,
		appLogger: appLogger,
	}

	// Create service
	return service.New(prg, svcConfig)
}

// getServiceStatus returns the service status with detailed information.
func getServiceStatus(args *cliArgs.ParsedArgs) error {
	svc, err := createService(args)
	if err != nil {
		return err
	}

	status, err := svc.Status()
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	var statusText string
	var statusSymbol string
	switch status {
	case service.StatusRunning:
		statusText = "Running"
		statusSymbol = "✅"
	case service.StatusStopped:
		statusText = "Stopped"
		statusSymbol = "⏹️"
	case service.StatusUnknown:
		statusText = "Unknown"
		statusSymbol = "❓"
	default:
		statusText = "Unknown"
		statusSymbol = "❓"
	}

	// Display status header
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("  MCP Server PRTG - Service Status\n")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("%s Service Status:  %s\n", statusSymbol, statusText)
	fmt.Printf("📋 Service Name:    %s\n", args.ServiceName)
	fmt.Println()

	// Display configuration
	fmt.Println("Configuration:")
	fmt.Printf("  Config File:  %s\n", args.ConfigPath)
	fmt.Printf("  Log File:     %s\n", args.LogFile)
	fmt.Printf("  Log Level:    %s\n", args.LogLevel)
	fmt.Println()

	// Try to load config for more details (only if config exists)
	if _, err := os.Stat(args.ConfigPath); err == nil {
		// Create a silent logger for status check
		silentLogger := logger.NewSilentLogger()
		config, err := configuration.NewConfiguration(args, silentLogger)
		if err == nil {
			defer config.Shutdown(context.Background())

			fmt.Println("Server:")
			fmt.Printf("  Address:      %s\n", config.GetServerAddress())
			fmt.Printf("  Public URL:   %s\n", config.GetPublicURL())
			fmt.Printf("  TLS Enabled:  %v\n", config.IsTLSEnabled())
			if config.IsTLSEnabled() {
				fmt.Printf("  Certificate:  %s\n", config.GetTLSCertFile())
			}
			fmt.Println()

			// Database information
			fmt.Println("Database:")
			fmt.Printf("  Host:     %s:%d\n", config.GetDatabaseHost(), config.GetDatabasePort())
			fmt.Printf("  Database: %s\n", config.GetDatabaseName())
			fmt.Printf("  User:     %s\n", config.GetDatabaseUser())
			fmt.Printf("  SSL Mode: %s\n", config.GetDatabaseSSLMode())

			// Try to test database connection
			connStr := config.GetDatabaseConnectionString()
			db, err := database.New(connStr, silentLogger)
			if err != nil {
				fmt.Printf("  Status:   ❌ Connection Failed (%v)\n", err)
			} else {
				defer db.Close()
				fmt.Printf("  Status:   ✅ Connected\n")
			}
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════")
	return nil
}

// cleanupFiles removes configuration files, logs, and certificates during uninstall.
func cleanupFiles(args *cliArgs.ParsedArgs) {
	var filesToRemove []string
	var dirsToRemove []string

	// Get executable directory
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Warning: Could not get executable path: %v\n", err)
		return
	}
	workingDir := filepath.Dir(execPath)

	// Configuration file
	configPath := args.ConfigPath
	if configPath == "" {
		configPath = filepath.Join(workingDir, "config.yaml")
	}
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workingDir, configPath)
	}
	if _, err := os.Stat(configPath); err == nil {
		filesToRemove = append(filesToRemove, configPath)
	}

	// Certificate directory
	certsDir := filepath.Join(workingDir, "certs")
	if _, err := os.Stat(certsDir); err == nil {
		dirsToRemove = append(dirsToRemove, certsDir)
	}

	// Log directory
	logFile := args.LogFile
	if logFile == "" {
		logFile = filepath.Join(workingDir, "logs", "mcp-server-prtg.log")
	}
	if !filepath.IsAbs(logFile) {
		logFile = filepath.Join(workingDir, logFile)
	}
	logDir := filepath.Dir(logFile)
	if _, err := os.Stat(logDir); err == nil {
		dirsToRemove = append(dirsToRemove, logDir)
	}

	// Remove files
	for _, file := range filesToRemove {
		if err := os.Remove(file); err != nil {
			fmt.Printf("Warning: Could not remove %s: %v\n", file, err)
		} else {
			fmt.Printf("✅ Removed: %s\n", file)
		}
	}

	// Remove directories
	for _, dir := range dirsToRemove {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("Warning: Could not remove directory %s: %v\n", dir, err)
		} else {
			fmt.Printf("✅ Removed directory: %s\n", dir)
		}
	}

	if len(filesToRemove) == 0 && len(dirsToRemove) == 0 {
		fmt.Println("✅ No additional files to clean up")
	} else {
		fmt.Printf("\n🧹 Cleanup completed - removed %d files and %d directories\n",
			len(filesToRemove), len(dirsToRemove))
	}
}
