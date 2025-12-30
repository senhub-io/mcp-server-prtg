# MCP Server PRTG - Go stdio Implementation Plan

**Date**: 2025-12-30
**Objective**: Migrate from HTTP Remote Server to stdio Plugin (Keep Go)
**Version**: 1.3.0-beta.1 → 2.0.0 (stdio-based)

## Executive Summary

**Approach**: ✅ **KEEP GO, ADD STDIO MODE**

Instead of rewriting in TypeScript, we'll:
- ✅ Keep all existing Go code (~15,000 lines)
- ✅ Add stdio transport mode alongside HTTP mode
- ✅ Support both architectures (users choose)
- ✅ Distribute via npm with bundled Go binaries

**Result**: Minimal effort (2-3 days), maximum compatibility.

---

## Architecture: Dual-Mode Server

### Current (HTTP only)

```bash
# Start as HTTP server
./mcp-server-prtg run
```

### Proposed (HTTP + stdio)

```bash
# Mode 1: HTTP server (existing)
./mcp-server-prtg run

# Mode 2: stdio plugin (new)
./mcp-server-prtg stdio
```

**Same binary, different transport modes.**

---

## Technical Design

### MCP Protocol over stdio

**Protocol**: JSON-RPC 2.0 over stdin/stdout

**Example Exchange:**

```
Client → stdin:
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}

Server → stdout:
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "prtg_get_sensors",
        "description": "...",
        "inputSchema": { ... }
      }
    ]
  }
}
```

**Key Methods to Implement:**
- `initialize` - Handshake
- `tools/list` - List available tools
- `tools/call` - Execute a tool
- `notifications/` - Progress updates (optional)

---

## Code Changes Required

### 1. New stdio Server (NEW FILE)

**File**: `internal/server/stdio_server.go`

```go
package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matthieu/mcp-server-prtg/internal/handlers"
	"github.com/rs/zerolog"
)

// StdioServer handles MCP protocol over stdin/stdout
type StdioServer struct {
	toolHandler *handlers.ToolHandler
	logger      *zerolog.Logger
	reader      *bufio.Reader
	writer      io.Writer
}

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewStdioServer creates a new stdio-based MCP server
func NewStdioServer(toolHandler *handlers.ToolHandler, logger *zerolog.Logger) *StdioServer {
	return &StdioServer{
		toolHandler: toolHandler,
		logger:      logger,
		reader:      bufio.NewReader(os.Stdin),
		writer:      os.Stdout,
	}
}

// Run starts the stdio server loop
func (s *StdioServer) Run(ctx context.Context) error {
	s.logger.Info().Msg("MCP stdio server started")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Server shutting down")
			return ctx.Err()
		default:
			// Read JSON-RPC request from stdin
			line, err := s.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					s.logger.Info().Msg("Client disconnected")
					return nil
				}
				s.logger.Error().Err(err).Msg("Error reading from stdin")
				return err
			}

			// Parse request
			var req JSONRPCRequest
			if err := json.Unmarshal(line, &req); err != nil {
				s.writeError(nil, -32700, "Parse error", err.Error())
				continue
			}

			// Handle request
			s.handleRequest(ctx, req)
		}
	}
}

// handleRequest dispatches a JSON-RPC request to the appropriate handler
func (s *StdioServer) handleRequest(ctx context.Context, req JSONRPCRequest) {
	s.logger.Debug().
		Str("method", req.Method).
		Interface("id", req.ID).
		Msg("Handling JSON-RPC request")

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "ping":
		s.handlePing(req)
	default:
		s.writeError(req.ID, -32601, "Method not found", req.Method)
	}
}

// handleInitialize handles the MCP initialize request
func (s *StdioServer) handleInitialize(req JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{},
		},
		"serverInfo": map[string]string{
			"name":    "mcp-server-prtg",
			"version": "2.0.0",
		},
	}
	s.writeResponse(req.ID, result)
}

// handleToolsList returns the list of available tools
func (s *StdioServer) handleToolsList(req JSONRPCRequest) {
	// Reuse existing tool definitions from handlers package
	tools := s.toolHandler.GetToolDefinitions()

	result := map[string]interface{}{
		"tools": tools,
	}
	s.writeResponse(req.ID, result)
}

// handleToolsCall executes a tool
func (s *StdioServer) handleToolsCall(ctx context.Context, req JSONRPCRequest) {
	// Parse params
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	// Convert to MCP CallToolRequest format (reuse existing handlers)
	mcpRequest := mcp.CallToolRequest{
		Params: mcp.CallToolRequestParams{
			Name:      params.Name,
			Arguments: params.Arguments,
		},
	}

	// Execute tool using existing handler
	result, err := s.toolHandler.HandleToolCall(ctx, mcpRequest)
	if err != nil {
		s.writeError(req.ID, -32000, "Tool execution error", err.Error())
		return
	}

	s.writeResponse(req.ID, result)
}

// handlePing responds to ping requests
func (s *StdioServer) handlePing(req JSONRPCRequest) {
	s.writeResponse(req.ID, map[string]string{"status": "ok"})
}

// writeResponse writes a JSON-RPC response to stdout
func (s *StdioServer) writeResponse(id interface{}, result interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(response)
}

// writeError writes a JSON-RPC error response to stdout
func (s *StdioServer) writeError(id interface{}, code int, message string, data interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeJSON(response)
}

// writeJSON writes a JSON object to stdout with newline
func (s *StdioServer) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to marshal JSON response")
		return
	}

	// Write to stdout with newline
	if _, err := fmt.Fprintf(s.writer, "%s\n", data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to write to stdout")
	}
}
```

**Lines of Code**: ~250 lines

---

### 2. Update Main Entry Point

**File**: `cmd/server/main.go`

```go
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/matthieu/mcp-server-prtg/internal/handlers"
	"github.com/matthieu/mcp-server-prtg/internal/prtg"
	"github.com/matthieu/mcp-server-prtg/internal/server"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Parse command
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		runStdioMode()
		return
	}

	// Existing HTTP server mode
	runHTTPMode()
}

func runStdioMode() {
	// Initialize logger (write to stderr, not stdout)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger.Info().Msg("Starting MCP server in stdio mode")

	// Load configuration from environment variables
	config := loadConfigFromEnv()

	// Initialize PRTG client
	prtgClient, err := prtg.NewClient(prtg.ClientConfig{
		BaseURL:   config.PRTGURL,
		Token:     config.PRTGToken,
		Timeout:   config.Timeout,
		VerifySSL: config.VerifySSL,
		Logger:    &logger,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create PRTG client")
	}

	// Initialize tool handler (reuse existing)
	toolHandler := handlers.NewToolHandler(prtgClient, nil, &logger)

	// Create stdio server
	stdioServer := server.NewStdioServer(toolHandler, &logger)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// Run server
	if err := stdioServer.Run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server error")
	}
}

func runHTTPMode() {
	// Existing HTTP server code (unchanged)
	// ...
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() *Config {
	return &Config{
		PRTGURL:   getEnv("PRTG_URL", ""),
		PRTGToken: getEnv("PRTG_API_TOKEN", ""),
		Timeout:   getEnvInt("PRTG_TIMEOUT", 30),
		VerifySSL: getEnvBool("PRTG_VERIFY_SSL", true),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
```

**Changes**: ~100 lines added

---

### 3. Add Tool Definitions Helper

**File**: `internal/handlers/tool_definitions.go` (NEW)

```go
package handlers

import "github.com/matthieu/mcp-server-prtg/internal/mcp"

// GetToolDefinitions returns MCP tool definitions for all 15 tools
func (h *ToolHandler) GetToolDefinitions() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "prtg_get_sensors",
			Description: "Retrieve PRTG sensors with optional filters (name, device, status, tags)",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"device_name": map[string]interface{}{
						"type":        "string",
						"description": "Filter by device name (partial match, case-insensitive)",
					},
					"sensor_name": map[string]interface{}{
						"type":        "string",
						"description": "Filter by sensor name (partial match, case-insensitive)",
					},
					"status": map[string]interface{}{
						"type":        "integer",
						"description": "Filter by status code (3=Up, 4=Warning, 5=Down)",
					},
					// ... other parameters
				},
			},
		},
		// ... all other tools (copy from existing code)
	}
}
```

**Lines**: ~500 lines (tool definitions extracted from existing code)

---

## Distribution: npm Package with Go Binaries

### Package Structure

```
@senhub-io/mcp-server-prtg/
├── package.json
├── README.md
├── bin/
│   ├── index.js                       # Launcher script
│   └── binaries/
│       ├── mcp-server-prtg-darwin-amd64
│       ├── mcp-server-prtg-darwin-arm64
│       ├── mcp-server-prtg-linux-amd64
│       ├── mcp-server-prtg-linux-arm64
│       └── mcp-server-prtg-windows-amd64.exe
└── LICENSE
```

### Launcher Script

**File**: `bin/index.js`

```javascript
#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');

// Detect platform and architecture
const platform = os.platform();
const arch = os.arch();

// Map to binary names
const binaryMap = {
  'darwin-x64': 'mcp-server-prtg-darwin-amd64',
  'darwin-arm64': 'mcp-server-prtg-darwin-arm64',
  'linux-x64': 'mcp-server-prtg-linux-amd64',
  'linux-arm64': 'mcp-server-prtg-linux-arm64',
  'win32-x64': 'mcp-server-prtg-windows-amd64.exe',
};

const binaryName = binaryMap[`${platform}-${arch}`];

if (!binaryName) {
  console.error(`Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

// Path to binary
const binaryPath = path.join(__dirname, 'binaries', binaryName);

// Spawn Go binary in stdio mode
const child = spawn(binaryPath, ['stdio'], {
  stdio: ['inherit', 'inherit', 'inherit'],
  env: process.env,
});

child.on('error', (err) => {
  console.error('Failed to start MCP server:', err);
  process.exit(1);
});

child.on('exit', (code) => {
  process.exit(code || 0);
});
```

### package.json

```json
{
  "name": "@senhub-io/mcp-server-prtg",
  "version": "2.0.0",
  "description": "MCP Server for PRTG Network Monitor - stdio plugin",
  "keywords": ["mcp", "prtg", "monitoring", "model-context-protocol"],
  "author": "SenHub.io",
  "license": "MIT",
  "repository": {
    "type": "git",
    "url": "https://github.com/senhub-io/mcp-server-prtg.git"
  },
  "bin": {
    "mcp-server-prtg": "./bin/index.js"
  },
  "files": [
    "bin/",
    "README.md",
    "LICENSE"
  ],
  "engines": {
    "node": ">=16"
  },
  "os": ["darwin", "linux", "win32"],
  "cpu": ["x64", "arm64"]
}
```

**Package Size**: ~20MB (all 5 binaries included)

---

## User Configuration

### Installation

```bash
# Option 1: npx (no install, always latest)
# Just add to MCP client config

# Option 2: Global install
npm install -g @senhub-io/mcp-server-prtg
```

### Claude Desktop Config

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-prtg-api-token",
        "PRTG_VERIFY_SSL": "true",
        "PRTG_TIMEOUT": "30"
      }
    }
  }
}
```

### Alternative: Direct Binary

Users can also download binaries directly from GitHub releases and use them:

```json
{
  "mcpServers": {
    "prtg": {
      "command": "/usr/local/bin/mcp-server-prtg",
      "args": ["stdio"],
      "env": { ... }
    }
  }
}
```

---

## Implementation Plan

### Phase 1: stdio Protocol (1 day)

**Tasks:**
1. Create `internal/server/stdio_server.go`
2. Implement JSON-RPC 2.0 protocol
3. Handle `initialize`, `tools/list`, `tools/call` methods
4. Test with manual JSON-RPC requests

**Testing:**
```bash
# Manual test
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | \
  PRTG_URL=https://prtg:1616 PRTG_API_TOKEN=xxx ./mcp-server-prtg stdio
```

**Deliverable**: ✅ Working stdio mode

---

### Phase 2: Integration (1 day)

**Tasks:**
1. Update `cmd/server/main.go` with stdio mode switch
2. Extract tool definitions to `GetToolDefinitions()`
3. Environment variable configuration loading
4. Logging to stderr (not stdout - important for stdio)
5. Integration tests

**Testing:**
```bash
# Test with real MCP client
# Add to claude_desktop_config.json and test
```

**Deliverable**: ✅ stdio mode works with Claude Desktop

---

### Phase 3: npm Package (0.5 days)

**Tasks:**
1. Create npm package structure
2. Write launcher script (`bin/index.js`)
3. Build all 5 platform binaries
4. Test npm package locally
5. Publish to npm registry

**Testing:**
```bash
# Local test
npm pack
npm install -g senhub-io-mcp-server-prtg-2.0.0.tgz

# Add to MCP config and test
```

**Deliverable**: ✅ npm package published

---

### Phase 4: Documentation (0.5 days)

**Tasks:**
1. Update README.md for stdio mode
2. Add installation instructions for npm
3. Update configuration examples
4. Create migration guide from v1.x HTTP
5. Update all docs to reflect dual-mode support

**Deliverable**: ✅ Complete documentation

---

## Timeline

**Total Duration**: 3 days

| Phase | Duration | Effort |
|-------|----------|--------|
| Phase 1: stdio Protocol | 1 day | 8 hours |
| Phase 2: Integration | 1 day | 8 hours |
| Phase 3: npm Package | 0.5 days | 4 hours |
| Phase 4: Documentation | 0.5 days | 4 hours |
| **Total** | **3 days** | **24 hours** |

**Realistic with testing**: 3-4 days

---

## Testing Strategy

### Unit Tests

```go
// internal/server/stdio_server_test.go
func TestJSONRPCProtocol(t *testing.T) {
	// Test request parsing
	// Test response formatting
	// Test error handling
}

func TestToolsList(t *testing.T) {
	// Test tools/list returns all 15 tools
}

func TestToolsCall(t *testing.T) {
	// Test tools/call with mock PRTG client
}
```

### Integration Tests

```bash
# Test with real MCP client
1. Add to claude_desktop_config.json
2. Verify all 15 tools work
3. Test error handling
4. Test with invalid credentials
```

### Manual Testing

```bash
# Test JSON-RPC manually
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mcp-server-prtg stdio

# Expected output:
# {"jsonrpc":"2.0","id":1,"result":{"tools":[...]}}
```

---

## Backward Compatibility

### Dual Mode Support

The same binary supports **both** modes:

```bash
# Mode 1: HTTP server (existing users)
./mcp-server-prtg run
./mcp-server-prtg install  # Windows service
./mcp-server-prtg start

# Mode 2: stdio plugin (new users)
./mcp-server-prtg stdio
```

**Result**: No breaking changes for HTTP users, new stdio option available.

---

## Migration Path

### For Current HTTP Users (Optional)

**If they want to switch to stdio:**

1. Stop HTTP server:
   ```bash
   ./mcp-server-prtg stop
   ./mcp-server-prtg uninstall
   ```

2. Update MCP client config:
   ```json
   // OLD (HTTP)
   {
     "command": "npx",
     "args": ["mcp-remote", "https://server:8443/mcp", ...]
   }

   // NEW (stdio)
   {
     "command": "npx",
     "args": ["-y", "@senhub-io/mcp-server-prtg"],
     "env": {
       "PRTG_URL": "https://prtg:1616",
       "PRTG_API_TOKEN": "token"
     }
   }
   ```

3. Restart MCP client (Claude Desktop, etc.)

**Migration Time**: 5 minutes per user

---

## Security Considerations

### stdio Mode Security

**Advantages over HTTP:**
- ✅ No network exposure (local process only)
- ✅ No TLS certificate management
- ✅ No bearer token distribution
- ✅ Credentials stored in user's config (OS-level permissions)

**Considerations:**
- ⚠️ PRTG credentials in plaintext config file
- ⚠️ Each user needs PRTG API access

**Mitigations:**
- Use environment variables for secrets
- MCP client config file permissions (user-only read)
- PRTG API tokens with minimal permissions

---

## Advantages of This Approach

### ✅ Keep Go Codebase
- No rewrite needed
- All existing code reused (~99%)
- Only ~350 new lines of code

### ✅ Easy Distribution
- npm package for convenience
- Direct binary download also supported
- Cross-platform (5 platforms bundled)

### ✅ Dual Mode Support
- HTTP mode still works (existing users)
- stdio mode for plugin use case
- Same binary, user choice

### ✅ Fast Implementation
- 3-4 days vs 7-10 days (TypeScript rewrite)
- Lower risk (small changes)
- Easier testing (existing tests reused)

---

## Final Recommendation

### ✅ PROCEED WITH GO stdio IMPLEMENTATION

**Why:**
1. **Keep Go** - No language change needed
2. **Fast** - 3-4 days implementation
3. **Low risk** - Small code changes (~350 lines)
4. **Backward compatible** - HTTP mode still works
5. **Easy distribution** - npm package with Go binaries

**Next Steps:**
1. Create branch `feature/stdio-mode`
2. Implement Phase 1 (stdio protocol)
3. Test with Claude Desktop
4. Create npm package
5. Release v2.0.0 with dual-mode support

**Questions:**
- Start implementation immediately?
- Target release date? (1 week realistic)
- Keep both HTTP and stdio modes in v2.0.0?

---

**Prepared by**: Claude Sonnet 4.5
**Implementation ready**: Yes - all code sketched, just needs typing
