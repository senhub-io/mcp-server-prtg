# MCP Server PRTG

[![Version](https://img.shields.io/badge/version-v2.0.0--alpha-blue)](https://github.com/senhub-io/mcp-server-prtg)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

**MCP Server PRTG** is a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that exposes PRTG monitoring data through a standardized API. It enables LLMs (like LLM) to query sensor status, analyze alerts, and generate reports on your monitoring infrastructure in real-time.

## Features

- **SSE Transport (Server-Sent Events)** - v2 architecture with HTTPS proxy and internal server
- **6 MCP Tools** to query PRTG data (sensors, alerts, statistics, SQL)
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
  api_key: "your-generated-api-key"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
database:
  host: localhost
  port: 5432
  name: prtg_data_exporter
  user: prtg_reader
  sslmode: disable
logging:
  level: info
```

**See:** [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for complete documentation

## Usage with MCP Client

1. Install [mcp-proxy](https://github.com/sparfenyuk/mcp-proxy):
   ```bash
   pip install mcp-proxy
   ```

2. Configure MCP Client (`mcp_client_config.json`):
   ```json
   {
     "mcpServers": {
       "prtg": {
         "command": "mcp-proxy",
         "args": [
           "http://your-server:8443/sse",
           "--headers",
           "Authorization",
           "Bearer your-api-key"
         ]
       }
     }
   }
   ```

3. Restart MCP Client

**Note on HTTPS/TLS:** mcp-proxy does not support the `--insecure` flag for self-signed certificates. For development, use HTTP as shown above. For production, either use trusted CA certificates or add the self-signed certificate to your system trust store. See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md#certificate-verification-failed-with-mcp-proxy) for details.

**See:** [docs/USAGE.md](docs/USAGE.md) for usage examples

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `prtg_get_sensors` | List sensors with filters (name, status, tags) |
| `prtg_get_sensor_status` | Details of a specific sensor by ID |
| `prtg_get_alerts` | Sensors in alert state (warning/down) |
| `prtg_device_overview` | Complete overview of a device |
| `prtg_top_sensors` | Top sensors by uptime/downtime/alerts |
| `prtg_query_sql` | Custom SQL queries on PRTG database |

**See:** [docs/TOOLS.md](docs/TOOLS.md) for complete tool documentation

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

**Version:** v2.0.0-alpha
**Organization:** SenHub.io
**MCP Protocol:** [modelcontextprotocol.io](https://modelcontextprotocol.io)
