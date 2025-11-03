# Cursor IDE Configuration

> ⚠️ **Configuration Status**: Based on official Cursor MCP documentation. Not yet tested with MCP Server PRTG. Feedback welcome!

Configure [Cursor IDE](https://cursor.com/) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running
- Cursor IDE installed
- Server API key from `config.yaml`

## Configuration File Location

Cursor uses an `mcp.json` file for MCP server configuration:

**Project-specific:** `.cursor/mcp.json` (in your project directory)
**Global:** `~/.cursor/mcp.json` (in your home directory)

## Basic Configuration

Create or edit `~/.cursor/mcp.json`:

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

### Multiple Environments

Configure different servers for different projects:

**Production** (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "prtg-prod": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg.company.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_PROD_KEY}"
      ],
      "env": {
        "PRTG_PROD_KEY": "production-key"
      }
    }
  }
}
```

**Development** (`.cursor/mcp.json` in project):
```json
{
  "mcpServers": {
    "prtg-dev": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://dev-prtg.local:8443/mcp",
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

## Setup Steps

### 1. Get Your API Key

Your API key is auto-generated during installation:

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

### 2. Create Configuration File

```bash
# Global configuration
mkdir -p ~/.cursor
nano ~/.cursor/mcp.json

# Or project-specific
mkdir -p .cursor
nano .cursor/mcp.json
```

### 3. Add Configuration

Paste your MCP server configuration (see examples above).

### 4. Restart Cursor

Close and reopen Cursor IDE to load the new configuration.

## Verification

### Check Configuration

1. Open Cursor
2. Open Cursor Settings
3. Look for MCP section (may be in AI/Extensions settings)
4. Verify "prtg" server appears

### Test Connection

In Cursor's chat interface, try:

```
Show me all critical PRTG sensors
```

Cursor should use the MCP tools to query your PRTG data.

## Available Tools

Once connected, Cursor will have access to all MCP Server PRTG tools:

**PostgreSQL-based (12 tools):**
- Sensors, alerts, devices, groups, probes
- Channels, tags, history, notifications
- Business processes, statistics
- Custom SQL queries (if enabled)

**PRTG API v2 (3 tools):**
- Real-time channel values
- Time series data
- Custom historical queries

See [TOOLS.md](TOOLS.md) for complete documentation.

## Troubleshooting

### Configuration Not Loaded

**Solutions:**
1. Check JSON syntax (use JSONLint.com)
2. Verify file location: `~/.cursor/mcp.json` or `.cursor/mcp.json`
3. Restart Cursor completely
4. Check Cursor logs (Help → Show Logs)

### Connection Failed

**Error:** "Cannot connect to MCP server"

**Check:**
1. Server is running: `curl https://your-server:8443/health`
2. API key is correct
3. Firewall allows port 8443
4. URL format is correct

**View server logs:**
```bash
# Linux
tail -f /opt/mcp-server-prtg/logs/mcp-server-prtg.log

# Windows
type logs\mcp-server-prtg.log
```

### Authentication Errors

**Error:** "Unauthorized" or "403 Forbidden"

**Solutions:**
1. Verify API key from `config.yaml`
2. Check Bearer token format in configuration
3. Ensure no typos in header name: `Authorization:Bearer`

### Tools Not Appearing

**Check:**
1. MCP server connection is active
2. Database connection works (check server status)
3. PRTG API v2 configured (for API tools)

```bash
# Check server status
./mcp-server-prtg status
```

### mcp-remote Issues

**Error:** "npx: command not found"

**Solution:** Install Node.js and npm:
```bash
# macOS
brew install node

# Ubuntu/Debian
sudo apt install nodejs npm

# Windows
# Download from nodejs.org
```

**Error:** "mcp-remote not found"

**Solution:** Install explicitly:
```bash
npm install -g mcp-remote
```

Then update configuration:
```json
{
  "command": "mcp-remote",
  "args": ["https://your-server:8443/mcp", ...]
}
```

## Example Usage in Cursor

### Query Sensors

```
Cursor Chat: Show me all down sensors in PRTG
```

Cursor uses `prtg_get_sensors` with status filter.

### Analyze Trends

```
Cursor Chat: Analyze the bandwidth usage trend for sensor 12345 over the last week
```

Cursor uses `prtg_get_sensor_history` or PRTG API v2 tools.

### Generate Reports

```
Cursor Chat: Create a summary of all critical alerts with their devices and groups
```

Cursor combines multiple tools (`prtg_get_alerts`, `prtg_get_devices`, `prtg_get_groups`).

## Configuration Best Practices

### Security

1. **Use environment variables** for API keys
2. **Use proper TLS certificates** in production
3. **Don't commit** `mcp.json` with secrets to git

Add to `.gitignore`:
```
.cursor/mcp.json
```

### Organization

**Global servers** (`~/.cursor/mcp.json`):
- Production PRTG servers
- Shared infrastructure

**Project-specific** (`.cursor/mcp.json`):
- Development/staging servers
- Project-specific configurations

## Advanced Configuration

### With HTTP Proxy

If behind a corporate proxy:

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

### With Custom Timeout

For slow networks:

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://your-server:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}",
        "--timeout",
        "60000"
      ],
      "env": {
        "PRTG_API_KEY": "your-key"
      }
    }
  }
}
```

## Transport Details

Cursor supports multiple MCP transport types:
- **STDIO**: Local processes
- **SSE**: Server-Sent Events
- **Streamable HTTP**: Our recommended approach (via mcp-remote)

MCP Server PRTG uses **Streamable HTTP**, which `mcp-remote` handles automatically.

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Server configuration
- [Tools Reference](TOOLS.md) - Available MCP tools
- [Usage Guide](USAGE.md) - Examples and best practices
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues

## Feedback

This configuration is based on Cursor's official MCP documentation. If you test it successfully (or encounter issues), please share feedback via [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues).
