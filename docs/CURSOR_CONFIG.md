# Cursor IDE Configuration

Configure [Cursor IDE](https://cursor.com/) to connect to your MCP Server PRTG instance.

## Prerequisites

- MCP Server PRTG installed and running (version 2.0.0 or later)
- Cursor IDE installed
- The MCP API key of the server

## Getting the Ready-to-Paste Configuration

The fastest way to configure Cursor is to copy a ready-made block:

- On the server, open the administration page at `https://localhost:<port>/admin/` (8443 by default), sign in with the MCP API key, and open the **AI clients** tab. It shows the Cursor configuration with a Copy button.
- Or run on the server: `mcp-server-prtg config client` (add `--host <name>` to use a different host name than the one detected). It prints the configurations for Claude Desktop, Claude Code, Cursor and VS Code.

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
      "url": "https://your-server:8443/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_KEY"
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
      "url": "https://localhost:8443/mcp",
      "headers": {
        "Authorization": "Bearer a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
      }
    }
  }
}
```

### Remote Server

```json
{
  "mcpServers": {
    "prtg": {
      "url": "https://prtg-mcp.company.com:8443/mcp",
      "headers": {
        "Authorization": "Bearer a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
      }
    }
  }
}
```

### Multiple Environments

Configure different servers for different projects:

**Production** (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "prtg-prod": {
      "url": "https://prtg.company.com:8443/mcp",
      "headers": {
        "Authorization": "Bearer production-key"
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
      "url": "https://dev-prtg.local:8443/mcp",
      "headers": {
        "Authorization": "Bearer dev-key"
      }
    }
  }
}
```

## Server Certificate (TLS)

MCP Server PRTG uses HTTPS with a self-signed certificate by default (`certs/server.crt` next to `config.yaml`). Cursor must trust this certificate; do not disable certificate verification.

Copy `server.crt` from the server to each client computer and import it into the computer's trusted root certificates:

- **Windows:** `certlm.msc` → Trusted Root Certification Authorities, or `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`
- **macOS:** Keychain Access, then set the certificate to "Always Trust"

A certificate from your company PKI can be used instead (server settings `cert_file` / `key_file`).

## Setup Steps

### 1. Get Your API Key

The API key is generated during installation and stored encrypted in `config.yaml`. Print it on the server with:

```bash
# Windows
"C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"

# Linux
sudo mcp-server-prtg config show-api-key --config /etc/mcp-server-prtg/config.yaml
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

Paste the Cursor block from the administration page **AI clients** tab or from `config client`, or write it manually (see examples above).

### 4. Restart Cursor

Close and reopen Cursor IDE to load the new configuration.

## Verification

### Check Configuration

1. Open Cursor
2. Open Cursor Settings
3. Look for the MCP section
4. Verify the "prtg" server appears and is connected

### Test Connection

In Cursor's chat interface, try:

```
Show me all critical PRTG sensors
```

Cursor should use the MCP tools to query your PRTG data.

## Available Tools

Once connected, Cursor has access to the MCP Server PRTG tools:

**Database tools (PostgreSQL target database of PRTG Data Exporter):**
- Sensors, sensor status, alerts, device overview, top sensors
- Hierarchy, search, groups, tags
- Business processes, statistics
- Custom SQL queries (only if `allow_custom_queries: true`)

**Metrics tools (database first):**
- `prtg_get_channel_current_values` and `prtg_get_sensor_timeseries` read the channel history in the database first; each answer starts with a `Source:` line. The optional `source` parameter selects `auto` (default), `database` or `prtg` (real time from the PRTG API v2 when configured).

See [TOOLS.md](TOOLS.md) for complete documentation.

## Troubleshooting

### Configuration Not Loaded

**Solutions:**
1. Check JSON syntax
2. Verify file location: `~/.cursor/mcp.json` or `.cursor/mcp.json`
3. Restart Cursor completely
4. Check Cursor logs (Help → Show Logs)

### Connection Failed

**Error:** "Cannot connect to MCP server"

**Check:**
1. Server is running: `mcp-server-prtg status`
2. API key is correct (`config show-api-key`)
3. Firewall allows the MCP port (8443 by default)
4. URL format is correct (`https://host:port/mcp`)

**View server logs:**
```bash
# Linux
tail -f /var/log/mcp-server-prtg/mcp-server-prtg.log

# Windows
type "C:\ProgramData\Sensor Factory\MCP Server PRTG\logs\mcp-server-prtg.log"
```

### Certificate Errors

**Error:** TLS or certificate verification failure

**Solution:** Import the server certificate (`server.crt`) into the trusted root certificates of the client computer (see "Server Certificate (TLS)" above). Do not disable certificate verification.

### Authentication Errors

**Error:** "Unauthorized" or "403 Forbidden"

**Solutions:**
1. Verify the API key with `config show-api-key`
2. Check the header format: `"Authorization": "Bearer <key>"`
3. Ensure there are no typos in the header name

### Tools Not Appearing

**Check:**
1. MCP server connection is active in Cursor settings
2. Database connection works (check the administration page Overview tab)
3. The metrics tools appear when the database holds channel history or the PRTG API v2 is configured

## Example Usage in Cursor

### Query Sensors

```
Cursor Chat: Show me all down sensors in PRTG
```

Cursor uses `prtg_get_sensors` with a status filter.

### Analyze Trends

```
Cursor Chat: Analyze the bandwidth usage trend for sensor 12345 over the last week
```

Cursor uses `prtg_get_sensor_timeseries` with period `medium`.

### Generate Reports

```
Cursor Chat: Create a summary of all critical alerts with their devices and groups
```

Cursor combines multiple tools (`prtg_get_alerts`, `prtg_device_overview`, `prtg_get_groups`).

## Configuration Best Practices

### Security

1. **Keep the API key out of shared files** — use the value from `config show-api-key` only on the client machine that needs it
2. **Trust the server certificate** instead of disabling verification
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

## Transport Details

MCP Server PRTG uses **Streamable HTTP** when it runs as a service; Cursor connects to it directly with the `url` and `headers` fields shown above.

## Next Steps

- [Configuration Guide](CONFIGURATION.md) - Server configuration
- [Tools Reference](TOOLS.md) - Available MCP tools
- [Usage Guide](USAGE.md) - Examples and best practices
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues
