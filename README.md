# MCP Server PRTG

[![Version](https://img.shields.io/github/v/release/senhub-io/mcp-server-prtg?include_prereleases)](https://github.com/senhub-io/mcp-server-prtg/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/senhub-io/mcp-server-prtg/go-test.yml?branch=main)](https://github.com/senhub-io/mcp-server-prtg/actions)
[![Go](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**MCP Server PRTG** is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that exposes PRTG monitoring data through a standardized API. It enables LLMs (like Claude) to query sensor status, analyze alerts, and generate reports on your monitoring infrastructure in real-time.

## Prerequisites

**MCP Server PRTG** requires access to:
1. **PRTG Data Exporter** - PostgreSQL database (`prtg_data_exporter`) for sensor status and configuration
2. **PRTG Core Server** - API v2 (optional, for historical metrics and channel data)

### Deployment Options

**Option A: Co-located (Recommended)**
- Install MCP Server PRTG on the same Windows server as PRTG Data Exporter
- Best performance with local PostgreSQL access
- No network latency

**Option B: Remote Deployment**
- Install MCP Server PRTG on any server (Linux, macOS, Windows)
- Requires network access to:
  - PostgreSQL database (default port 5432)
  - PRTG API v2 (default port 1616)
- Configure firewall rules accordingly

**Note:** PRTG Data Exporter is Windows-only, but MCP Server PRTG runs on any platform.

## Features

- **Streamable HTTP Transport** - Modern MCP protocol (2025-03-26) with HTTP SSE streaming
- **15 MCP Tools** to query PRTG data:
  - **12 tools** for PostgreSQL database (sensors, alerts, hierarchy, groups, tags, business processes, statistics, SQL)
  - **3 tools** for PRTG API v2 (historical metrics, time series, channel values)
- **PRTG API v2 Integration** - Query historical metrics and real-time channel data directly from PRTG
- **Bearer Token Authentication** (RFC 6750)
- **TLS/HTTPS Support** with automatic certificate generation
- **Windows Service** - Installation and management via kardianos/service
- **File Logging** with rotation (lumberjack)
- **Hot-reload** configuration
- **Multi-platform** - Windows, Linux, macOS

## Quick Installation

### Windows

```powershell
# 1. Download the binary
#    mcp-server-prtg_windows_amd64.exe

# 2. Install as Windows service
.\mcp-server-prtg.exe install

# 3. Start the service
.\mcp-server-prtg.exe start

# 4. Check status
.\mcp-server-prtg.exe status
```

### Linux / macOS

```bash
# 1. Download the appropriate binary
#    mcp-server-prtg_linux_amd64
#    mcp-server-prtg_darwin_arm64

# 2. Make it executable
chmod +x mcp-server-prtg

# 3. Install as systemd service (Linux)
sudo ./mcp-server-prtg install

# 4. Start
sudo ./mcp-server-prtg start

# Or run in console (foreground)
./mcp-server-prtg run
```

## Configuration

On first installation, a `config.yaml` file is automatically generated with:
- Unique API key
- Self-signed TLS certificates
- Default PostgreSQL configuration

**Location:** `./config.yaml` (or specified via `--config`)

**Minimal example:**
```yaml
version: 1
server:
  api_key: "your-mcp-server-api-key"  # MCP Server authentication key
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
database:
  host: localhost                     # PostgreSQL server (PRTG Data Exporter)
  port: 5432
  name: prtg_data_exporter
  user: prtg_reader
  sslmode: disable
prtg:
  enabled: true                       # Enable PRTG API v2 integration
  base_url: "https://prtg.example.com:1616"  # PRTG Core Server API v2
  api_token: "your-prtg-api-v2-token" # PRTG API v2 Bearer token
  timeout: 30
  verify_ssl: true
logging:
  level: info
```

**PRTG API v2 Configuration (optional):**

Enable PRTG API v2 to query historical metrics and real-time channel data:

- `enabled`: Enable/disable PRTG API access (default: false)
- `base_url`: PRTG server URL with API v2 port (typically 1616, not 443)
- `api_token`: PRTG API v2 Bearer token (get from PRTG web interface)
- `timeout`: HTTP request timeout in seconds (default: 30)
- `verify_ssl`: Verify SSL certificates (set to false for self-signed certs)

**Note:** PRTG API v2 configuration is optional. If not configured, only PostgreSQL-based tools will be available.

**See:** [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for complete documentation

## Usage with MCP Clients

Configure your MCP Client (like Claude Desktop) to connect to MCP Server PRTG using `mcp-remote`:

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "your-mcp-server-api-key"
      }
    }
  }
}
```

**Important:**
- `<MCP_SERVER_HOST>`: IP or hostname where MCP Server PRTG is running (not PRTG Core Server)
- `MCP_SERVER_API_KEY`: API key from MCP Server PRTG `config.yaml` (not PRTG API v2 token)

**For HTTP (development only):**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "http://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "your-mcp-server-api-key"
      }
    }
  }
}
```

**Note:** For production with self-signed certificates, add `NODE_TLS_REJECT_UNAUTHORIZED=0` to `env` (not recommended) or use trusted CA certificates.

**See:** [docs/USAGE.md](docs/USAGE.md) for usage examples

## Available MCP Tools

### PostgreSQL-Based Tools (12)

| Tool | Description |
|------|-------------|
| `prtg_get_sensors` | List sensors with filters (name, status, tags) |
| `prtg_get_sensor_status` | Details of a specific sensor by ID |
| `prtg_get_alerts` | Sensors in alert state (warning/down) |
| `prtg_device_overview` | Complete overview of a device with group info and tags |
| `prtg_top_sensors` | Top sensors by uptime/downtime/alerts |
| `prtg_get_hierarchy` | Navigate PRTG hierarchy tree (groups/devices/sensors) |
| `prtg_search` | Universal search across groups, devices, and sensors |
| `prtg_get_groups` | List groups/probes with filtering options |
| `prtg_get_tags` | List tags with usage statistics |
| `prtg_get_business_processes` | Query Business Process sensors |
| `prtg_get_statistics` | Server-wide aggregated statistics |
| `prtg_query_sql` | Custom SQL queries on PRTG database |

### PRTG API v2 Tools (3)

| Tool | Description |
|------|-------------|
| `prtg_get_channel_current_values` | **PRIMARY tool** for current sensor state - Get all channel values, units, and timestamps |
| `prtg_get_sensor_timeseries` | Query historical time series data (live, short, medium, long periods) |
| `prtg_get_sensor_history_custom` | Query historical data for custom date/time ranges |

**See:** [docs/TOOLS.md](docs/TOOLS.md) for complete tool documentation

## MCP Client Configuration

MCP Server PRTG works with any MCP-compatible client:

**Tested & Validated:**
- **[Claude Desktop](docs/CLAUDE_DESKTOP_CONFIG.md)** ✅ - Anthropic's official desktop app (fully tested)

**Community Configurations** (based on official specs, not yet tested):
- **[Continue.dev](docs/CONTINUE_CONFIG.md)** ⚠️ - VS Code & JetBrains extension
- **[Cursor](docs/CURSOR_CONFIG.md)** ⚠️ - AI-first code editor
- **[Cline](docs/CLINE_CONFIG.md)** ⚠️ - Autonomous coding agent for VS Code

> 💡 **Feedback welcome!** If you test these configurations, please share your experience via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues).

## Documentation

- [Installation Guide](docs/INSTALLATION.md)
- [Configuration Guide](docs/CONFIGURATION.md)
- [Usage Guide](docs/USAGE.md)
- [MCP Tools Reference](docs/TOOLS.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## Build from Source

```bash
# Clone the repo
git clone https://github.com/senhub-io/mcp-server-prtg.git
cd mcp-server-prtg

# Build for current platform
make build

# Build for all platforms
make build-all

# Build Windows only
make build-windows

# Binaries are in ./build/
```

## Useful Commands

```bash
# Run in console mode (foreground)
./mcp-server-prtg run

# View detailed status (service + database)
./mcp-server-prtg status

# Stop the service
./mcp-server-prtg stop

# Uninstall (with automatic cleanup)
./mcp-server-prtg uninstall

# Show version
./mcp-server-prtg --version

# Help
./mcp-server-prtg --help
```

## Troubleshooting

**Service won't start?**
- Check logs: `./logs/mcp-server-prtg.log`
- Check config: `./mcp-server-prtg status`
- Enable debug: `log_level: debug` in config.yaml

**Database connection failed?**
- Verify PostgreSQL is running
- Check credentials in config.yaml
- For SSL: `sslmode: require` (or `disable` for testing)

**See:** [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

## License

MIT License - See [LICENSE](LICENSE) for details

## Contributing

Contributions are welcome! Feel free to open an issue or submit a PR.

## Support

For questions or issues:
- Open a [GitHub Issue](https://github.com/senhub-io/mcp-server-prtg/issues)
- Check the [documentation](docs/)

---

**Organization:** SenHub.io
**MCP Protocol:** [modelcontextprotocol.io](https://modelcontextprotocol.io)
