package cliargs

import "github.com/alexflint/go-arg"

// ParsedArgs contains all command-line arguments for the MCP Server.
type ParsedArgs struct {
	// Command to execute
	Command string `arg:"positional" help:"Command to execute (run, install, start, stop, uninstall, config)"`

	// Configuration
	ConfigPath string `arg:"--config,-c" help:"Path to configuration file" default:"./config.yaml"`

	// Server settings
	Port        int    `arg:"--port,-p" help:"Server port" default:"8443"`
	BindAddress string `arg:"--bind" help:"Bind address" default:"0.0.0.0"`
	EnableHTTPS bool   `arg:"--https" help:"Enable HTTPS" default:"true"`
	CertFile    string `arg:"--cert" help:"Path to TLS certificate file"`
	KeyFile     string `arg:"--key" help:"Path to TLS private key file"`
	AuthKey     string `arg:"--api-key" help:"API key for authentication"`

	// Database settings
	DBHost     string `arg:"--db-host" help:"Database host" env:"PRTG_DB_HOST"`
	DBPort     int    `arg:"--db-port" help:"Database port" env:"PRTG_DB_PORT"`
	DBName     string `arg:"--db-name" help:"Database name" env:"PRTG_DB_NAME"`
	DBUser     string `arg:"--db-user" help:"Database user" env:"PRTG_DB_USER"`
	DBPassword string `arg:"--db-password" help:"Database password" env:"PRTG_DB_PASSWORD"`
	DBSSLMode  string `arg:"--db-sslmode" help:"Database SSL mode" env:"PRTG_DB_SSLMODE"`

	// Logging
	Verbose      bool     `arg:"--verbose,-v" help:"Enable verbose logging (debug level)"`
	LogLevel     string   `arg:"--log-level" help:"Log level (debug, info, warn, error)" default:"info"`
	LogFile      string   `arg:"--log-file" help:"Log file path" default:"./logs/mcp-server-prtg.log"`
	DebugModules []string `arg:"--debug-modules" help:"Comma-separated list of modules to debug"`

	// Service
	ServiceName string `arg:"--service-name" help:"Service name" default:"mcp-server-prtg"`
	WorkingDir  string `arg:"--working-dir" help:"Working directory for service"`
}

// Parse parses command-line arguments.
func Parse() *ParsedArgs {
	return ParseWithVersion("mcp-server-prtg v1.0.0 (dev)")
}

// ParseWithVersion parses command-line arguments with a custom version string.
func ParseWithVersion(version string) *ParsedArgs {
	args := &argsWithVersion{
		version: version,
	}
	arg.MustParse(args)

	return &args.ParsedArgs
}

// argsWithVersion wraps ParsedArgs with a version string.
type argsWithVersion struct {
	ParsedArgs
	version string
}

// Description returns the program description.
func (a *argsWithVersion) Description() string {
	return "MCP Server for PRTG monitoring data - provides remote access to PRTG PostgreSQL database via MCP protocol over SSE transport"
}

// Version returns the version information.
func (a *argsWithVersion) Version() string {
	return a.version
}
