package main

import (
	"fmt"
	"os"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/cliArgs"
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
	// Parse CLI arguments
	args := cliArgs.Parse()

	// Execute command
	if err := executeCommand(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// executeCommand executes the specified command.
func executeCommand(args *cliArgs.ParsedArgs) error {
	switch args.Command {
	case cmdRun, "":
		// Run in console mode (default command)
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
