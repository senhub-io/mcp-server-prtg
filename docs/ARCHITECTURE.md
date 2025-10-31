# Architecture Documentation

Technical architecture documentation for MCP Server PRTG.

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Components](#components)
  - [Streamable HTTP Transport Layer](#streamable-http-transport-layer)
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

MCP Server PRTG is a Go-based server that bridges PRTG monitoring data with Large Language Models (LLMs) through the Model Context Protocol (MCP). It uses **Streamable HTTP transport** (MCP 2025-03-26) for real-time bidirectional communication over HTTP.

### Key Features

- **Streamable HTTP Transport**: Modern MCP protocol with HTTP SSE streaming
- **MCP Protocol**: Full implementation of Model Context Protocol
- **PostgreSQL Database**: Read-only access to PRTG monitoring data
- **Multi-platform Service**: Windows, Linux, and macOS support via kardianos/service
- **TLS/HTTPS**: Built-in TLS with automatic certificate generation
- **Hot-reload**: Dynamic configuration reloading without restarts
- **Structured Logging**: JSON-based logging with rotation support
- **Rate Limiting**: Built-in authentication brute-force protection

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     MCP Client (Claude Desktop, etc.)           │
│                                                                 │
│  Uses mcp-remote to connect via Streamable HTTP transport      │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS + Bearer Token
                      │ POST/GET /mcp (Streamable HTTP + SSE)
                      │ Heartbeat: every 30s
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    StreamableHTTPServer                         │
│                    (Single unified server)                      │
│                                                                 │
│  • Binds to: 0.0.0.0:8443 (configurable)                       │
│  • TLS/HTTPS enabled by default                                │
│  • Bearer token authentication (RFC 6750)                      │
│  • Rate limiting (5 attempts/min, 5 min lockout)              │
│  • Heartbeat mechanism (30s interval)                         │
│                                                                 │
│  Endpoints:                                                     │
│    • POST/GET /mcp     → Streamable HTTP (auth required)      │
│    • GET     /health   → Health check (public)                │
│    • GET     /status   → Server status (auth required)        │
│                                                                 │
│  Features:                                                      │
│    • Single endpoint for all MCP operations                    │
│    • Optimized timeouts for streaming (infinite)              │
│    • Automatic heartbeat to prevent connection drops          │
│    • IP-based rate limiting for security                      │
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
│  • Streamable HTTP protocol implementation                    │
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
│  • prtg_query_sql (if enabled)                                │
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

### Streamable HTTP Transport Layer

**File**: `internal/server/streamable_http_server.go`

The Streamable HTTP architecture uses a **single unified server** design (much simpler than the deprecated SSE v2 dual-server approach):

#### StreamableHTTPServer
- **Binding**: Configurable (default: `0.0.0.0:8443`)
- **Purpose**: Handles MCP protocol via Streamable HTTP transport
- **Security**: Bearer token authentication with rate limiting, TLS encryption
- **Framework**: mark3labs/mcp-go Streamable HTTP server

```go
type StreamableHTTPServer struct {
    mcpServer      *server.MCPServer      // MCP protocol handler
    streamableHTTP http.Handler           // Streamable HTTP handler
    httpServer     *http.Server           // HTTP server
    config         *configuration.Configuration
    logger         *logger.ModuleLogger
    db             *database.DB
    rateLimiter    *authRateLimiter      // Rate limiting for auth
    address        string
    shutdownCh     chan struct{}
}
```

#### Key Features

**Streamable HTTP Protocol:**
- **Single endpoint**: `/mcp` handles all MCP operations
- **Bidirectional streaming**: Uses HTTP SSE for server-to-client messages
- **Heartbeat mechanism**: Automatic 30-second heartbeat to prevent timeouts
- **Simpler than SSE v2**: No separate internal server or proxy needed

**Optimized Timeouts:**
```go
s.httpServer = &http.Server{
    ReadTimeout:       0,                // No read timeout for streaming
    WriteTimeout:      0,                // No write timeout for streaming
    IdleTimeout:       60 * time.Minute, // Close inactive connections after 1 hour
    ReadHeaderTimeout: 10 * time.Second, // Protection against slow-loris attacks
    MaxHeaderBytes:    1 << 20,          // 1MB max header size
}
```

**Endpoints:**
```go
mux.Handle("/mcp", s.createAuthMiddleware(s.streamableHTTP))      // MCP endpoint (authenticated)
mux.HandleFunc("/health", s.handleHealth)                          // Health check (public)
mux.Handle("/status", s.createAuthMiddleware(...))                 // Status (authenticated)
```

#### Connection Flow

1. Client connects to server (`https://server:8443/mcp`)
2. Server validates Bearer token (with rate limiting)
3. Server establishes Streamable HTTP connection
4. Bidirectional communication via HTTP + SSE streaming
5. Heartbeat sent every 30 seconds to keep connection alive

**Benefits over SSE v2:**
- Simpler architecture (one server instead of two)
- Standards-compliant MCP protocol
- Built-in heartbeat mechanism
- Better connection stability
- Easier to deploy and configure

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

#### Authentication with Rate Limiting

**Type**: Bearer Token (RFC 6750) with IP-based rate limiting
**Implementation**: Custom HTTP middleware

```go
func (s *StreamableHTTPServer) createAuthMiddleware(next http.Handler) http.Handler {
    expectedToken := s.config.GetAPIKey()

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract client IP for rate limiting
        clientIP := getClientIP(r)

        // Check rate limit BEFORE validating token (prevent brute-force)
        if !s.rateLimiter.checkAndRecord(clientIP, false) {
            s.logger.Warn().
                Str("client_ip", clientIP).
                Msg("Rate limit exceeded - IP temporarily blocked")

            w.Header().Set("Retry-After", "300") // 5 minutes
            http.Error(w, "Too many authentication attempts", http.StatusTooManyRequests)
            return
        }

        // Extract Bearer token from Authorization header
        authHeader := r.Header.Get("Authorization")
        const bearerPrefix = "Bearer "
        var providedToken string

        if strings.HasPrefix(authHeader, bearerPrefix) {
            providedToken = authHeader[len(bearerPrefix):]
        }

        // Fallback: query parameter (for compatibility)
        if providedToken == "" {
            providedToken = r.URL.Query().Get("token")
        }

        // Validate token
        if providedToken != expectedToken {
            w.Header().Set("WWW-Authenticate", "Bearer realm=\"MCP Server PRTG\"")
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Reset rate limiter on successful auth
        s.rateLimiter.checkAndRecord(clientIP, true)
        next.ServeHTTP(w, r)
    })
}
```

**Rate Limiting Configuration:**
- **Max attempts**: 5 failed attempts
- **Time window**: 1 minute
- **Lockout duration**: 5 minutes
- **Scope**: Per client IP address

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
   ├─→ POST/GET /mcp
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
2. StreamableHTTPServer
   │
   ├─→ Check rate limit for client IP
   ├─→ Validate Bearer token
   ├─→ Forward to Streamable HTTP handler
   │
3. Streamable HTTP Handler
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
8. MCP Server → Streamable HTTP → Client
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

### Heartbeat Mechanism

```
1. Connection Established
   │
   ├─→ Client connects to /mcp
   ├─→ Streamable HTTP connection established
   │
2. Background Heartbeat
   │
   ├─→ Server sends heartbeat every 30 seconds
   ├─→ Prevents connection timeout
   ├─→ Keeps connection alive
   │
3. Connection Monitoring
   │
   ├─→ If heartbeat fails, connection is closed
   ├─→ Client reconnects automatically
   │
   └─→ Ensures reliable long-lived connections
```

## Technology Stack

### Core Technologies

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | Go 1.25+ | High-performance, concurrent server |
| MCP Protocol | [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | Model Context Protocol implementation |
| Database Driver | [jackc/pgx/v5](https://github.com/jackc/pgx) | PostgreSQL driver with connection pooling |
| Service Management | [kardianos/service](https://github.com/kardianos/service) | Cross-platform service framework |
| Configuration | [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) | YAML parsing |
| Logging | [rs/zerolog](https://github.com/rs/zerolog) | Structured JSON logging |
| Log Rotation | [natefinch/lumberjack](https://github.com/natefinch/lumberjack) | Log file rotation |
| File Watching | [fsnotify](https://github.com/fsnotify/fsnotify) | Configuration hot-reload |
| HTTP Server | Go net/http | Built-in HTTP server |

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

### Why Streamable HTTP Transport?

**Migration from SSE v2:**
The server was migrated from SSE v2 (dual-server architecture) to Streamable HTTP for several reasons:

**Problems with SSE v2:**
- Complex dual-server architecture (internal + proxy)
- Connection stability issues
- Protocol deprecated as of MCP 2024-11-05

**Benefits of Streamable HTTP:**
- **Standards-compliant**: Official MCP 2025-03-26 transport
- **Simpler architecture**: Single unified server
- **Better stability**: Built-in heartbeat mechanism
- **Single endpoint**: `/mcp` instead of `/sse` + `/message`
- **Easier deployment**: No internal proxy configuration
- **Future-proof**: Modern MCP protocol standard

### Why Bearer Token Authentication?

**Alternatives Considered:**
- Basic Auth: Less standard for API access
- OAuth 2.0: Too complex for this use case
- API Key in query: Less secure, harder to manage

**Benefits:**
- RFC 6750 standard
- Widely supported by HTTP clients
- Easy to rotate
- Works with Streamable HTTP protocol
- Compatible with mcp-remote client

### Why Add Rate Limiting?

**Security Enhancement:**
- Prevents brute-force authentication attacks
- Per-IP tracking and lockout
- Automatic cleanup of old entries
- No impact on legitimate users

**Configuration:**
- 5 attempts per minute per IP
- 5-minute lockout after max attempts
- Automatic recovery after lockout period

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

### Heartbeat Mechanism

**Purpose**: Prevents connection timeouts for long-lived connections

```go
heartbeatInterval := 30 * time.Second
heartbeatOption := server.WithHeartbeatInterval(heartbeatInterval)
s.streamableHTTP = server.NewStreamableHTTPServer(s.mcpServer, heartbeatOption)
```

This sends periodic heartbeats to keep connections alive, addressing the timeout issues experienced with earlier implementations.

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
2. **Application Layer**: Bearer token authentication, rate limiting
3. **Database Layer**: Read-only user, parameterized queries
4. **File System**: Restricted file permissions
5. **Query Layer**: SQL injection prevention

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| Unauthorized access | Bearer token authentication |
| Brute-force attacks | IP-based rate limiting (5 attempts/min, 5 min lockout) |
| Man-in-the-middle | TLS encryption |
| SQL injection | Parameterized queries, keyword filtering |
| Data modification | Read-only database user |
| Credential exposure | File permissions (0600), password masking in logs |
| DoS attacks | Query timeouts, result limits, rate limiting |
| Certificate theft | Secure file permissions |
| Slow-loris attacks | ReadHeaderTimeout protection |

### Security Best Practices

1. **Principle of Least Privilege**: Read-only database user
2. **Defense in Depth**: Multiple security layers
3. **Fail Secure**: Errors deny access by default
4. **Security by Design**: Security built into architecture
5. **Logging**: All authentication attempts logged
6. **Rate Limiting**: Automatic protection against brute-force

## See Also

- [Configuration Guide](CONFIGURATION.md)
- [Usage Guide](USAGE.md)
- [Tools Reference](TOOLS.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
- [MCP Protocol Specification](https://modelcontextprotocol.io)
