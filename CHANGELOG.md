# Changelog

All notable changes to this project are documented in this file.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Security
- HTTP mode: an empty `api_key` no longer authenticates requests; the server refuses to start with a key shorter than 16 characters.
- HTTP mode: the API key is compared in constant time and accepted only from the `Authorization: Bearer` header. **Breaking:** the `?token=` query parameter is no longer accepted.
- HTTP mode: the authentication rate limiter keys on the TCP peer address (IPv6 by /64) instead of client-supplied `X-Forwarded-For` / `X-Real-IP` headers, and locked addresses are never evicted.
- `prtg_query_sql` runs in a read-only transaction with a 15 s statement timeout and an enforced row limit, and rejects multiple statements, comments, PRTG Data Exporter credential tables and dangerous functions.
- PRTG API v2 filters drop quotes, backslashes and control characters from user input, so a name cannot rewrite the filter expression.
- PRTG API v2 responses are capped at 64 MiB and logged bodies are truncated.
- Log masking now covers UUID-shaped keys.
- `uninstall` no longer deletes a custom log directory recursively.
- Go 1.26.11 toolchain: fixes 27 standard-library vulnerabilities reported by govulncheck.

### Added
- Windows MSI installer with in-place upgrades; configuration lives in `%ProgramData%\Sensor Factory\MCP Server PRTG` and survives upgrades and uninstall. See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).
- PRTG API v2 request guard: TTL cache, coalescing of identical concurrent requests and a concurrency limit (`PRTG_MAX_CONCURRENT`, default 2) to protect the PRTG core server. `PRTG_CACHE_DISABLED` turns the cache off.
- stdio mode: 11 PRTG API v2 tools replacing the database tools (sensors, alerts, device overview, hierarchy, search, groups, tags, business processes, top sensors, statistics).
- Deployment and update procedure: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

### Changed
- Generated TLS certificates are written next to the configuration file instead of the executable directory.
- Dependencies upgraded, including `mark3labs/mcp-go` 1.6.
- `.golangci.yml` migrated to the golangci-lint v2 format.

### Fixed
- `QueryRow` cancelled its context before callers scanned the row, failing some database tools intermittently.
- `ILIKE` filters escape `%`, `_` and `\` in user input.
- Listener and TLS errors now fail startup instead of leaving a service that reports itself started.
- CI release workflows used action versions that no longer run on GitHub runners.

### Removed
- `prtg_get_sensor_history_custom`: the PRTG API v2 endpoint does not support custom ranges.

## [2.0.0-alpha.1] - 2025-12-30

### Added

#### stdio Transport Mode (Breaking Change: New Primary Deployment Model)
- **stdio Protocol Support**: Server now runs as a local child process communicating via stdin/stdout
- **Dual-Mode Architecture**: HTTP server mode (v1.x) + stdio mode (v2.0+) running side-by-side
- **New Command**: `mcp-server-prtg stdio` - Launches server in stdio mode for MCP client plugins
- **Zero-Config Plugin Usage**: No server infrastructure needed (no TLS certs, no firewall rules, no systemd)
- **Environment-Based Configuration**: Configuration via environment variables (replaces config.yaml in stdio mode)
  - `PRTG_URL`: PRTG server URL with API port
  - `PRTG_API_TOKEN`: PRTG API v2 Bearer token
  - `PRTG_VERIFY_SSL`: SSL certificate verification (default: true)
  - `PRTG_TIMEOUT`: HTTP timeout in seconds (default: 30)
  - `LOG_LEVEL`: Logging level (default: info)

#### npm Package Distribution
- **Package Name**: `@senhub-io/mcp-server-prtg` (scoped to SenHub.io organization)
- **Cross-Platform Binary Wrapper**: Automatic platform/architecture detection and binary selection
- **Supported Platforms**:
  - macOS: x64 (Intel), ARM64 (Apple Silicon M1/M2/M3)
  - Linux: x64, ARM64
  - Windows: x64
- **Installation Methods**:
  - `npx -y @senhub-io/mcp-server-prtg` (no installation, always latest)
  - `npm install -g @senhub-io/mcp-server-prtg` (global installation)
- **Node.js Wrapper**: `npm-package/bin/index.js` (87 lines) - Platform detection and process spawning
- **Smart Binary Loading**: Maps Node.js platform/arch to Go binary names automatically
- **Graceful Signal Handling**: SIGINT/SIGTERM forwarding for clean shutdowns

#### MCP Client Plugin Integration
- **Direct Claude Desktop Support**: Ready-to-use npx configuration
- **Continue.dev Compatibility**: VS Code & JetBrains extension support
- **Cursor Support**: AI-first code editor integration
- **Cline Support**: Autonomous coding agent for VS Code
- **Local-First Architecture**: No remote servers, all processing local
- **Secure by Design**: No network exposure, environment-scoped credentials

#### Documentation
- **npm Package README**: Complete installation, configuration, and usage guide (201 lines)
- **Implementation Analysis**: `docs/GO_STDIO_IMPLEMENTATION.md` (870 lines)
  - stdio architecture and design decisions
  - Environment variable configuration patterns
  - Logging strategy (stderr for logs, stdout for MCP protocol)
  - Error handling and connectivity checks
- **Migration Analysis**: `docs/API_V2_MIGRATION_ANALYSIS.md` (389 lines)
  - Path from database-dependent tools to API v2-only tools
  - stdio mode limitations and workarounds
- **Plugin Architecture**: `docs/PLUGIN_ARCHITECTURE_ANALYSIS.md` (677 lines)
  - MCP plugin ecosystem analysis
  - stdio vs HTTP transport comparison
  - Multi-client deployment strategies

### Changed

#### Breaking Changes (MAJOR version bump)
- **Primary Deployment Model**: stdio mode is now the recommended deployment (HTTP mode still supported)
- **Tool Availability**: In stdio mode, only PRTG API v2 tools work (3 metrics tools)
  - ✅ Available: `prtg_get_channel_current_values`, `prtg_get_sensor_timeseries`, `prtg_get_sensor_history_custom`
  - ⚠️ Database tools require HTTP mode: 12 PostgreSQL-based tools (sensors, alerts, hierarchy, etc.)
- **Configuration Paradigm**: Environment variables (stdio) vs YAML file (HTTP)
- **No Database in stdio Mode**: Database connection not available in stdio mode (limitation by design)

#### Architecture Evolution
- **Transport Layer**: Added stdio support alongside existing HTTP/SSE transport
- **Command Structure**: New `stdio` command joins existing `run`, `install`, `uninstall`, `start`, `stop`, `status`
- **Logging Strategy**: stderr for application logs (stdio mode), stdout reserved for MCP JSON-RPC protocol
- **Process Model**: Child process (stdio) vs daemon/service (HTTP)

### Fixed
- **PRTG Connectivity Check**: Non-blocking ping on startup (warns but doesn't fail if PRTG unreachable)
- **Signal Handling**: Proper SIGINT/SIGTERM handling in stdio mode for graceful shutdowns

### Documentation
- **npm-package/README.md**: Complete npm package documentation (+201 lines)
- **docs/GO_STDIO_IMPLEMENTATION.md**: Technical implementation guide (+870 lines)
- **docs/API_V2_MIGRATION_ANALYSIS.md**: Migration roadmap (+389 lines)
- **docs/PLUGIN_ARCHITECTURE_ANALYSIS.md**: Plugin ecosystem overview (+677 lines)

### Technical Details

#### New Files
- `cmd/server/stdio.go` (203 lines): stdio mode implementation
  - `runStdioMode()`: Main stdio server entry point
  - `loadConfigFromEnv()`: Environment variable configuration loader
  - `Config` struct: Environment-based configuration
  - Helper functions: `getEnv()`, `getEnvInt()`, `getEnvBool()`
- `npm-package/package.json` (48 lines): npm package manifest
- `npm-package/README.md` (201 lines): npm package documentation
- `npm-package/bin/index.js` (87 lines): Cross-platform binary wrapper
- `npm-package/bin/.gitignore`: Ignore platform-specific binaries
- `docs/GO_STDIO_IMPLEMENTATION.md` (870 lines)
- `docs/API_V2_MIGRATION_ANALYSIS.md` (389 lines)
- `docs/PLUGIN_ARCHITECTURE_ANALYSIS.md` (677 lines)

#### Modified Files
- `cmd/server/main.go` (+5 lines):
  - Added `cmdStdio = "stdio"` constant
  - Added stdio case to command switch

#### Binary Builds
- **Platforms**: darwin (x64, ARM64), linux (x64, ARM64), windows (x64)
- **Binaries Built**: 5 platform binaries in `build/` directory
- **Total Size**: ~55 MB (uncompressed), ~22 MB (ZIP archives)

#### Commits Included
- `b614964`: feat: add stdio mode for MCP client plugin usage

---

### Migration Path from v1.x

#### For New Users (Recommended: stdio mode)
```bash
# No installation needed
# Add to Claude Desktop config with npx
```

#### For Existing v1.x Users (HTTP mode)
- **No action required**: HTTP server mode still fully supported
- **Optional migration**: Switch to stdio mode for simpler deployment
- **Considerations**:
  - stdio mode: 3 API v2 tools only (metrics)
  - HTTP mode: 15 tools (12 database + 3 API v2)

---

### Alpha Release Notes

**Status:** ALPHA - Local testing only, NOT published to npm yet

**What's Ready:**
- ✅ stdio mode implementation complete
- ✅ Environment-based configuration working
- ✅ npm package structure ready
- ✅ Cross-platform binary wrapper functional
- ✅ Documentation complete

**What's NOT Ready:**
- ⚠️ npm package binaries: Only darwin-arm64 currently staged
- ⚠️ npm publish: Not executed (alpha testing first)
- ⚠️ Full platform testing: Needs validation on all 5 platforms
- ⚠️ CI/CD pipeline: Binary packaging automation needed

**Before v2.0.0 final release:**
1. Copy all platform binaries to `npm-package/bin/binaries/`
2. Test npm package locally on each platform
3. Validate stdio mode with Claude Desktop, Cursor, Continue.dev, Cline
4. Update main README.md to highlight stdio mode as primary
5. Publish to npm registry
6. Create GitHub release with binaries + npm package announcement

**Testing Instructions (Local Alpha):**
```bash
# 1. Ensure all binaries in npm-package/bin/binaries/
cp build/mcp-server-prtg_darwin_amd64 npm-package/bin/binaries/mcp-server-prtg-darwin-amd64
cp build/mcp-server-prtg_darwin_arm64 npm-package/bin/binaries/mcp-server-prtg-darwin-arm64
cp build/mcp-server-prtg_linux_amd64 npm-package/bin/binaries/mcp-server-prtg-linux-amd64
cp build/mcp-server-prtg_linux_arm64 npm-package/bin/binaries/mcp-server-prtg-linux-arm64
cp build/mcp-server-prtg_windows_amd64.exe npm-package/bin/binaries/mcp-server-prtg-windows-amd64.exe

# 2. Test local npm package
cd npm-package
npm pack
npm install -g senhub-io-mcp-server-prtg-2.0.0-alpha.1.tgz

# 3. Test with MCP client (Claude Desktop)
# Add to config with: command: "mcp-server-prtg", args: ["stdio"]
# Or: command: "node", args: ["/path/to/npm-package/bin/index.js"]

# 4. Verify environment variables work
export PRTG_URL="https://prtg.example.com:1616"
export PRTG_API_TOKEN="your-token"
node npm-package/bin/index.js

# 5. Validate binary auto-detection
# Check logs for correct platform binary loaded
```

---

### Benefits

**For Users:**
- **Zero Infrastructure**: No server setup, no TLS certificates, no firewall configuration
- **Simple Installation**: One-line npx command, no global installations required
- **Automatic Updates**: npx always fetches latest version
- **Local-First Security**: All processing local, no remote servers, no network exposure
- **Cross-Platform**: Same configuration works on macOS, Linux, Windows

**For Developers:**
- **Simpler Deployment**: stdin/stdout communication, no HTTP stack needed
- **Faster Iteration**: No service restarts, just reload MCP client
- **Better Debugging**: Direct process logs in stderr, MCP protocol in stdout
- **Modular Architecture**: HTTP and stdio modes coexist cleanly

**For LLMs (Claude, etc.):**
- **Standard MCP Protocol**: Full stdio JSON-RPC compliance
- **Real-Time Metrics**: Direct PRTG API v2 access for current data
- **Environment-Scoped**: Per-client PRTG credentials via environment variables

---

### Known Limitations (Alpha)

1. **Database Tools Unavailable**: stdio mode doesn't support PostgreSQL connection (by design)
   - **Impact**: 12 database tools not available (hierarchy, search, alerts, etc.)
   - **Workaround**: Use HTTP mode for database tools, stdio mode for metrics only
   - **Future**: Migrate database tools to API v2 (planned for v2.1.0)

2. **npm Binaries Incomplete**: Only darwin-arm64 binary currently staged
   - **Impact**: Other platforms will fail with "binary not found"
   - **Fix**: Copy all binaries before npm publish

3. **No CI/CD Automation**: Manual binary copying required
   - **Impact**: Error-prone release process
   - **Future**: Add GitHub Actions workflow for binary packaging

4. **Untested on Windows**: stdio mode untested on Windows platform
   - **Impact**: Unknown compatibility issues
   - **Testing Needed**: Windows validation before final release

---

**Release prepared by:** Matthieu Noirbusson

**Version:** 2.0.0-alpha.1 (ALPHA - NOT published to npm)

**Testing Status:**
- Core stdio implementation: PASS ✓
- Environment configuration: PASS ✓
- Binary wrapper script: PASS ✓
- macOS ARM64: TESTED ✓
- Other platforms: UNTESTED ⚠️

**Next Steps:**
1. Stage all platform binaries in npm-package/bin/binaries/
2. Test npm pack locally on each platform
3. Validate with real MCP clients (Claude Desktop priority)
4. Tag as v2.0.0-alpha.1 (local only, no push)
5. After validation → v2.0.0-rc.1 → v2.0.0 final

## [1.3.0-beta.1] - 2025-12-30

### Added

#### ASCII Visualizations for Time Series Data
- **Sparklines**: Mini-graphs using Unicode blocks (▁▂▃▄▅▆▇█) for visual trend display
- **Trend Indicators**: UP/DOWN/FLAT indicators showing direction of metric changes
- **Compact Statistics**: Min/Max/Avg/Current value summaries in token-efficient format
- **Anomaly Detection**: Statistical outlier identification (>2σ) automatically highlighted
- **Health Bars**: Visual progress indicators for percentage-based metrics
- **Automatic Integration**: Sparklines automatically appear in time series responses
- **New Module**: `internal/handlers/visualization.go` (178 lines) with comprehensive test suite (227 lines)

#### Contextual Suggestions for LLM Guidance
- **Smart Recommendations**: Context-aware "Next suggested actions" in all major response formatters
- **Alert-Based Suggestions**: When alerts detected, suggests investigating critical sensors
- **Sensor-Based Suggestions**: Recommends viewing alerts, trends, or narrowing search based on status
- **Device Overview Suggestions**: Proactive guidance for exploring device details
- **Search Result Suggestions**: Helps inspect and navigate found items
- **Command-Ready Format**: All suggestions use valid MCP command syntax, ready to execute
- **Reduces Back-and-Forth**: Proactive tool discovery reduces LLM query iterations

#### Pedagogical Error Handling System (from commit 661ccc8)
- **ErrorGuidance Helper**: Educational, actionable error messages designed for LLM users
- **6 Predefined Error Types**:
  - `ErrInvalidSensorID`: Guides users to search tools for valid sensor IDs
  - `ErrDeviceNotFound`: Suggests hierarchy/search tools to discover devices
  - `ErrSensorNotFound`: Provides troubleshooting steps for missing sensors
  - `ErrEmptySearchTerm`: Explains minimum search requirements
  - `ErrInvalidTimeRange`: Guides on valid time range parameters
  - `ErrCustomQueriesDisabled`: Explains security restrictions and alternatives
- **Enhanced Input Validation**: Early sensor_id validation in all metrics tool handlers
- **Test Coverage**: 14 test cases covering all error scenarios

### Changed
- **Time Series Formatting**: Enhanced `formatTimeSeriesForLLM()` to include automatic sparklines and trend analysis
- **Response Formatters**: All major formatters now include contextual suggestions
- **Error Philosophy**: Errors now guide toward resolution instead of just blocking actions
- **Token Efficiency**: Compact ASCII visualizations reduce token usage vs verbose JSON
- **User Experience**: LLMs can now visualize trends at a glance and get proactive guidance

### Fixed
- **Division by Zero**: Critical fix in `TrendIndicator()` when `firstAvg == 0`
- **Edge Cases**: Added test coverage for zero values (zero_to_positive, zero_to_negative, all_zeros)
- **Test Suite**: All tests now pass including new edge case scenarios

### Documentation
- **README.md**: Added ASCII Visualizations and Contextual Suggestions feature descriptions
- **ARCHITECTURE.md**: Complete visualization and suggestion system documentation (+141 lines)
- **TOOLS.md**: LLM-optimized features and updated tool response formats (+62 lines)
- **USAGE.md**: Detailed examples and usage guides for new features (+147 lines)
- **TROUBLESHOOTING.md**: Updated with error handling guidance

### Technical Details

#### New Files
- `internal/handlers/visualization.go` (178 lines)
- `internal/handlers/visualization_test.go` (227 lines)
- `internal/handlers/tools_metrics_integration_test.go` (221 lines)
- `internal/handlers/formatting_suggestions_test.go` (252 lines)
- `internal/handlers/errors.go` (from commit 661ccc8)
- `internal/handlers/errors_test.go` (from commit 661ccc8)

#### Modified Files
- `internal/handlers/formatting.go` (+93 lines for suggestions)
- `internal/handlers/tools_metrics.go` (+70 lines for sparklines, validation)
- Documentation files: 5 markdown files updated

#### Test Coverage
- All unit tests passing (100%)
- New visualization tests: 30+ test cases
- New suggestion tests: 15 test cases
- New error handling tests: 14 test cases
- Race condition tests: All passing
- Total test coverage: Comprehensive across all modules

#### Commits Included
- `193989f`: fix: handle division by zero in TrendIndicator and update documentation
- `e6ad019`: feat: add ASCII visualizations and contextual suggestions for LLM guidance
- `661ccc8`: feat: add pedagogical error handling system for LLM users

---

### Binary Checksums (SHA256)

```
6756276c3abe11497ea54c4ce52e1118c95832b2bdaf4e85466116fe9d5c8adb  mcp-server-prtg_darwin_amd64.zip
85f743088e30939a8232d103a5a1ed909dae9311d87b0b173b9196ce3c600c7e  mcp-server-prtg_darwin_arm64.zip
fb762be729d2a06124c4a67b96c00b6245dfc09572323880d0cd695cb256999b  mcp-server-prtg_linux_amd64.zip
470db0ff0b55576a8666fa8c18bb436dda63507f650e9c0963888a47b23d8916  mcp-server-prtg_linux_arm64.zip
840035d5d46fe3b5403203cc77ee7ab9b3a2f43d98a035d7c828e4b682e32243  mcp-server-prtg_windows_amd64.zip
```

---

### Benefits

**For LLMs:**
- Visual trend comprehension at a glance without parsing JSON
- Automatic anomaly alerts reduce need for statistical analysis
- Proactive command suggestions reduce back-and-forth queries
- Educational error messages guide toward successful tool usage

**For Users:**
- Token-efficient compact format reduces API costs
- Faster troubleshooting with visual indicators
- Better tool discovery through contextual suggestions
- More intuitive error handling

**For Developers:**
- Comprehensive test coverage ensures reliability
- Modular visualization system easy to extend
- Well-documented architecture for future enhancements
- Backward compatible with existing features

---

**Release prepared by:** Matthieu Noirbusson

**Code Review Score:** 10/10 (after division by zero fix)

**Note:** This is a beta release for testing the new visualization and suggestion systems. The changes are backward compatible and add new functionality without breaking existing features. Inspired by best practices from Honeycomb's MCP implementation.

**Testing Status:**
- All unit tests: PASS ✓
- Race condition tests: PASS ✓
- Security tests: PASS ✓
- Edge case tests: PASS ✓
- Integration tests: PASS ✓
