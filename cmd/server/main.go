package main

import (
	"fmt"
	"os"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
)

// Build-time variables injected via ldflags.
var (
	Version    = "v1.0.0"     // Injected via -X flag at build time
	CommitHash = "unknown"    // Injected via -X flag at build time
	BuildTime  = "unknown"    // Injected via -X flag at build time
	GoVersion  = "unknown"    // Injected via -X flag at build time
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
		// Run in console mode
		return runConsole(args)

	case cmdInstall:
		fmt.Println("Installing MCP Server PRTG as system service...")
		if err := installService(args); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
		fmt.Println("✓ Service installed successfully")
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
	fmt.Println("  --verbose, -v       Enable verbose logging")
	fmt.Println("  --version           Show version information")
	fmt.Println("  --help, -h          Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Printf("  %s run --config /etc/mcp-server-prtg/config.yaml\n", os.Args[0])
	fmt.Printf("  %s install --config /etc/mcp-server-prtg/config.yaml\n", os.Args[0])
	fmt.Printf("  %s start\n", os.Args[0])
	fmt.Println()
	fmt.Println("For more information, see: https://github.com/matthieu/mcp-server-prtg")
	return nil
}
