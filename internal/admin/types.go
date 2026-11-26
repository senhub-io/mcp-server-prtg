// Package admin serves the local administration page of the MCP server.
package admin

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/senhub-io/mcp-server-prtg/internal/services/configuration"
)

// Store reads and writes the editable settings.
type Store interface {
	Settings() configuration.Settings
	ApplySettings(configuration.Settings) error
	GetAPIKey() string
}

// TestResult is the outcome of a connection test.
type TestResult struct {
	OK         bool     `json:"ok"`
	Message    string   `json:"message"`
	Details    []string `json:"details,omitempty"`
	DurationMs int64    `json:"duration_ms"`
}

// Deps are the services the administration page relies on.
type Deps struct {
	Store        Store
	Status       func(ctx context.Context) any
	TestDatabase func(ctx context.Context, s configuration.DatabaseSettings) TestResult
	TestPRTG     func(ctx context.Context, s configuration.PRTGSettings) TestResult
	ClientConfig func(host string) (text string, resolvedHost string, err error)
	Restart      func()
	Logger       *zerolog.Logger
}
