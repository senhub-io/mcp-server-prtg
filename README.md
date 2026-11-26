# MCP Server PRTG

[![Release](https://img.shields.io/github/v/release/senhub-io/mcp-server-prtg?include_prereleases)](https://github.com/senhub-io/mcp-server-prtg/releases)
[![Tests](https://img.shields.io/github/actions/workflow/status/senhub-io/mcp-server-prtg/go-test.yml?branch=main&label=tests)](https://github.com/senhub-io/mcp-server-prtg/actions/workflows/go-test.yml)
[![Go](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

A [Model Context Protocol](https://modelcontextprotocol.io) server that gives MCP clients (Claude Desktop, Cursor, Continue, Cline, ...) read access to Paessler PRTG Network Monitor data: sensor status, alerts, device overviews, hierarchy, search, tags, business processes, statistics and channel time series.

## Data sources

- **PostgreSQL database filled by Paessler PRTG Data Exporter** — primary source in HTTP mode. Reading the database instead of calling PRTG keeps the load off the PRTG core server.
- **PRTG API v2** — optional in HTTP mode, the only source in stdio mode. Every API call goes through a request guard: short-lived cache, coalescing of identical concurrent requests, at most 2 requests in flight by default.

## Modes

| Mode | For | Data sources | Tools | Install |
|---|---|---|---|---|
| HTTP service | A shared server (typically next to PRTG) that several users' MCP clients connect to over HTTPS at `https://<host>:8443/mcp` with a Bearer API key | PostgreSQL + optional PRTG API v2 | 14 tools (12 PostgreSQL tools + 2 PRTG API v2 tools when enabled) | Windows: MSI installer (recommended); Linux: systemd service via the release binary |
| stdio | A single user's machine, as a child process of the MCP client | PRTG API v2 only | 13 tools | `npx -y @senhub-io/mcp-server-prtg` |

## Tools

| Tool | What it returns | HTTP | stdio |
|---|---|---|---|
| `prtg_get_sensors` | sensors with filters | yes | yes |
| `prtg_get_sensor_status` | one sensor in detail | yes | yes |
| `prtg_get_alerts` | sensors in warning or down state | yes | yes |
| `prtg_device_overview` | device with its sensors and status breakdown | yes | yes |
| `prtg_top_sensors` | sensors ranked by a metric | yes | yes |
| `prtg_get_hierarchy` | group / device / sensor tree | yes | yes |
| `prtg_search` | search across groups, devices, sensors | yes | yes |
| `prtg_get_groups` | groups and probes | yes | yes |
| `prtg_get_tags` | tags and usage | yes | yes |
| `prtg_get_business_processes` | business process sensors | yes | yes |
| `prtg_get_statistics` | server-wide statistics | yes | yes |
| `prtg_query_sql` | read-only SQL on the Data Exporter database, disabled by default | yes | no |
| `prtg_get_channel_current_values` | current channel values with units | yes (PRTG API v2 enabled) | yes |
| `prtg_get_sensor_timeseries` | channel history for live, short, medium or long periods, with sparklines | yes (PRTG API v2 enabled) | yes |

See [docs/TOOLS.md](docs/TOOLS.md) for details.

## Quick start

### 1. HTTP service on Windows

Download `mcp-server-prtg-<version>-amd64.msi` from [Releases](https://github.com/senhub-io/mcp-server-prtg/releases) and run it. The service starts and generates `C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml` with a random API key and a self-signed certificate. Set the database (and optionally PRTG API v2) settings there, then:

```powershell
Restart-Service mcp-server-prtg
```

Check `https://localhost:8443/health`. Full steps, updates and rollback: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

### 2. HTTP service on Linux

Unzip the release archive and run:

```sh
sudo ./mcp-server-prtg install
# edit config.yaml next to the binary
sudo ./mcp-server-prtg start
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

### 3. Connect a client to the HTTP service (Claude Desktop example)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["mcp-remote", "https://<MCP_SERVER_HOST>:8443/mcp", "--header", "Authorization:Bearer ${MCP_SERVER_API_KEY}"],
      "env": { "MCP_SERVER_API_KEY": "<api_key from config.yaml>" }
    }
  }
}
```

With the generated self-signed certificate, add `"NODE_TLS_REJECT_UNAUTHORIZED": "0"` to `env`, or install a certificate trusted by the clients.

### 4. stdio mode (Claude Desktop example)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": { "PRTG_URL": "https://prtg.example.com:1616", "PRTG_API_TOKEN": "<PRTG API v2 token>" }
    }
  }
}
```

Other stdio variables: `PRTG_VERIFY_SSL` (true), `PRTG_TIMEOUT` (30), `PRTG_MAX_CONCURRENT` (2), `PRTG_CACHE_DISABLED` (false), `LOG_LEVEL` (info).

## Minimal config.yaml (HTTP mode)

```yaml
config_version: 1
server:
  api_key: "<generated, 16 characters minimum>"
  port: 8443
  enable_tls: true
database:
  host: "<Data Exporter PostgreSQL host>"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: "<password or PRTG_DB_PASSWORD env var>"
  sslmode: "require"
prtg:
  enabled: false
  base_url: "https://prtg.example.com:1616"
  api_token: "<PRTG API v2 token>"
logging:
  level: "info"
```

All options: [docs/CONFIGURATION.md](docs/CONFIGURATION.md).

## Security

- Bearer API key only (header, no query string), constant-time comparison, brute-force rate limiting.
- TLS by default.
- `prtg_query_sql` is off by default and, when enabled, runs read-only with a timeout, a row cap and a deny-list of credential tables.
- Use a dedicated read-only PostgreSQL role.
- The Windows data folder is restricted to SYSTEM and Administrators.

Report vulnerabilities privately via [GitHub Security Advisories](https://github.com/senhub-io/mcp-server-prtg/security/advisories/new), not public issues.

## Documentation

- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — install, update, roll back on Windows MSI, Linux, stdio
- [docs/CONFIGURATION.md](docs/CONFIGURATION.md) — configuration options
- [docs/TOOLS.md](docs/TOOLS.md) — tool reference
- [docs/USAGE.md](docs/USAGE.md) — usage
- Client guides: [docs/CLAUDE_DESKTOP_CONFIG.md](docs/CLAUDE_DESKTOP_CONFIG.md), [docs/CURSOR_CONFIG.md](docs/CURSOR_CONFIG.md), [docs/CONTINUE_CONFIG.md](docs/CONTINUE_CONFIG.md), [docs/CLINE_CONFIG.md](docs/CLINE_CONFIG.md)
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) — troubleshooting
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture
- [CHANGELOG.md](CHANGELOG.md) — changelog

## Build from source

Requires Go 1.26 and `make`.

```sh
make build        # current platform, output in ./build/
make build-all
make test
```

## Contributing

Issues and pull requests are welcome. Run `make test` before opening a PR; one logical change per commit.

## License

MIT, see [LICENSE](LICENSE). Maintained by Sensor Factory (SenHub).