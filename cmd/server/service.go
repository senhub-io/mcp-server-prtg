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
)

// program implements the service.Interface.
type program struct {
	agent *agent.Agent
	args  *cliArgs.ParsedArgs
	done  chan bool
}

// Start is called when the service starts.
func (p *program) Start(_ service.Service) error {
	// Initialize agent
	var err error
	p.agent, err = agent.NewAgent(p.args)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Start agent in background
	p.done = make(chan bool, 1)
	go p.run()

	return nil
}

// run executes the agent.
func (p *program) run() {
	if err := p.agent.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
	}
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

	return svc.Uninstall()
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

// runConsole runs the agent in console mode (non-service).
func runConsole(args *cliArgs.ParsedArgs) error {
	// Create agent
	ag, err := agent.NewAgent(args)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start agent in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- ag.Start()
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Shutting down gracefully...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return ag.Shutdown(ctx)

	case err := <-errChan:
		return err
	}
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

	// Service arguments
	serviceArgs := []string{"run"}
	if args.ConfigPath != "" {
		serviceArgs = append(serviceArgs, "--config", args.ConfigPath)
	}
	if args.Verbose {
		serviceArgs = append(serviceArgs, "--verbose")
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

	// Create program
	prg := &program{
		args: args,
	}

	// Create service
	return service.New(prg, svcConfig)
}

// getServiceStatus returns the service status.
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
	switch status {
	case service.StatusUnknown:
		statusText = "Unknown"
	case service.StatusRunning:
		statusText = "Running"
	case service.StatusStopped:
		statusText = "Stopped"
	default:
		statusText = "Unknown"
	}

	fmt.Printf("Service Status: %s\n", statusText)
	return nil
}
