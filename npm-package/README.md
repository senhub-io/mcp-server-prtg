# MCP Server PRTG

[![npm version](https://img.shields.io/npm/v/@senhub-io/mcp-server-prtg)](https://www.npmjs.com/package/@senhub-io/mcp-server-prtg)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**MCP Server PRTG** is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that exposes PRTG monitoring data to LLMs like Claude. Run it locally as a plugin via stdio (standard input/output) for seamless integration with MCP clients.

## Features

- ✅ **15 MCP Tools** for querying PRTG data (sensors, alerts, hierarchy, metrics, etc.)
- ✅ **stdio Protocol** - Runs locally as a child process (no server infrastructure needed)
- ✅ **PRTG API v2** Integration for real-time metrics and historical data
- ✅ **ASCII Visualizations** - Sparklines, trend indicators, and statistics
- ✅ **Contextual Suggestions** - Smart "next actions" recommendations for LLMs
- ✅ **Cross-platform** - macOS, Linux, Windows (x64, ARM64)

## Quick Start

### Prerequisites

- **PRTG Network Monitor** with API v2 enabled
- **PRTG API Token** (get from PRTG web interface → Setup → Account Settings → API Keys)
- **Node.js 16+** (for npm)

### Installation

```bash
# Option 1: Use with npx (no installation, always latest version)
# Just add to your MCP client config (see below)

# Option 2: Global installation
npm install -g @senhub-io/mcp-server-prtg
```

### Configuration

#### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%/Claude/claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-prtg-api-token-here",
        "PRTG_VERIFY_SSL": "true"
      }
    }
  }
}
```

#### Cursor / Continue.dev / Cline

Similar configuration in their respective config files. See [full documentation](https://github.com/senhub-io/mcp-server-prtg/tree/main/docs) for details.

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PRTG_URL` | ✅ Yes | - | PRTG server URL with API port (e.g., `https://prtg.example.com:1616`) |
| `PRTG_API_TOKEN` | ✅ Yes | - | PRTG API v2 Bearer token |
| `PRTG_VERIFY_SSL` | No | `true` | Verify SSL certificates (`true`/`false`) |
| `PRTG_TIMEOUT` | No | `30` | HTTP request timeout in seconds |
| `LOG_LEVEL` | No | `info` | Logging level (`debug`, `info`, `warn`, `error`) |

### Restart MCP Client

After updating the configuration, restart your MCP client (Claude Desktop, Cursor, etc.) to load the new server.

## Usage Examples

Once configured, you can ask your LLM:

```
"Show me all sensors in Down status"
"What's the current status of sensor ID 2024?"
"Get performance metrics for the last 24 hours for sensor 2024"
"List all devices in the Production group"
"Show me a sparkline chart of CPU usage over the last day"
```

The MCP server provides 15 tools:

### Core Tools (Database)
- `prtg_get_sensors` - List sensors with filters
- `prtg_get_sensor_status` - Get sensor details
- `prtg_get_alerts` - Find sensors in alert state
- `prtg_device_overview` - Complete device overview
- `prtg_top_sensors` - Top sensors by metrics
- `prtg_get_hierarchy` - Navigate PRTG tree
- `prtg_search` - Universal search
- `prtg_get_groups` - List groups/probes
- `prtg_get_tags` - List tags
- `prtg_get_business_processes` - Query Business Process sensors
- `prtg_get_statistics` - Server-wide statistics
- `prtg_query_sql` - Custom SQL queries

### Metrics Tools (API v2)
- `prtg_get_channel_current_values` - Current channel values
- `prtg_get_sensor_timeseries` - Historical time series
- `prtg_get_sensor_history_custom` - Custom date range queries

## Architecture

```
┌─────────────────────────────────────┐
│  MCP Client (Claude Desktop, etc.) │
│  ┌───────────────────────────────┐  │
│  │ MCP Protocol Handler          │  │
│  └──────────┬────────────────────┘  │
│             │ stdio (JSON-RPC)      │
│  ┌──────────▼────────────────────┐  │
│  │ @senhub-io/mcp-server-prtg    │  │
│  │ (Node.js wrapper → Go binary) │  │
│  └──────────┬────────────────────┘  │
└─────────────┼───────────────────────┘
              │ HTTPS
              ▼
   ┌─────────────────────────┐
   │  PRTG API v2            │
   │  (your PRTG server)     │
   └─────────────────────────┘
```

- **stdio mode**: Runs as a local process, communicates via stdin/stdout (no network exposure)
- **Go binary**: High-performance native binaries for each platform
- **npm wrapper**: Automatically selects the correct binary for your OS/architecture

## Troubleshooting

### "Binary not found" Error

```bash
# Reinstall the package
npm install -g @senhub-io/mcp-server-prtg --force
```

### "Unsupported platform" Error

Check that your platform is supported:
- macOS: x64, ARM64 (M1/M2/M3)
- Linux: x64, ARM64
- Windows: x64

### Connection Issues

- Verify `PRTG_URL` includes the API port (usually `:1616`, not `:443`)
- Check that PRTG API v2 is enabled (PRTG web interface → Setup → System → API)
- Test API token: `curl -H "Authorization: Bearer YOUR_TOKEN" https://prtg:1616/api/v2/health`

### Self-Signed SSL Certificates

If using self-signed certificates, set `PRTG_VERIFY_SSL=false` in your config.

## Migration from v1.x (HTTP Server)

v2.0 uses stdio mode instead of HTTP server. To migrate:

1. **Stop v1.x HTTP server** (if running)
2. **Update MCP client config** to use the new stdio format (see above)
3. **Remove server infrastructure** (no more TLS certs, firewall rules, systemd services)

Old v1.x HTTP mode is still available via the standalone binary (`mcp-server-prtg run`).

## Documentation

- [Installation Guide](https://github.com/senhub-io/mcp-server-prtg/blob/main/docs/INSTALLATION.md)
- [Configuration Guide](https://github.com/senhub-io/mcp-server-prtg/blob/main/docs/CONFIGURATION.md)
- [MCP Tools Reference](https://github.com/senhub-io/mcp-server-prtg/blob/main/docs/TOOLS.md)
- [Usage Examples](https://github.com/senhub-io/mcp-server-prtg/blob/main/docs/USAGE.md)

## Development

This package wraps pre-compiled Go binaries. To build from source:

```bash
git clone https://github.com/senhub-io/mcp-server-prtg.git
cd mcp-server-prtg
make build-all  # Builds for all platforms
```

Binaries are in `./build/` directory.

## License

MIT License - See [LICENSE](https://github.com/senhub-io/mcp-server-prtg/blob/main/LICENSE)

## Support

- [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues)
- [Documentation](https://github.com/senhub-io/mcp-server-prtg/tree/main/docs)

---

**Organization:** [SenHub.io](https://senhub.io)
**MCP Protocol:** [modelcontextprotocol.io](https://modelcontextprotocol.io)
