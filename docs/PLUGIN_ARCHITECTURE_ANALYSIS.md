# MCP Server PRTG - Plugin Architecture Analysis

**Date**: 2025-12-30
**Target**: Migrate from HTTP Remote Server to Local stdio Plugin
**Version**: 1.3.0-beta.1 → 2.0.0 (plugin-based)

## Executive Summary

**Recommendation**: ✅ **PLUGIN ARCHITECTURE IS IDEAL**

Migrating to a local stdio plugin offers significant advantages:
- ✅ **Simpler deployment**: Users install via npm/binary, no server to host
- ✅ **Better security**: Credentials stored locally, no central attack surface
- ✅ **Standard MCP pattern**: Follows official MCP server conventions
- ✅ **Easier distribution**: npm package or standalone binary
- ✅ **Native client integration**: Works with Claude Desktop, Cursor, Continue, Cline

### Trade-offs vs HTTP Remote Server

**Gains:**
- ✅ No server infrastructure needed (no hosting, monitoring, maintenance)
- ✅ No network configuration (no firewall rules, TLS certs, public IPs)
- ✅ Simpler user setup (one config file vs server deployment)
- ✅ Per-user credentials (no shared authentication token)
- ✅ Lower latency (local process vs network round-trip)

**Losses:**
- ❌ No centralized access control
- ❌ No shared caching across users (each client caches independently)
- ❌ Each user needs direct PRTG API access (firewall considerations)
- ❌ Credentials duplicated per client (vs single server credential)

**Verdict**: Plugin architecture is **superior for most use cases**, especially for:
- Individual developers/operators
- Small teams (< 20 users)
- Organizations with existing PRTG API access for users

---

## MCP Protocol: stdio vs HTTP

### Current: HTTP Remote (Streamable HTTP Transport)

**Architecture:**
```
Client → HTTP(S) → Remote Server → PRTG API
         (port 8443, SSE streaming)
```

**Pros:**
- Centralized deployment
- Shared cache
- Single credential management

**Cons:**
- Infrastructure overhead (hosting, TLS, firewall)
- Network latency
- Single point of failure
- Complex setup (systemd/Windows service)

### Proposed: stdio Local (Standard MCP Transport)

**Architecture:**
```
Client → stdio (stdin/stdout) → Local Plugin Process → PRTG API
         (parent-child process)
```

**Pros:**
- Zero infrastructure
- Simple installation (npm/binary)
- Fast (no network overhead)
- Standard MCP pattern

**Cons:**
- No centralized control
- Per-client PRTG API access required

---

## Implementation Options

### Option 1: Node.js Package (Recommended)

**Distribution:**
```bash
npm install -g @senhub-io/mcp-server-prtg
# or
npx -y @senhub-io/mcp-server-prtg
```

**Configuration:**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-token-here",
        "PRTG_VERIFY_SSL": "true",
        "CACHE_TTL_SECONDS": "60"
      }
    }
  }
}
```

**Technology Stack:**
- **Language**: TypeScript (Node.js 18+)
- **MCP SDK**: `@modelcontextprotocol/sdk` (official)
- **HTTP Client**: `axios` or `node-fetch`
- **Protocol**: stdio (stdin/stdout JSON-RPC)

**Pros:**
- ✅ Easy distribution (npm registry)
- ✅ Official MCP SDK support (TypeScript)
- ✅ Fast development (existing TypeScript ecosystem)
- ✅ Auto-updates via npm
- ✅ Cross-platform (macOS, Linux, Windows)

**Cons:**
- ❌ Requires Node.js runtime installed
- ❌ Larger footprint than compiled binary

**Effort**: 3-5 days (rewrite from Go to TypeScript)

---

### Option 2: Standalone Binary (Go)

**Distribution:**
```bash
# Download binary
curl -L https://github.com/senhub-io/mcp-server-prtg/releases/download/v2.0.0/mcp-server-prtg-darwin-arm64 -o mcp-server-prtg
chmod +x mcp-server-prtg
```

**Configuration:**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "/usr/local/bin/mcp-server-prtg",
      "args": ["stdio"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-token-here"
      }
    }
  }
}
```

**Technology Stack:**
- **Language**: Go (keep current codebase)
- **MCP Protocol**: Custom JSON-RPC implementation over stdio
- **HTTP Client**: `net/http` (existing `internal/prtg/client.go`)

**Pros:**
- ✅ No runtime dependency (single binary)
- ✅ Smaller footprint (~5MB vs Node.js ~50MB)
- ✅ Keep existing Go codebase
- ✅ Fast startup time

**Cons:**
- ❌ No official MCP SDK for Go (must implement JSON-RPC manually)
- ❌ More complex distribution (5 binaries for cross-platform)
- ❌ Manual updates (no npm auto-update)

**Effort**: 2-3 days (adapt existing code to stdio)

---

### Option 3: Hybrid (npm + Go Binary)

**Distribution:**
```bash
npm install -g @senhub-io/mcp-server-prtg
# npm package bundles Go binary
```

**How it works:**
1. npm package contains platform-specific Go binaries
2. Node.js wrapper selects correct binary for OS/arch
3. Spawns Go binary with stdio communication

**Pros:**
- ✅ Easy installation (npm)
- ✅ Fast execution (Go binary)
- ✅ Keep existing codebase
- ✅ Auto-updates via npm

**Cons:**
- ⚠️ Larger npm package (~20MB with all binaries)
- ⚠️ More complex build pipeline

**Effort**: 4-6 days (npm wrapper + stdio adaptation)

---

## Recommended Approach

### ✅ **Option 1: Node.js Package**

**Rationale:**
1. **Official MCP SDK**: TypeScript SDK is mature and well-documented
2. **Ecosystem fit**: Most MCP servers are Node.js (filesystem, github, postgres)
3. **Easy distribution**: npm registry standard for MCP servers
4. **Faster development**: Rewrite is simpler than custom JSON-RPC in Go
5. **Better maintenance**: TypeScript ecosystem for web APIs

**Example Official MCP Servers (all Node.js):**
- `@modelcontextprotocol/server-filesystem`
- `@modelcontextprotocol/server-github`
- `@modelcontextprotocol/server-postgres`
- `@modelcontextprotocol/server-brave-search`

---

## Migration Architecture

### Current (v1.3.0-beta.1): HTTP Remote Server

```
┌────────────────────────────────────────────────┐
│  MCP Client (Claude Desktop)                  │
└───────────────────┬────────────────────────────┘
                    │ HTTPS + Bearer Auth
                    │ POST /mcp (SSE streaming)
                    ▼
┌────────────────────────────────────────────────┐
│  MCP Server PRTG (Go binary)                  │
│  - HTTP Server (port 8443)                    │
│  - TLS Certificate Management                 │
│  - Bearer Token Authentication                │
│  - Windows Service / systemd                  │
│  └────────────────┬───────────────────────────┤
│                   │                            │
│  ┌────────────────▼───────────┐               │
│  │ Database Layer (optional)  │               │
│  │ PostgreSQL Connection Pool │               │
│  └────────────────────────────┘               │
└────────────────────┬───────────────────────────┘
                     │ HTTPS
                     ▼
┌────────────────────────────────────────────────┐
│  PRTG API v2 (port 1616)                      │
└────────────────────────────────────────────────┘
```

**Components:**
- HTTP Server with SSE streaming
- TLS certificate generation
- Bearer token authentication
- Service management (systemd/Windows Service)
- Database connection pooling (if using PostgreSQL)
- Configuration file management
- Logging infrastructure

**Complexity**: High (9 major components)

---

### Proposed (v2.0.0): stdio Local Plugin

```
┌────────────────────────────────────────────────┐
│  MCP Client (Claude Desktop, Cursor, etc.)    │
│  ┌──────────────────────────────────────────┐ │
│  │ Built-in MCP Client                      │ │
│  └──────────────┬───────────────────────────┘ │
│                 │ stdio (JSON-RPC)             │
│  ┌──────────────▼───────────────────────────┐ │
│  │ @senhub-io/mcp-server-prtg               │ │
│  │ (Node.js process, spawned by parent)     │ │
│  │ - MCP Protocol Handler                   │ │
│  │ - Tool Implementations                   │ │
│  │ - PRTG API Client                        │ │
│  │ - Local Cache (in-memory)                │ │
│  └──────────────┬───────────────────────────┘ │
└─────────────────┼───────────────────────────────┘
                  │ HTTPS (from user's machine)
                  ▼
┌────────────────────────────────────────────────┐
│  PRTG API v2 (port 1616)                      │
└────────────────────────────────────────────────┘
```

**Components:**
- MCP protocol handler (stdio JSON-RPC)
- Tool implementations (15 tools)
- PRTG API client (HTTP requests)
- In-memory cache (optional)

**Complexity**: Low (4 major components)

**Removed:**
- ❌ HTTP server
- ❌ TLS management
- ❌ Authentication layer
- ❌ Service management
- ❌ Database layer

---

## Code Structure Comparison

### Current (Go, HTTP Remote)

```
mcp-server-prtg/
├── cmd/
│   └── server/
│       └── main.go                    # Entry point, service management
├── internal/
│   ├── server/
│   │   ├── streamable_http_server.go  # HTTP + SSE server
│   │   └── routes.go                  # HTTP routing
│   ├── handlers/
│   │   ├── tools.go                   # MCP tool handlers
│   │   ├── tools_metrics.go           # API v2 tools
│   │   ├── formatting.go              # Response formatting
│   │   ├── visualization.go           # ASCII visualizations
│   │   └── errors.go                  # Error handling
│   ├── database/                      # PostgreSQL layer (optional)
│   │   ├── queries.go
│   │   └── connection.go
│   ├── prtg/
│   │   ├── client.go                  # PRTG API client
│   │   └── types.go                   # Response types
│   ├── services/
│   │   ├── configuration/             # Config management
│   │   └── auth/                      # Bearer token auth
│   └── types/                         # Shared types
├── docs/                              # Documentation
└── Makefile                           # Build automation
```

**Lines of Code**: ~15,000 (including tests)

---

### Proposed (TypeScript, stdio Plugin)

```
mcp-server-prtg/
├── src/
│   ├── index.ts                       # Entry point (stdio handler)
│   ├── server.ts                      # MCP Server instance
│   ├── tools/
│   │   ├── sensors.ts                 # Sensor-related tools
│   │   ├── devices.ts                 # Device tools
│   │   ├── groups.ts                  # Group tools
│   │   ├── metrics.ts                 # Time series tools
│   │   └── index.ts                   # Tool registry
│   ├── prtg/
│   │   ├── client.ts                  # PRTG API client
│   │   ├── types.ts                   # TypeScript interfaces
│   │   └── cache.ts                   # In-memory cache
│   ├── formatters/
│   │   ├── responses.ts               # Response formatting
│   │   └── visualizations.ts          # ASCII visualizations
│   └── utils/
│       ├── errors.ts                  # Error handling
│       └── config.ts                  # Environment config
├── tests/                             # Jest tests
├── docs/                              # Documentation
├── package.json                       # npm package config
└── tsconfig.json                      # TypeScript config
```

**Estimated Lines of Code**: ~5,000-7,000 (much simpler)

**Key Simplifications:**
- No HTTP/TLS layer (~2,000 lines removed)
- No service management (~1,000 lines removed)
- No database layer (~3,000 lines removed if migrating to API v2)
- Official MCP SDK handles protocol (~500 lines we don't write)

---

## Implementation Plan

### Phase 1: Project Setup (1 day)

**Tasks:**
1. Create new repository structure for Node.js/TypeScript
2. Set up build pipeline (TypeScript, esbuild/webpack)
3. Configure MCP SDK (`@modelcontextprotocol/sdk`)
4. Set up testing framework (Jest)
5. Create npm package configuration

**Deliverables:**
- ✅ Working stdio "hello world" MCP server
- ✅ Build pipeline functional
- ✅ Test framework configured

---

### Phase 2: PRTG API Client (2-3 days)

**Tasks:**
1. Port `internal/prtg/client.go` to TypeScript
2. Implement all API v2 endpoints:
   - `/experimental/sensors`
   - `/experimental/devices`
   - `/experimental/groups`
   - `/experimental/objects`
   - `/experimental/channels`
   - `/experimental/timeseries`
   - `/sensor-status-summary`
   - `/objects/count`
3. Add request caching (simple in-memory LRU)
4. Add error handling and retries
5. Unit tests for client

**Deliverables:**
- ✅ Complete PRTG API v2 TypeScript client
- ✅ 100% API coverage
- ✅ Unit tests passing

---

### Phase 3: MCP Tools Implementation (3-4 days)

**Tasks:**
1. Implement all 15 MCP tools:
   - **Sensors**: get_sensors, get_sensor_status, get_alerts
   - **Devices**: device_overview
   - **Groups**: get_groups, get_hierarchy
   - **Search**: search, top_sensors
   - **Tags**: get_tags, get_business_processes
   - **Stats**: get_statistics
   - **Metrics**: get_channel_current_values, get_sensor_timeseries, get_sensor_history_custom
2. Port response formatters:
   - ASCII visualizations (sparklines, trends)
   - Contextual suggestions
   - Pedagogical error messages
3. Tool input validation and error handling
4. Integration tests

**Deliverables:**
- ✅ All 15 tools functional
- ✅ Response formatting preserved
- ✅ Tests passing

---

### Phase 4: Documentation & Publishing (1-2 days)

**Tasks:**
1. Update README.md for npm package
2. Create installation guide for stdio usage
3. Update configuration examples
4. Create migration guide from v1.x
5. Publish to npm registry (`@senhub-io/mcp-server-prtg`)
6. Create GitHub release

**Deliverables:**
- ✅ Complete documentation
- ✅ npm package published
- ✅ GitHub release v2.0.0

---

## Timeline

**Total Duration**: 7-10 days

| Phase | Duration | Effort |
|-------|----------|--------|
| Phase 1: Setup | 1 day | 8 hours |
| Phase 2: PRTG Client | 2-3 days | 16-24 hours |
| Phase 3: Tools | 3-4 days | 24-32 hours |
| Phase 4: Docs | 1-2 days | 8-16 hours |
| **Total** | **7-10 days** | **56-80 hours** |

**Accelerated**: Focus on core tools only → 5-7 days

---

## Configuration Migration

### Old (v1.x): Remote Server config.yaml

```yaml
version: 1
server:
  api_key: "server-auth-token"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
  cert_file: "./certs/cert.pem"
  key_file: "./certs/key.pem"
database:  # Optional
  host: localhost
  port: 5432
  name: prtg_data_exporter
  user: prtg_reader
  password: secret
prtg:
  base_url: "https://prtg.example.com:1616"
  api_token: "prtg-api-token"
  timeout: 30
  verify_ssl: true
```

### New (v2.0): stdio Plugin Environment Variables

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
        "PRTG_TIMEOUT": "30",
        "CACHE_TTL": "60",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

**Simplification**:
- 15 config params → 6 env vars
- No server management config
- No TLS config
- No database config (using API v2 only)

---

## Security Considerations

### Old (Remote Server)

**Threats:**
- Central server is attack target
- Bearer token compromise = all users affected
- TLS certificate management
- Network exposure (firewall rules)

**Mitigations:**
- Strong bearer token
- TLS encryption
- Firewall restrictions
- Token rotation

### New (Local Plugin)

**Threats:**
- PRTG credentials stored in client config
- Per-user PRTG API access required
- No centralized access control

**Mitigations:**
- Credentials in user's config file (OS-level permissions)
- Use MCP client's secure storage (if available)
- PRTG API token per user (granular permissions)
- No network exposure of MCP server

**Conclusion**: Local plugin is **more secure** (smaller attack surface, no central server)

---

## Distribution Comparison

### Old (v1.x): Binary Distribution

**Process:**
1. Download platform-specific binary
2. Create config.yaml
3. Install as system service
4. Configure firewall
5. Generate TLS certificates
6. Start service
7. Configure MCP client to connect via HTTP

**Complexity**: High (7 steps, platform-specific)

### New (v2.0): npm Package

**Process:**
1. Add to MCP client config
2. Set environment variables (PRTG URL + token)

**Complexity**: Low (2 steps, platform-agnostic)

**Example Installation:**

```bash
# Option 1: npx (no install)
# Just add to claude_desktop_config.json:
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-token"
      }
    }
  }
}

# Option 2: Global install
npm install -g @senhub-io/mcp-server-prtg

# Then use in config:
{
  "mcpServers": {
    "prtg": {
      "command": "mcp-server-prtg",
      "env": { ... }
    }
  }
}
```

---

## Recommendation

### ✅ PROCEED WITH PLUGIN ARCHITECTURE (Option 1: Node.js)

**Rationale:**
1. **Simpler deployment**: npm install vs server infrastructure
2. **Standard pattern**: Aligns with official MCP servers
3. **Better security**: No central attack surface
4. **Easier maintenance**: TypeScript + MCP SDK
5. **Faster development**: Rewrite is straightforward

**Proposed Timeline:**
- Week 1: Setup + PRTG client
- Week 2: Tools implementation + testing
- Release: v2.0.0 as stdio-based npm package

**Breaking Changes:**
- Architecture change (HTTP → stdio)
- No backward compatibility with v1.x config
- Users must reconfigure MCP clients

**Migration Path for Users:**
1. Uninstall/stop v1.x server
2. Install v2.0 via npm
3. Update MCP client config (new format)
4. Restart MCP client

---

## Next Steps

**If approved:**

1. **Immediate**: Create new repo or branch for TypeScript rewrite
2. **Day 1-2**: Project setup + PRTG client skeleton
3. **Day 3-5**: Implement core tools
4. **Day 6-7**: Documentation + npm publish
5. **Release**: v2.0.0-beta.1 for community testing

**Questions for decision:**
- Proceed with Node.js/TypeScript (recommended) or keep Go binary?
- Target release date? (2 weeks realistic)
- Keep v1.x available for users who need HTTP remote?
- Package name: `@senhub-io/mcp-server-prtg` or different namespace?

---

**Prepared by**: Claude Sonnet 4.5
**Review required**: Architecture approval + technology stack confirmation
