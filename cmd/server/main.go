package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
	"github.com/matthieu/mcp-server-prtg/internal/version"
)

// Build-time variables injected via ldflags.
var (
	Version    = "v1.0.0"  // Injected via -X flag at build time
	CommitHash = "unknown" // Injected via -X flag at build time
	BuildTime  = "unknown" // Injected via -X flag at build time
	GoVersion  = "unknown" // Injected via -X flag at build time
)

const (
	cmdRun       = "run"
	cmdInstall   = "install"
	cmdUninstall = "uninstall"
	cmdStart     = "start"
	cmdStop      = "stop"
	cmdStatus    = "status"
	cmdConfig    = "config"
)

func main() {
	// Initialize application version
	version.Set(Version)

	// Parse CLI arguments with version info
	args := cliArgs.ParseWithVersion(getVersionString())

	// Execute command
	if err := executeCommand(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// executeCommand executes the specified command.
func executeCommand(args *cliArgs.ParsedArgs) error {
	// If no command provided, show help
	if args.Command == "" {
		return showHelp()
	}

	switch args.Command {
	case cmdRun:
		// Run via service framework (handles both console and service mode)
		return runService(args)

	case cmdInstall:
		fmt.Println("Installing MCP Server PRTG as system service...")

		// Ensure configuration exists before installing service
		if err := ensureConfiguration(args); err != nil {
			return fmt.Errorf("failed to create configuration: %w", err)
		}

		if err := installService(args); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
		fmt.Println("✓ Service installed successfully")
		fmt.Printf("  Configuration: %s\n", args.ConfigPath)
		fmt.Printf("  Use '%s start' to start the service\n", os.Args[0])
		return nil

	case cmdUninstall:
		fmt.Println("Uninstalling MCP Server PRTG service...")
		if err := uninstallService(args); err != nil {
			return fmt.Errorf("failed to uninstall service: %w", err)
		}
		fmt.Println("✓ Service uninstalled successfully")
		return nil

	case cmdStart:
		fmt.Println("Starting MCP Server PRTG service...")
		if err := startService(args); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}
		fmt.Println("✓ Service started successfully")
		// Wait a bit and check status
		time.Sleep(2 * time.Second)
		return getServiceStatus(args)

	case cmdStop:
		fmt.Println("Stopping MCP Server PRTG service...")
		if err := stopService(args); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		fmt.Println("✓ Service stopped successfully")
		return nil

	case cmdStatus:
		return getServiceStatus(args)

	case cmdConfig:
		return handleConfigCommand(args)

	default:
		return fmt.Errorf("unknown command: %s\n\nAvailable commands: run, install, uninstall, start, stop, status, config", args.Command)
	}
}

// handleConfigCommand handles config-related commands.
func handleConfigCommand(args *cliArgs.ParsedArgs) error {
	fmt.Println("Configuration management")
	fmt.Printf("  Config file: %s\n", args.ConfigPath)
	fmt.Println("\nTo generate a new configuration file, run:")
	fmt.Printf("  %s run --config %s\n", os.Args[0], args.ConfigPath)
	fmt.Println("\nThis will create a new config with auto-generated API key and TLS certificates.")
	return nil
}

// getVersionString returns the full version string.
func getVersionString() string {
	return fmt.Sprintf("mcp-server-prtg %s (commit: %s, built: %s, %s)",
		Version, CommitHash, BuildTime, GoVersion)
}

// ensureConfiguration creates configuration file if it doesn't exist.
func ensureConfiguration(args *cliArgs.ParsedArgs) error {
	// Check if config file already exists
	if _, err := os.Stat(args.ConfigPath); err == nil {
		fmt.Printf("  Configuration file already exists: %s\n", args.ConfigPath)
		return nil
	}

	// Create directory for config file if needed
	configDir := filepath.Dir(args.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create silent logger for config generation (no console output)
	tempLogger := logger.NewSilentLogger()

	// Create configuration (will generate default if not exists)
	config, err := configuration.NewConfiguration(args, tempLogger)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Shutdown config to close file watcher
	_ = config.Shutdown(nil)

	fmt.Printf("  ✓ Configuration file created: %s\n", args.ConfigPath)
	fmt.Printf("  ✓ API Key generated (see config file)\n")
	if config.IsTLSEnabled() {
		fmt.Printf("  ✓ TLS certificates generated\n")
	}

	return nil
}

// showHelp displays usage information.
func showHelp() error {
	fmt.Printf("MCP Server PRTG - %s\n\n", getVersionString())
	fmt.Println("USAGE:")
	fmt.Printf("  %s <command> [options]\n\n", os.Args[0])
	fmt.Println("COMMANDS:")
	fmt.Println("  run         Run the server in console mode (foreground)")
	fmt.Println("  install     Install as system service")
	fmt.Println("  uninstall   Uninstall system service")
	fmt.Println("  start       Start the system service")
	fmt.Println("  stop        Stop the system service")
	fmt.Println("  status      Show service status")
	fmt.Println("  config      Show configuration information")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --config PATH       Path to configuration file (default: ./config.yaml)")
	fmt.Println("  --db-password PASS  Database password (or set PRTG_DB_PASSWORD)")
	fmt.Println("  --version           Show version information")
	fmt.Println("  --help, -h          Show this help message")
	fmt.Println()
	fmt.Println("NOTE:")
	fmt.Println("  Log level is controlled via config.yaml (log_level: debug|info|warn|error)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Printf("  %s run --config /etc/mcp-server-prtg/config.yaml\n", os.Args[0])
	fmt.Printf("  %s install --config /etc/mcp-server-prtg/config.yaml\n", os.Args[0])
	fmt.Printf("  %s start\n", os.Args[0])
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/matthieu/mcp-server-prtg")
	return nil
}
