# Cline Configuration

Configure [Cline](https://github.com/cline/cline) (VS Code extension) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running
- VS Code installed
- Cline extension installed from VS Code Marketplace
- Server API key and client configuration from the administration page or `config client` command

## Getting the Ready-to-Paste Configuration

On the server, open the administration page at `https://localhost:<port>/admin/` (8443 by default), sign in with the MCP API key, and open the **AI clients** tab. Copy the Cline/VS Code block with the Copy button.

Alternatively, run on the server:

```bash
mcp-server-prtg config client
```

This prints the configurations for Claude Desktop, Claude Code, Cursor and VS Code.

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
        "-y",
        "mcp-remote",
        "https://your-server:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "your-api-key",
        "NODE_EXTRA_CA_CERTS": "/path/to/server.crt"
      }
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
        "-y",
        "mcp-remote",
        "https://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7",
        "NODE_EXTRA_CA_CERTS": "C:\\path\\to\\server.crt"
      }
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
        "-y",
        "mcp-remote",
        "https://prtg-mcp.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "your-api-key-here",
        "NODE_EXTRA_CA_CERTS": "/home/user/certs/server.crt"
      }
    }
  }
}
```

### Self-Signed Certificates

The server uses a self-signed certificate (`server.crt` next to `config.yaml`). Do not disable certificate verification. Instead, trust the certificate:

1. Copy `server.crt` from the server to the client computer
2. Set `NODE_EXTRA_CA_CERTS` to the path of that file (as shown above)

Alternatively, import `server.crt` into the computer's trusted root certificates (Windows: `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`; macOS: Keychain Access, "Always Trust").

## Setup Steps

### 1. Install Cline Extension

In VS Code:
1. Open Extensions (`Ctrl+Shift+X` / `Cmd+Shift+X`)
2. Search for "Cline"
3. Click Install

### 2. Get Your API Key

On the server:

```bash
# Windows
"C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"

# Linux
sudo mcp-server-prtg config show-api-key --config /etc/mcp-server-prtg/config.yaml
```

Or copy the key from the **AI clients** tab of the administration page.

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
- `prtg_get_statistics` - System statistics
- `prtg_query_sql` - Custom SQL queries (only if `allow_custom_queries: true`)

**PRTG API v2 / database metrics (2 tools, when the database holds channel history or the PRTG API is configured):**
- `prtg_get_channel_current_values` - Current channel values (Source line indicates database or PRTG API; optional `source` parameter: `auto`, `database`, `prtg`)
- `prtg_get_sensor_timeseries` - Time series data for a predefined period (`live`, `short`, `medium`, `long`)

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
2. Check API key matches the one from `config show-api-key`
3. Ensure firewall allows port 8443
4. Verify `NODE_EXTRA_CA_CERTS` points to a valid copy of `server.crt`

**Check server:**
```bash
# Verify server is listening
curl https://localhost:8443/health

# Check server status
mcp-server-prtg status
```

### Authentication Errors

**Error:** "Unauthorized" or "401"

**Check:**
1. API key is correct (from `config show-api-key`)
2. Bearer token format: `Authorization:Bearer ${MCP_PRTG_API_KEY}` (no space after colon)
3. No extra quotes or whitespace

### Tools Not Available

**Check:**
1. Server connected (green indicator in Cline)
2. Database connection works (server logs)
3. PRTG API v2 configured or channel history present (for metrics tools)

**View server logs:**
```bash
# Linux
tail -f /var/log/mcp-server-prtg/mcp-server-prtg.log

# Windows
type "C:\ProgramData\Sensor Factory\MCP Server PRTG\logs\mcp-server-prtg.log"
```

## Example Usage in Cline

### Autonomous Analysis

Cline is an autonomous coding agent. You can ask it to:

```
Analyze PRTG alerts and create a report with recommendations
```

Cline will:
1. Query alerts using `prtg_get_alerts`
2. Get device details using `prtg_device_overview`
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
3. Each user updates with their API key and certificate path

**Example shareable config:**
```json
{
  "mcpServers": {
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
        "MCP_PRTG_API_KEY": "REPLACE_WITH_YOUR_KEY",
        "NODE_EXTRA_CA_CERTS": "REPLACE_WITH_CERT_PATH"
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
        "-y",
        "mcp-remote",
        "https://prtg-prod.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_PROD_KEY}"
      ],
      "env": {
        "PRTG_PROD_KEY": "prod-key",
        "NODE_EXTRA_CA_CERTS": "/certs/prod-server.crt"
      }
    },
    "prtg-dev": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "https://prtg-dev.local:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_DEV_KEY}"
      ],
      "env": {
        "PRTG_DEV_KEY": "dev-key",
        "NODE_EXTRA_CA_CERTS": "/certs/dev-server.crt"
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
        "-y",
        "mcp-remote",
        "https://your-server:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "your-key",
        "NODE_EXTRA_CA_CERTS": "/path/to/server.crt",
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
3. **Trust the server certificate** via `NODE_EXTRA_CA_CERTS` or the system trust store; never disable certificate verification
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
```
