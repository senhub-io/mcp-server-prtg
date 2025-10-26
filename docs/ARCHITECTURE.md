# Architecture Documentation

Technical architecture documentation for MCP Server PRTG.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Components](#components)
  - [SSE v2 Transport Layer](#sse-v2-transport-layer)
  - [MCP Protocol Handler](#mcp-protocol-handler)
  - [Database Layer](#database-layer)
  - [Service Layer](#service-layer)
  - [Security Layer](#security-layer)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Design Decisions](#design-decisions)
- [Performance Considerations](#performance-considerations)
- [Security Architecture](#security-architecture)

## Overview

MCP Server PRTG is a Go-based server that bridges PRTG monitoring data with Large Language Models (LLMs) through the Model Context Protocol (MCP). It uses a dual-server architecture with SSE (Server-Sent Events) transport for real-time bidirectional communication.

### Key Features

- **SSE v2 Transport**: Internal server + authentication proxy architecture
- **MCP Protocol**: Full implementation of Model Context Protocol
- **PostgreSQL Database**: Read-only access to PRTG monitoring data
- **Multi-platform Service**: Windows, Linux, and macOS support via kardianos/service
- **TLS/HTTPS**: Built-in TLS with automatic certificate generation
- **Hot-reload**: Dynamic configuration reloading without restarts
- **Structured Logging**: JSON-based logging with rotation support

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        MCP Client / MCP Client              │
│                                                                 │
│  Uses mcp-proxy to connect via SSE transport                   │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS + Bearer Token
                      │ GET /sse (SSE stream)
                      │ POST /message (JSON-RPC)
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    External Authentication Proxy                │
│                    (Public-facing server)                       │
│                                                                 │
│  • Binds to: 0.0.0.0:8443 (configurable)                       │
│  • TLS/HTTPS enabled by default                                │
│  • Bearer token authentication (RFC 6750)                      │
│  • Reverse proxy to internal SSE server                        │
│                                                                 │
│  Endpoints:                                                     │
│    • GET  /sse      → Proxy to internal SSE (auth required)    │
│    • POST /message  → Proxy to internal SSE (auth required)    │
│    • GET  /health   → Health check (public)                    │
│    • GET  /status   → Server status (auth required)            │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTP (localhost only)
                      │ Reverse proxy
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Internal SSE Server                          │
│                    (Localhost only)                             │
│                                                                 │
│  • Binds to: 127.0.0.1:18443                                   │
│  • No authentication (protected by proxy)                      │
│  • MCP protocol implementation                                 │
│  • Manages SSE connections and message routing                 │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      │ MCP Protocol (JSON-RPC 2.0)
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                       MCP Server Core                           │
│                       (mark3labs/mcp-go)                        │
│                                                                 │
│  • Tool registration and discovery                             │
│  • Request routing and validation                              │
│  • Response formatting                                         │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      │ Tool calls
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Tool Handlers                             │
│                                                                 │
│  • prtg_get_sensors                                            │
│  • prtg_get_sensor_status                                      │
│  • prtg_get_alerts                                             │
│  • prtg_device_overview                                        │
│  • prtg_top_sensors                                            │
│  • prtg_query_sql                                              │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      │ SQL queries
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Database Layer                             │
│                      (jackc/pgx)                                │
│                                                                 │
│  • Connection pooling (25 connections)                         │
│  • Query timeout management (30s)                              │
│  • Result scanning and mapping                                 │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      │ PostgreSQL protocol
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PostgreSQL Database                        │
│                      (PRTG Data Exporter)                       │
│                                                                 │
│  Tables:                                                        │
│    • prtg_sensor                                               │
│    • prtg_device                                               │
│    • prtg_sensor_path                                          │
│    • prtg_device_path                                          │
│    • prtg_tag                                                  │
│    • prtg_sensor_tag                                           │
│    • prtg_group                                                │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### SSE v2 Transport Layer

**File**: `internal/server/sse_server_v2.go`

The SSE v2 architecture uses a dual-server design for security and flexibility:

#### Internal SSE Server
- **Binding**: `127.0.0.1:18443` (localhost only)
- **Purpose**: Handles MCP protocol and SSE connections
- **Security**: No authentication (protected by proxy layer)
- **Framework**: mark3labs/mcp-go SSE server

```go
type SSEServerV2 struct {
    mcpServer    *server.MCPServer      // MCP protocol handler
    sseServer    *server.SSEServer      // Internal SSE server
    proxyServer  *http.Server           // External authentication proxy
    config       *configuration.Configuration
    logger       *logger.ModuleLogger
    db           *database.DB
}
```

#### External Authentication Proxy
- **Binding**: Configurable (default: `0.0.0.0:8443`)
- **Purpose**: TLS termination and authentication
- **Security**: Bearer token validation, TLS encryption
- **Framework**: Go net/http with reverse proxy

**Key Features:**
- Infinite timeouts for SSE long-lived connections
- Custom middleware for Bearer token authentication
- Automatic reverse proxying to internal server
- Health and status endpoints

**Endpoints:**
```go
mux.HandleFunc("/sse", authHandler)        // SSE stream (authenticated)
mux.HandleFunc("/message", authHandler)    // RPC messages (authenticated)
mux.HandleFunc("/health", handleHealth)    // Health check (public)
mux.HandleFunc("/status", authHandler)     // Status (authenticated)
```

#### Connection Flow

1. Client connects to external proxy (`https://server:8443/sse`)
2. Proxy validates Bearer token
3. If valid, proxy forwards to internal server (`http://127.0.0.1:18443/sse`)
4. Internal server establishes SSE connection
5. Bidirectional communication via SSE + POST messages

### MCP Protocol Handler

**Library**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)

The MCP protocol implementation provides:

#### Tool Registration
```go
s.AddTool(mcp.Tool{
    Name: "prtg_get_sensors",
    Description: "Retrieve PRTG sensors...",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "device_name": map[string]string{
                "type": "string",
                "description": "Filter by device name",
            },
            // ... more properties
        },
    },
}, h.handleGetSensors)
```

#### Request Handling
- JSON-RPC 2.0 protocol
- Tool discovery (list available tools)
- Tool invocation with parameter validation
- Error handling and reporting

#### Response Formatting
```go
type CallToolResult struct {
    Content []interface{} // Array of content items
}

type TextContent struct {
    Type string // "text"
    Text string // JSON-formatted result
}
```

### Database Layer

**File**: `internal/database/db.go`, `internal/database/queries.go`
**Driver**: [jackc/pgx/v5](https://github.com/jackc/pgx)

#### Connection Management

```go
type DB struct {
    conn   *sql.DB              // Connection pool
    logger *zerolog.Logger
}

// Connection string format
"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s"
```

**Pool Configuration:**
- Maximum open connections: 25
- Maximum idle connections: 25
- Connection max idle time: 30 minutes
- Connection max lifetime: 1 hour

#### Query Implementation

All queries use parameterized statements to prevent SQL injection:

```go
query := `
    SELECT s.id, s.name, s.status, d.name as device_name
    FROM prtg_sensor s
    INNER JOIN prtg_device d ON s.prtg_device_id = d.id
    WHERE s.status = $1
    LIMIT $2
`
rows, err := db.Query(ctx, query, status, limit)
```

#### Query Timeout

All database operations use context with 30-second timeout:

```go
dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := h.db.GetSensors(dbCtx, ...)
```

#### Result Scanning

Custom scan functions map database rows to Go structs:

```go
func scanSensors(rows *sql.Rows) ([]types.Sensor, error) {
    sensors := []types.Sensor{}
    for rows.Next() {
        var sensor types.Sensor
        // Handle nullable fields with sql.Null* types
        var lastCheckUTC, lastDownUTC sql.NullTime
        var uptimeSecs, downtimeSecs sql.NullFloat64

        err := rows.Scan(&sensor.ID, &lastCheckUTC, ...)
        // Convert nullable types to pointers
        if lastCheckUTC.Valid {
            sensor.LastCheckUTC = &lastCheckUTC.Time
        }
        sensors = append(sensors, sensor)
    }
    return sensors, rows.Err()
}
```

### Service Layer

**File**: `cmd/server/service.go`
**Library**: [kardianos/service](https://github.com/kardianos/service)

Cross-platform service management for Windows, Linux, and macOS.

#### Service Interface

```go
type program struct {
    agent     *agent.Agent
    args      *cliArgs.ParsedArgs
    done      chan bool
    appLogger *logger.Logger
}

func (p *program) Start(service.Service) error {
    // Start agent in background
    go p.run()
    return nil
}

func (p *program) Stop(service.Service) error {
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return p.agent.Shutdown(ctx)
}
```

#### Platform-Specific Configuration

**Windows:**
- Service name: `mcp-server-prtg`
- Runs as Windows Service
- Logs to file

**Linux:**
- Service name: `mcp-server-prtg.service`
- Systemd unit file
- Restart policy: always
- User: root (configurable)

**macOS:**
- LaunchAgent/LaunchDaemon
- Automatic restart on failure

#### Service Operations

```bash
install   # Install service
uninstall # Uninstall service (with cleanup)
start     # Start service
stop      # Stop service
restart   # Restart service
status    # Show detailed status
run       # Run in console mode (interactive)
```

### Security Layer

#### Authentication

**Type**: Bearer Token (RFC 6750)
**Implementation**: Custom HTTP middleware

```go
func (s *SSEServerV2) createAuthMiddleware() func(http.HandlerFunc) http.HandlerFunc {
    expectedToken := s.config.GetAPIKey()

    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // Extract token from Authorization header
            authHeader := r.Header.Get("Authorization")
            const bearerPrefix = "Bearer "
            var providedToken string

            if strings.HasPrefix(authHeader, bearerPrefix) {
                providedToken = authHeader[len(bearerPrefix):]
            }

            // Fallback: query parameter (for SSE compatibility)
            if providedToken == "" {
                providedToken = r.URL.Query().Get("token")
            }

            // Validate token
            if providedToken != expectedToken {
                w.Header().Set("WWW-Authenticate", "Bearer realm=\"MCP Server PRTG\"")
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            next(w, r)
        }
    }
}
```

#### TLS Configuration

```go
tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS12,
    PreferServerCipherSuites: true,
}

// Load certificates
certFile := config.GetTLSCertFile()
keyFile := config.GetTLSKeyFile()

err := server.ListenAndServeTLS(certFile, keyFile)
```

**Self-Signed Certificate Generation:**
- RSA 2048-bit key
- 1-year validity
- Subject Alternative Names (SANs): localhost, 127.0.0.1, ::1
- Automatic generation on first run

#### SQL Injection Prevention

Custom query validation in `prtg_query_sql`:

```go
// 1. Check query starts with SELECT
queryUpper := strings.ToUpper(strings.TrimSpace(query))
if !strings.HasPrefix(queryUpper, "SELECT") {
    return nil, fmt.Errorf("only SELECT queries are allowed")
}

// 2. Block dangerous keywords
dangerous := []string{"DROP", "DELETE", "UPDATE", "INSERT",
                      "ALTER", "CREATE", "TRUNCATE", "EXEC",
                      "EXECUTE", "/*", "--", ";"}
for _, keyword := range dangerous {
    if strings.Contains(queryUpper, keyword) {
        return nil, fmt.Errorf("query contains forbidden keyword: %s", keyword)
    }
}

// 3. Enforce maximum limit
if limit > 1000 {
    limit = 1000
}
```

#### File Permissions

Automatic secure file permissions:
- `config.yaml`: 0600 (owner read/write only)
- `server.key`: 0600 (owner read/write only)
- `server.crt`: 0600 (owner read/write only)
- Log files: 0640 (owner read/write, group read)

## Data Flow

### Tool Call Flow

```
1. Client Request
   │
   ├─→ POST /message
   │   Headers: Authorization: Bearer <token>
   │   Body: {
   │     "jsonrpc": "2.0",
   │     "method": "tools/call",
   │     "params": {
   │       "name": "prtg_get_sensors",
   │       "arguments": {"limit": 10}
   │     }
   │   }
   │
2. Authentication Proxy
   │
   ├─→ Validate Bearer token
   ├─→ Forward to internal server (127.0.0.1:18443)
   │
3. Internal SSE Server
   │
   ├─→ Parse JSON-RPC request
   ├─→ Route to MCP Server
   │
4. MCP Server
   │
   ├─→ Validate tool name
   ├─→ Parse arguments
   ├─→ Call tool handler
   │
5. Tool Handler (e.g., handleGetSensors)
   │
   ├─→ Parse arguments to struct
   ├─→ Validate parameters
   ├─→ Create database context (30s timeout)
   ├─→ Call database layer
   │
6. Database Layer
   │
   ├─→ Build parameterized SQL query
   ├─→ Execute query with pgx
   ├─→ Scan results to Go structs
   ├─→ Return data
   │
7. Tool Handler
   │
   ├─→ Format results as JSON
   ├─→ Create MCP response
   ├─→ Return to MCP Server
   │
8. MCP Server → SSE Server → Proxy → Client
   │
   └─→ Response: {
         "result": {
           "content": [{
             "type": "text",
             "text": "Found 10 results:\n\n[...]"
           }]
         }
       }
```

### Configuration Hot-Reload Flow

```
1. File System Change
   │
   ├─→ fsnotify detects config.yaml modification
   │
2. Configuration Manager
   │
   ├─→ Reload configuration from disk
   ├─→ Parse YAML
   ├─→ Validate new configuration
   │
3. Notify Callbacks
   │
   ├─→ Logger: Update log level
   ├─→ Database: New connections use new settings
   ├─→ Server: API key change (requires client reconnect)
   │
4. Continue Operation
   │
   └─→ No restart required
```

## Technology Stack

### Core Technologies

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | Go 1.21+ | High-performance, concurrent server |
| MCP Protocol | [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | Model Context Protocol implementation |
| Database Driver | [jackc/pgx/v5](https://github.com/jackc/pgx) | PostgreSQL driver with connection pooling |
| Service Management | [kardianos/service](https://github.com/kardianos/service) | Cross-platform service framework |
| Configuration | [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML parsing |
| Logging | [rs/zerolog](https://github.com/rs/zerolog) | Structured JSON logging |
| Log Rotation | [natefinch/lumberjack](https://github.com/natefinch/lumberjack) | Log file rotation |
| File Watching | [fsnotify](https://github.com/fsnotify/fsnotify) | Configuration hot-reload |
| HTTP Server | Go net/http | Built-in HTTP server |
| HTTP Proxy | Go net/http/httputil | Reverse proxy |

### Dependencies

```go
require (
    github.com/fsnotify/fsnotify v1.7.0
    github.com/jackc/pgx/v5 v5.5.0
    github.com/kardianos/service v1.2.2
    github.com/mark3labs/mcp-go v0.6.1
    github.com/rs/zerolog v1.31.0
    gopkg.in/natefinch/lumberjack.v2 v2.2.1
    gopkg.in/yaml.v3 v3.0.1
)
```

## Design Decisions

### Why Dual-Server Architecture (SSE v2)?

**Problem**: SSE connections need infinite timeouts, but we also need request authentication.

**Solution**: Separate internal SSE server from external authentication proxy.

**Benefits:**
- Clean separation of concerns
- Internal server focuses on MCP protocol
- External proxy handles security and TLS
- Easier to test and debug
- Can scale independently

### Why Bearer Token Authentication?

**Alternatives Considered:**
- Basic Auth: Less standard for API access
- OAuth 2.0: Too complex for this use case
- API Key in query: Less secure, harder to manage

**Benefits:**
- RFC 6750 standard
- Widely supported by HTTP clients
- Easy to rotate
- Works with SSE (query parameter fallback)

### Why PostgreSQL with pgx?

**Alternatives Considered:**
- database/sql: Less performant, no native PostgreSQL features
- ORM (GORM, etc.): Unnecessary overhead for read-only queries

**Benefits:**
- Native PostgreSQL driver (fastest)
- Connection pooling built-in
- Type-safe queries
- Excellent error handling
- Active development

### Why Go?

**Alternatives Considered:**
- Python: Slower, GIL limitations
- Node.js: Callback complexity, less type-safe
- Rust: Steeper learning curve

**Benefits:**
- Excellent concurrency (goroutines)
- Fast compilation and execution
- Cross-compilation for multiple platforms
- Strong typing and error handling
- Rich standard library
- Single binary deployment

### Why Read-Only Database Access?

**Security**: Prevents accidental or malicious data modification
**Safety**: No risk of corrupting PRTG data
**Performance**: Read-only user can have restricted permissions

## Performance Considerations

### Connection Pooling

- **25 connections**: Balances resource usage with concurrency
- **Idle timeout (30min)**: Keeps connections fresh
- **Max lifetime (1h)**: Prevents stale connections
- **Connection reuse**: Reduces connection overhead

### Query Optimization

- **Parameterized queries**: Prepared statement benefits
- **Result limits**: Prevents overwhelming responses
- **Indexed columns**: Queries use device/sensor name indexes
- **Selective joins**: Only join tables when necessary

### Memory Management

- **Streaming results**: Large result sets processed row-by-row
- **Context timeouts**: Prevents memory leaks from hanging queries
- **Graceful shutdown**: Proper cleanup of resources
- **Log rotation**: Prevents disk space exhaustion

### Concurrency

- **Goroutines**: Each request handled concurrently
- **Context cancellation**: Proper cleanup on client disconnect
- **Channel-based communication**: Safe concurrent patterns
- **No shared state**: Each request independent

### Caching Strategy

Currently, no caching is implemented. Considerations for future:

**Pros of caching:**
- Reduced database load
- Faster response times
- Better scalability

**Cons of caching:**
- Stale data
- Memory usage
- Cache invalidation complexity

**Decision**: Rely on PostgreSQL query performance and let LLMs request fresh data.

## Security Architecture

### Defense in Depth

1. **Network Layer**: TLS encryption, firewall rules
2. **Application Layer**: Bearer token authentication
3. **Database Layer**: Read-only user, parameterized queries
4. **File System**: Restricted file permissions
5. **Query Layer**: SQL injection prevention

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| Unauthorized access | Bearer token authentication |
| Man-in-the-middle | TLS encryption |
| SQL injection | Parameterized queries, keyword filtering |
| Data modification | Read-only database user |
| Credential exposure | File permissions (0600), password masking in logs |
| DoS attacks | Query timeouts, result limits |
| Certificate theft | Secure file permissions |

### Security Best Practices

1. **Principle of Least Privilege**: Read-only database user
2. **Defense in Depth**: Multiple security layers
3. **Fail Secure**: Errors deny access by default
4. **Security by Design**: Security built into architecture
5. **Logging**: All authentication attempts logged

## See Also

- [Configuration Guide](CONFIGURATION.md)
- [Usage Guide](USAGE.md)
- [Tools Reference](TOOLS.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
- [MCP Protocol Specification](https://modelcontextprotocol.io)
