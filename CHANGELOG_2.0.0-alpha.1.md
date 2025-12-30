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
