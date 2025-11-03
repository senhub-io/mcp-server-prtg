# Cline Configuration

> ⚠️ **UNTESTED CONFIGURATION**
>
> This configuration is based on official Cline MCP documentation but has **not been tested** with MCP Server PRTG.
>
> **Status**: Community contribution - feedback needed!
>
> **Known Issue**: Cline v3.17.5+ has reported HTTP transport regressions. Configuration may require workarounds.
>
> If you test this configuration successfully (or encounter issues), please share feedback via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues). Your experience will help improve this guide!

Configure [Cline](https://github.com/cline/cline) (VS Code extension) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running
- VS Code installed
- Cline extension installed from VS Code Marketplace
- Server API key from `config.yaml`

## Configuration Location

Cline uses `cline_mcp_settings.json` for MCP server configuration:

**Location:** `~/Documents/Cline/MCP/cline_mcp_settings.json`

**Access via UI:**
1. Open Cline panel in VS Code
2. Click "MCP Servers" icon (top navigation)
3. Select "Installed" tab
4. Click "Configure MCP Servers"

## Basic Configuration

```json
{
  "mcpServers": {
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
}
```

## Configuration Examples

### Local Server

```json
{
  "mcpServers": {
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
}
```

### Remote Server with Environment Variable

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-mcp.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

### Self-Signed Certificates

For development with self-signed certificates:

```json
{
  "mcpServers": {
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
}
```

⚠️ **Security Warning**: Only disable certificate validation in development environments.

## Setup Steps

### 1. Install Cline Extension

In VS Code:
1. Open Extensions (`Ctrl+Shift+X` / `Cmd+Shift+X`)
2. Search for "Cline"
3. Click Install

### 2. Get Your API Key

```bash
# Linux/macOS
cat config.yaml | grep api_key

# Windows (PowerShell)
Select-String -Path config.yaml -Pattern "api_key"
```

Output:
```yaml
api_key: "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
```

### 3. Configure via UI

1. Open Cline panel in VS Code (click Cline icon in sidebar)
2. Click "MCP Servers" icon at top
3. Select "Installed" tab
4. Click "Configure MCP Servers"
5. Paste your configuration JSON
6. Save

### 4. Verify Configuration

After saving:
1. Cline panel should show "prtg" server
2. Status indicator should be green
3. Tools/resources should be listed

## Verification

### Check Server Status

In Cline panel:
- **MCP Servers** section shows configured servers
- **Green indicator** = connected
- **Red indicator** = connection error
- Click server to see **tools, resources, and logs**

### Test Tools

In Cline chat, ask:

```
Show me all critical PRTG sensors
```

Cline should use `prtg_get_sensors` tool automatically.

## Available Tools

Once connected, Cline will have access to all MCP Server PRTG tools:

**PostgreSQL-based (12 tools):**
- `prtg_get_sensors` - Query sensors
- `prtg_get_alerts` - Active alerts
- `prtg_get_devices` - Devices list
- `prtg_get_groups` - Groups hierarchy
- `prtg_get_probes` - Probes list
- `prtg_get_channels` - Sensor channels
- `prtg_get_tags` - Tags list
- `prtg_get_sensor_history` - Historical data
- `prtg_get_notifications` - Notifications
- `prtg_get_business_processes` - Business processes
- `prtg_get_statistics` - System stats
- `prtg_custom_query` - Custom SQL (if enabled)

**PRTG API v2 (3 tools):**
- `prtg_get_channel_current_values` - Real-time data
- `prtg_get_sensor_timeseries` - Time series
- `prtg_get_sensor_history_custom` - Custom queries

See [TOOLS.md](TOOLS.md) for complete documentation.

## Troubleshooting

### Server Not Appearing

**Check:**
1. JSON syntax is valid
2. Configuration saved correctly
3. Cline extension reloaded (reload VS Code window)
4. Server is running: `curl https://your-server:8443/health`

**View Cline logs:**
1. Cline panel → MCP Servers
2. Click on "prtg" server
3. View error logs in details panel

### Connection Failed

**Error:** "Failed to connect"

**Solutions:**
1. Verify server URL is correct
2. Check API key matches `config.yaml`
3. Ensure firewall allows port 8443
4. For self-signed certs, add `NODE_TLS_REJECT_UNAUTHORIZED=0`

**Check server:**
```bash
# Verify server is listening
curl https://localhost:8443/health

# Check server status
./mcp-server-prtg status
```

### Known Issue: HTTP Transport Regression (v3.17.5+)

**Symptom:** Authorization headers not sent, StreamableHttpTransport errors

**Workaround 1** - Downgrade Cline:
1. VS Code → Extensions
2. Right-click Cline → "Install Another Version"
3. Select v3.17.4 or earlier

**Workaround 2** - Use mcp-remote@next:

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote@next",
        "https://your-server:8443/mcp",
        "--header",
        "Authorization:Bearer YOUR_API_KEY"
      ]
    }
  }
}
```

**Monitor issue:** [Cline Issue #4391](https://github.com/cline/cline/issues/4391)

### Authentication Errors

**Error:** "Unauthorized" or "401"

**Check:**
1. API key is correct (copy-paste from `config.yaml`)
2. Bearer token format: `Authorization:Bearer API_KEY` (no space after colon)
3. No extra quotes or whitespace

### Tools Not Available

**Check:**
1. Server connected (green indicator in Cline)
2. Database connection works (server logs)
3. PRTG API v2 configured (for API tools)

**View server logs:**
```bash
# Linux/macOS
tail -f /opt/mcp-server-prtg/logs/mcp-server-prtg.log

# Windows
type logs\mcp-server-prtg.log
```

## Example Usage in Cline

### Autonomous Analysis

Cline is an autonomous coding agent. You can ask it to:

```
Analyze PRTG alerts and create a report with recommendations
```

Cline will:
1. Query alerts using `prtg_get_alerts`
2. Get device details using `prtg_get_devices`
3. Analyze patterns
4. Create a markdown report
5. Save to file (with your permission)

### Monitoring Scripts

```
Create a Python script that monitors critical sensors and sends Slack notifications
```

Cline will:
1. Query sensor structure
2. Write Python script using PRTG tools
3. Add error handling
4. Create requirements.txt
5. Test the script (with permission)

### Historical Analysis

```
Compare bandwidth usage between sensors 12345 and 12346 over the last month
```

Cline will:
1. Query historical data for both sensors
2. Analyze trends
3. Create comparison charts
4. Generate insights

## Sharing Custom Servers

Cline allows sharing MCP server configurations:

**Share location:** `~/Documents/Cline/MCP/`

To share your PRTG configuration:
1. Save `prtg-server.json` in `~/Documents/Cline/MCP/`
2. Other team members can copy it to their Cline directory
3. Each user updates with their API key

**Example shareable config:**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-mcp.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "REPLACE_WITH_YOUR_KEY"
      }
    }
  }
}
```

## Advanced Configuration

### Multiple PRTG Environments

```json
{
  "mcpServers": {
    "prtg-prod": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-prod.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_PROD_KEY}"
      ],
      "env": {
        "PRTG_PROD_KEY": "prod-key"
      }
    },
    "prtg-dev": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-dev.local:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_DEV_KEY}"
      ],
      "env": {
        "PRTG_DEV_KEY": "dev-key"
      }
    }
  }
}
```

### With HTTP Proxy

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://your-server:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "your-key",
        "HTTP_PROXY": "http://proxy.company.com:8080",
        "HTTPS_PROXY": "http://proxy.company.com:8080"
      }
    }
  }
}
```

## Configuration Best Practices

### Security

1. **Use environment variables** for API keys
2. **Don't commit** configurations with secrets
3. **Use proper TLS** in production
4. **Rotate API keys** periodically

### Organization

**Team configs:**
- Store template in shared location
- Each user personalizes with their key
- Document server URLs in team wiki

**Testing:**
- Use separate dev server
- Test configuration before deploying
- Monitor Cline logs for issues

## Debugging

### Enable Debug Logging

1. Open VS Code Settings
2. Search "Cline"
3. Enable "Debug Mode"
4. Reload window

### View Detailed Logs

In Cline panel:
1. MCP Servers → Click server
2. View "Error Logs" tab
3. Check for connection/authentication errors

### Test Server Directly

```bash
# Test server health
curl https://your-server:8443/health

# Test MCP endpoint (expect JSON-RPC response)
curl -X POST https://your-server:8443/mcp \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "method": "ping", "id": 1}'
```

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Server configuration
- [Tools Reference](TOOLS.md) - Available MCP tools
- [Usage Guide](USAGE.md) - Examples and best practices
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues

## Feedback

This configuration is based on Cline's official MCP documentation. If you test it successfully (or encounter issues), please share feedback via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues).

**Note:** Cline v3.17.5+ has known HTTP transport issues. If you encounter problems, try the workarounds listed above or provide feedback to help improve this guide.
