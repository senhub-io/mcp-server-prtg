# Continue.dev Configuration

> ⚠️ **UNTESTED CONFIGURATION**
>
> This configuration is based on official Continue.dev MCP documentation but has **not been tested** with MCP Server PRTG.
>
> **Status**: Community contribution - feedback needed!
>
> If you test this configuration successfully (or encounter issues), please share feedback via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues). Your experience will help improve this guide!

Configure [Continue.dev](https://continue.dev/) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running
- Continue.dev extension installed in VS Code or JetBrains IDE
- Server API key from `config.yaml`

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
      "mcp-remote",
      "https://your-server:8443/mcp",
      "--header",
      "Authorization:Bearer ${PRTG_MCP_API_KEY}"
    ],
    "env": {
      "PRTG_MCP_API_KEY": "your-api-key-here"
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
      "mcp-remote",
      "https://prtg-mcp.company.com:8443/mcp",
      "--header",
      "Authorization:Bearer ${PRTG_MCP_API_KEY}"
    ],
    "env": {
      "PRTG_MCP_API_KEY": "production-api-key"
    }
  }
}
```

### Self-Signed Certificates

If using self-signed TLS certificates:

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "mcp-remote",
      "https://your-server:8443/mcp",
      "--header",
      "Authorization:Bearer YOUR_API_KEY"
    ],
    "env": {
      "NODE_TLS_REJECT_UNAUTHORIZED": "0"
    }
  }
}
```

⚠️ **Security Warning**: Only use `NODE_TLS_REJECT_UNAUTHORIZED=0` in development. Use proper certificates in production.

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

**PostgreSQL-based tools (12):**
- `prtg_get_sensors` - Query sensors with filters
- `prtg_get_alerts` - Get active alerts
- `prtg_get_devices` - List devices
- `prtg_get_groups` - List groups
- `prtg_get_probes` - List probes
- `prtg_get_channels` - Get sensor channels
- `prtg_get_tags` - List tags
- `prtg_get_sensor_history` - Historical data
- `prtg_get_notifications` - Notification settings
- `prtg_get_business_processes` - Business process status
- `prtg_get_statistics` - System statistics
- `prtg_custom_query` - Custom SQL queries (if enabled)

**PRTG API v2 tools (3):**
- `prtg_get_channel_current_values` - Real-time channel values
- `prtg_get_sensor_timeseries` - Time series data
- `prtg_get_sensor_history_custom` - Custom historical queries

## Troubleshooting

### MCP Server Not Appearing

**Check:**
1. JSON syntax is valid (use JSONLint.com)
2. File saved in correct location
3. Continue.dev reloaded
4. Server is running: `curl https://your-server:8443/health`

**View logs:**
- Continue sidebar → Settings → View Logs

### Connection Errors

**Error:** "Failed to connect to MCP server"

**Solutions:**
1. Verify server URL is correct
2. Check API key is valid (from `config.yaml`)
3. Ensure firewall allows port 8443
4. For self-signed certs, add `NODE_TLS_REJECT_UNAUTHORIZED=0`

### Authentication Failed

**Error:** "Unauthorized"

**Solutions:**
1. Verify API key matches `config.yaml`
2. Check Bearer token format: `Authorization:Bearer API_KEY` (no space after colon)
3. Ensure no extra quotes around API key

### Tools Not Available

**Check:**
1. MCP server connection successful
2. Database connection working (check server logs)
3. PRTG API v2 configured (if using API tools)

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

Continue will use `prtg_get_sensor_history` or PRTG API v2 tools.

## Configuration Examples

### Project-Specific (Workspace)

Create `.continue/mcpServers/prtg.json` in your project:

```json
{
  "prtg": {
    "command": "npx",
    "args": [
      "mcp-remote",
      "https://dev-prtg.local:8443/mcp",
      "--header",
      "Authorization:Bearer ${DEV_PRTG_KEY}"
    ],
    "env": {
      "DEV_PRTG_KEY": "dev-api-key"
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
      "mcp-remote",
      "https://prtg.company.com:8443/mcp",
      "--header",
      "Authorization:Bearer ${PRTG_API_KEY}"
    ],
    "env": {
      "PRTG_API_KEY": "production-key"
    }
  }
}
```

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - MCP Server PRTG configuration
- [Tools Reference](TOOLS.md) - Complete tools documentation
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues

## Feedback

This configuration is based on Continue.dev's MCP documentation. If you test it successfully (or encounter issues), please share feedback via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues).
