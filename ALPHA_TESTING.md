# Alpha Testing Guide - v2.0.0-alpha.1

## Overview

This alpha release introduces **stdio mode** for MCP Server PRTG, enabling local plugin deployment with Claude Desktop, Cursor, and other MCP clients.

**⚠️ This is an ALPHA release for testing purposes only.**

## What's New

- **stdio transport**: Local plugin mode (no HTTP server needed)
- **npm package**: Cross-platform distribution via npm
- **Environment-based config**: Simple PRTG_URL and PRTG_API_TOKEN configuration
- **3 PRTG API v2 tools**: Current values, timeseries, and history queries

## Known Limitations

- **Only 3/15 tools available** in stdio mode (database tools require HTTP mode)
- **Platform support**: Tested on macOS ARM64 only (Windows, Linux, Intel Mac untested)
- **Not published to npm yet**: Must build locally or download binaries from GitHub release

## Testing Instructions

### Option 1: Build from Source

1. **Clone and checkout:**
   ```bash
   git clone https://github.com/senhub-io/mcp-server-prtg.git
   cd mcp-server-prtg
   git checkout feature/stdio-mode
   ```

2. **Build binaries:**
   ```bash
   make build-all
   ```

3. **Copy binaries to npm package:**
   ```bash
   mkdir -p npm-package/bin/binaries
   cp build/mcp-server-prtg_darwin_amd64 npm-package/bin/binaries/mcp-server-prtg-darwin-amd64
   cp build/mcp-server-prtg_darwin_arm64 npm-package/bin/binaries/mcp-server-prtg-darwin-arm64
   cp build/mcp-server-prtg_linux_amd64 npm-package/bin/binaries/mcp-server-prtg-linux-amd64
   cp build/mcp-server-prtg_linux_arm64 npm-package/bin/binaries/mcp-server-prtg-linux-arm64
   cp build/mcp-server-prtg_windows_amd64.exe npm-package/bin/binaries/mcp-server-prtg-windows-amd64.exe
   ```

4. **Test npm wrapper:**
   ```bash
   PRTG_URL="https://prtg.example.com:1616" \
   PRTG_API_TOKEN="your-token-here" \
   node npm-package/bin/index.js
   ```

### Option 2: Download from GitHub Release

1. Download binaries from [GitHub Releases](https://github.com/senhub-io/mcp-server-prtg/releases/tag/2.0.0-alpha.1)
2. Extract to `npm-package/bin/binaries/`
3. Follow step 4 above

## Claude Desktop Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "prtg-stdio": {
      "command": "node",
      "args": [
        "/absolute/path/to/mcp-server-prtg/npm-package/bin/index.js"
      ],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-prtg-api-token-here",
        "PRTG_VERIFY_SSL": "true"
      }
    }
  }
}
```

**Windows:** `%APPDATA%/Claude/claude_desktop_config.json`

**Restart Claude Desktop** after updating the config.

## Testing Checklist

- [ ] Binary runs on your platform (macOS, Linux, or Windows)
- [ ] npm wrapper detects correct platform and architecture
- [ ] Environment variables are loaded correctly
- [ ] PRTG API connectivity works (check stderr logs)
- [ ] Claude Desktop recognizes the MCP server
- [ ] 3 tools appear in Claude Desktop: `prtg_get_channel_current_values`, `prtg_get_sensor_timeseries`, `prtg_get_sensor_history_custom`
- [ ] Tools execute and return valid data

## Available Tools (stdio mode)

1. **prtg_get_channel_current_values** - Get current values for sensor channels
2. **prtg_get_sensor_timeseries** - Get historical time series data (last N hours/days)
3. **prtg_get_sensor_history_custom** - Get historical data for custom date ranges

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PRTG_URL` | ✅ Yes | - | PRTG server URL with API port (e.g., `https://prtg.example.com:1616`) |
| `PRTG_API_TOKEN` | ✅ Yes | - | PRTG API v2 Bearer token |
| `PRTG_VERIFY_SSL` | No | `true` | Verify SSL certificates (`true`/`false`) |
| `PRTG_TIMEOUT` | No | `30` | HTTP request timeout in seconds |
| `LOG_LEVEL` | No | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

## Debugging

Check stderr output for logs:

```bash
PRTG_URL="..." PRTG_API_TOKEN="..." node npm-package/bin/index.js 2>&1 | grep level
```

Expected output:
```json
{"level":"info","mode":"stdio","message":"Starting MCP Server PRTG in stdio mode"}
{"level":"info","prtg_url":"https://...","message":"Configuration loaded from environment"}
{"level":"info","tools_count":3,"message":"PRTG API v2 metrics tools registered"}
{"level":"info","message":"MCP stdio server starting"}
```

## Reporting Issues

Please report issues at: https://github.com/senhub-io/mcp-server-prtg/issues

Include:
- Platform and architecture (e.g., macOS ARM64, Windows x64)
- Node.js version
- Complete stderr logs
- Steps to reproduce

## Comparison: stdio vs HTTP Mode

| Feature | stdio Mode (v2.0) | HTTP Server Mode (v1.x) |
|---------|-------------------|-------------------------|
| Deployment | Local plugin | Remote server |
| Configuration | Environment variables | YAML config file |
| Database tools | ❌ Not available (API v2 only) | ✅ 12 tools |
| Metrics tools | ✅ 3 tools (API v2) | ✅ 3 tools (API v2) |
| Security | Local process only | TLS certificates, API keys |
| Multi-client | One client only | Multiple clients |

## Next Steps (Post-Alpha)

1. **Migration to API v2**: All 12 database tools will be rewritten to use PRTG API v2
2. **Multi-platform testing**: Validate Windows, Linux, Intel Mac
3. **npm publication**: Publish to `@senhub-io/mcp-server-prtg` on npm
4. **Documentation updates**: Complete user guides for stdio mode
5. **RC release**: Release candidate with full platform support
6. **v2.0.0 final**: Stable release with API v2 migration complete

## License

MIT License - See [LICENSE](./LICENSE)
