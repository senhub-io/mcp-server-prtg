# Continue.dev Configuration

Configure [Continue.dev](https://continue.dev/) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running
- Continue.dev extension installed in VS Code or JetBrains IDE
- Server API key (run `mcp-server-prtg config show-api-key` on the server, or copy it from the administration page at `https://localhost:<port>/admin/`)

## Getting the Ready-to-Paste Configuration

Run `mcp-server-prtg config client` on the server to print ready-to-paste configuration blocks for supported clients, or open the administration page (`https://localhost:<port>/admin/`), sign in with the API key, and use the **AI clients** tab to copy the configuration.

## Configuration Methods

Continue.dev supports two configuration approaches:

### Method 1: JSON Config File (Recommended)

**Location:** `.continue/mcpServers/prtg.json` (in your workspace or home directory)

**Configuration:**

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://your-server:8443/mcp",
      "--header",
      "Authorization:Bearer YOUR_API_KEY"
    ]
  }
}
```

**With environment variable for API key:**

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://your-server:8443/mcp",
      "--header",
      "Authorization:Bearer ${MCP_PRTG_API_KEY}"
    ],
    "env": {
      "MCP_PRTG_API_KEY": "your-api-key-here"
    }
  }
}
```

### Method 2: Experimental Config (config.json)

**Location:** `.continue/config.json`

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "prtg",
        "transport": {
          "type": "streamable-http",
          "url": "https://your-server:8443/mcp",
          "headers": {
            "Authorization": "Bearer YOUR_API_KEY"
          }
        }
      }
    ]
  }
}
```

## Configuration Options

### Basic (Local Server)

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://localhost:8443/mcp",
      "--header",
      "Authorization:Bearer a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
    ]
  }
}
```

### Remote Server (Production)

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://prtg-mcp.company.com:8443/mcp",
      "--header",
      "Authorization:Bearer ${MCP_PRTG_API_KEY}"
    ],
    "env": {
      "MCP_PRTG_API_KEY": "production-api-key"
    }
  }
}
```

### Self-Signed Certificates

The server uses a self-signed certificate by default. Copy `server.crt` from the server's `certs` folder (next to `config.yaml`) to the client computer and trust it instead of disabling verification:

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://your-server:8443/mcp",
      "--header",
      "Authorization:Bearer YOUR_API_KEY"
    ],
    "env": {
      "NODE_EXTRA_CA_CERTS": "/path/to/server.crt"
    }
  }
}
```

Do not use `NODE_TLS_REJECT_UNAUTHORIZED=0`; it exposes the API key.

## Verification

### 1. Reload Continue

After configuration:
1. Open VS Code Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`)
2. Run: "Developer: Reload Window"

### 2. Check MCP Server Status

Continue.dev will show configured MCP servers in:
- Continue sidebar
- Settings > MCP Servers

### 3. Test Tools Availability

In Continue chat, try asking:

```
Show me all critical PRTG sensors
```

Continue should use the `prtg_get_sensors` tool to query your PRTG data.

## Available Tools

Once connected, Continue.dev will have access to all MCP Server PRTG tools:

**Database tools:**
- `prtg_get_sensors` - Query sensors with filters
- `prtg_get_sensor_status` - Detailed sensor status
- `prtg_get_alerts` - Sensors in alert state (not Up)
- `prtg_device_overview` - Device summary with its sensors
- `prtg_top_sensors` - Top sensors by uptime, downtime or alerts
- `prtg_get_hierarchy` - Group, device and sensor tree
- `prtg_search` - Search groups, devices and sensors
- `prtg_get_groups` - List groups
- `prtg_get_tags` - List tags
- `prtg_get_business_processes` - Business process status
- `prtg_get_statistics` - System statistics and Data Exporter collection profiles
- `prtg_query_sql` - Custom SQL queries (only if `allow_custom_queries: true`)

**Metrics tools (registered when the database holds channel history or the PRTG API is configured):**
- `prtg_get_channel_current_values` - Current channel values, read from the database first; each answer starts with a `Source:` line
- `prtg_get_sensor_timeseries` - Time series data for a predefined period (`live`, `short`, `medium`, `long`)

## Troubleshooting

### MCP Server Not Appearing

**Check:**
1. JSON syntax is valid
2. File saved in correct location
3. Continue.dev reloaded
4. Server is running: `curl https://your-server:8443/health`

**View logs:**
- Continue sidebar → Settings → View Logs

### Connection Errors

**Error:** "Failed to connect to MCP server"

**Solutions:**
1. Verify server URL is correct
2. Check API key is valid (run `config show-api-key`)
3. Ensure firewall allows port 8443
4. For self-signed certificates, set `NODE_EXTRA_CA_CERTS` to the path of the server certificate copy

### Authentication Failed

**Error:** "Unauthorized"

**Solutions:**
1. Verify API key matches the one from `config show-api-key`
2. Check Bearer token format: `Authorization:Bearer API_KEY` (no space after colon)
3. Ensure no extra quotes around API key

### Tools Not Available

**Check:**
1. MCP server connection successful
2. Database connection working (check server logs)
3. PRTG API v2 configured (if using API-dependent features)

## Example Usage

### Query Critical Sensors

```
Continue: Show me all critical PRTG sensors
```

Continue will use `prtg_get_sensors` with status filter.

### Analyze Alerts

```
Continue: What are the current PRTG alerts and their impact?
```

Continue will use `prtg_get_alerts` and analyze results.

### Historical Analysis

```
Continue: Show sensor 12345 data from the last 24 hours
```

Continue will use `prtg_get_sensor_timeseries` with period `short`.

## Configuration Examples

### Project-Specific (Workspace)

Create `.continue/mcpServers/prtg.json` in your project:

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://dev-prtg.local:8443/mcp",
      "--header",
      "Authorization:Bearer ${DEV_PRTG_KEY}"
    ],
    "env": {
      "DEV_PRTG_KEY": "dev-api-key",
      "NODE_EXTRA_CA_CERTS": "/path/to/server.crt"
    }
  }
}
```

### Global (All Projects)

Create `~/.continue/mcpServers/prtg.json`:

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "-y",
      "mcp-remote",
      "https://prtg.company.com:8443/mcp",
      "--header",
      "Authorization:Bearer ${PRTG_API_KEY}"
    ],
    "env": {
      "PRTG_API_KEY": "production-key",
      "NODE_EXTRA_CA_CERTS": "/path/to/server.crt"
    }
  }
}
```

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - MCP Server PRTG configuration
- [Tools Reference](TOOLS.md) - Complete tools documentation
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
```
