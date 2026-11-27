# Claude Desktop Configuration

This guide explains how to configure Claude Desktop to connect to your MCP Server PRTG instance using `mcp-remote`.

## Architecture Overview

```
Claude Desktop → mcp-remote → MCP Server PRTG → PostgreSQL (PRTG Data Exporter target database)
                                                → PRTG API v2 (PRTG Core Server, when needed)
```

**Important:**
- MCP Server PRTG must be running as a service or daemon
- Claude Desktop connects via HTTP/HTTPS using `mcp-remote`
- Alternatively, Claude Desktop can start MCP Server PRTG locally in stdio mode (see [stdio Mode](#stdio-mode-without-a-server)). stdio mode uses PRTG API v2 only, without PostgreSQL
- Metrics tools read the database first and return a `Source:` line indicating whether the answer comes from the database or the PRTG API

## Getting the Ready-to-Paste Configuration

The easiest way to get the correct configuration block is to use the administration page or the command line:

- **Administration page**: open `https://localhost:<port>/admin/` (8443 by default), sign in with the MCP API key, go to the **AI clients** tab and copy the Claude Desktop block.
- **Command line**: run `mcp-server-prtg config client --config <path>` (or without `--config` to use the installed location). It prints the configurations for Claude Desktop, Claude Code, Cursor and VS Code.

## Configuration File Location

### Windows
```
%APPDATA%\Claude\claude_desktop_config.json
```
Full path: `C:\Users\YourUsername\AppData\Roaming\Claude\claude_desktop_config.json`

### macOS
```
~/Library/Application Support/Claude/claude_desktop_config.json
```

### Linux
```
~/.config/Claude/claude_desktop_config.json
```

## Basic Configuration

### HTTPS with TLS (Recommended for Production)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "https://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "your-mcp-server-api-key",
        "NODE_EXTRA_CA_CERTS": "C:\\path\\to\\server.crt"
      }
    }
  }
}
```

**Replace:**
- `<MCP_SERVER_HOST>`: IP or hostname where MCP Server PRTG is running
- `your-mcp-server-api-key`: API key from MCP Server PRTG (see [Finding Your MCP Server API Key](#finding-your-mcp-server-api-key))
- `C:\path\to\server.crt`: path to a copy of the server certificate (`server.crt` from the `certs` folder next to `config.yaml` on the server)

**Trusting the server certificate:** MCP Server PRTG uses a self-signed certificate that covers `localhost`, the machine name and its IP addresses. Copy `server.crt` to the client computer and point `NODE_EXTRA_CA_CERTS` to it. Do not disable certificate verification.

### HTTP without TLS (Development/Testing Only)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "http://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "your-mcp-server-api-key"
      }
    }
  }
}
```

**Warning:** HTTP without TLS should only be used for local development/testing.

## stdio Mode (Without a Server)

In stdio mode, Claude Desktop starts MCP Server PRTG as a local process (`npx -y @senhub-io/mcp-server-prtg`). No service, API key or TLS certificate is needed. stdio mode talks to PRTG API v2 only: there is no PostgreSQL access, so `prtg_query_sql` is not available (13 tools instead of 14).

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": ["-y", "@senhub-io/mcp-server-prtg"],
      "env": {
        "PRTG_URL": "https://prtg.example.com:1616",
        "PRTG_API_TOKEN": "your-prtg-api-v2-token"
      }
    }
  }
}
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PRTG_URL` | Yes | - | PRTG Core Server URL with API v2 port (e.g., `https://prtg.example.com:1616`) |
| `PRTG_API_TOKEN` | Yes | - | PRTG API v2 token (here the PRTG token is the right one) |
| `PRTG_VERIFY_SSL` | No | `true` | Verify the PRTG server certificate |
| `PRTG_TIMEOUT` | No | `30` | Request timeout in seconds |
| `PRTG_MAX_CONCURRENT` | No | `2` | Maximum PRTG API requests in flight |
| `PRTG_CACHE_DISABLED` | No | `false` | Disable the response cache |
| `LOG_LEVEL` | No | `info` | Log level (`debug`, `info`, `warn`, `error`) |

See [npm-package/README.md](../npm-package/README.md) for details.

## Configuration Examples

### Example 1: Local Server (Same Machine)

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
        "MCP_PRTG_API_KEY": "abc123def456ghi789",
        "NODE_EXTRA_CA_CERTS": "C:\\ProgramData\\Sensor Factory\\MCP Server PRTG\\certs\\server.crt"
      }
    }
  }
}
```

### Example 2: Remote Server with Custom Port

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "https://prtg-mcp.company.com:9443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "prod-api-key-xyz",
        "NODE_EXTRA_CA_CERTS": "/home/user/certs/server.crt"
      }
    }
  }
}
```

### Example 3: Multiple MCP Servers

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "https://prtg-server.local:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "prtg-server-key",
        "NODE_EXTRA_CA_CERTS": "C:\\certs\\server.crt"
      }
    },
    "other-mcp": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "https://other-server.local:8080/mcp",
        "--header",
        "Authorization:Bearer ${OTHER_API_KEY}"
      ],
      "env": {
        "OTHER_API_KEY": "other-server-key"
      }
    }
  }
}
```

## Finding Your MCP Server API Key

The API key is stored encrypted in the MCP Server PRTG `config.yaml` file. To print the decrypted key:

```bash
# Windows
"C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"

# Linux
sudo mcp-server-prtg config show-api-key --config /etc/mcp-server-prtg/config.yaml
```

Without `--config`, the command uses the installed configuration location (ProgramData on Windows, `/etc/mcp-server-prtg` on Linux).

**Important:** This is the MCP Server authentication key, NOT the PRTG API v2 token!

### Key Distinction

```yaml
server:
  api_key: "enc:dpapi:..."    # ← THIS is for Claude Desktop (MCP_PRTG_API_KEY), stored encrypted

prtg:
  api_token: "enc:aes:..."    # ← This is for PRTG Core Server (internal use only)
```

## Verifying Configuration

### Step 1: Verify MCP Server is Running

```bash
# Test health endpoint
curl https://localhost:8443/health

# Expected response:
{"status":"ok"}
```

### Step 2: Test Authentication

```bash
# Test with your API key
curl -H "Authorization: Bearer YOUR_MCP_SERVER_API_KEY" \
     https://localhost:8443/status

# Expected response:
{
  "version": "2.0.0",
  "transport": "streamable-http",
  "protocol": "2025-03-26",
  "uptime": "1m30s",
  "database": "connected"
}
```

### Step 3: Restart Claude Desktop

After updating the configuration:

1. **Quit Claude Desktop completely** (not just close the window)
2. **Restart Claude Desktop**
3. Open a new conversation
4. Test with: "List available PRTG tools"

## Troubleshooting

### "Connection Refused" Error

**Causes:**
- MCP Server PRTG is not running
- Wrong hostname or port
- Firewall blocking connection

**Solution:**
```bash
# Verify server is running
curl https://localhost:8443/health

# Check MCP server logs
tail -f logs/mcp-server-prtg.log

# Verify server is listening
netstat -an | grep 8443
```

### "Unauthorized" Error

**Causes:**
- Wrong API key
- Using PRTG API v2 token instead of MCP Server API key
- Missing Authorization header

**Solution:**
- Verify you're using the key printed by `config show-api-key`
- NOT the `prtg.api_token`
- Check for extra spaces in the key

### "TLS Certificate Error"

**Causes:**
- Self-signed certificate not trusted
- Certificate expired
- Hostname mismatch

**Solutions:**

Copy `server.crt` from the server's `certs` folder to the client computer and set `NODE_EXTRA_CA_CERTS` to its path. Alternatively, import the certificate into the computer's trusted root certificates (Windows: `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`; macOS: Keychain Access, "Always Trust"). Do not set `NODE_TLS_REJECT_UNAUTHORIZED=0`.

**For Production:**
- Use a certificate from your company PKI (configure `server.cert_file` / `server.key_file` on the server)
- Or use HTTP (local network only)

### "Cannot find module 'mcp-remote'"

**Cause:** `mcp-remote` npm package not installed

**Solution:**
```bash
# Install globally
npm install -g mcp-remote

# Or let npx handle it (slower first run)
# npx will auto-download on first use
```

### Server Not Appearing in Claude Desktop

**Solutions:**
1. Verify JSON syntax (use [jsonlint.com](https://jsonlint.com))
2. Ensure file is named exactly `claude_desktop_config.json`
3. Check file permissions (must be readable)
4. Restart Claude Desktop completely
5. Check Claude Desktop logs for errors

## Security Best Practices

1. **Never commit API keys to version control**
   - Use environment variables
   - Add `claude_desktop_config.json` to `.gitignore`

2. **Use HTTPS in production**
   - Trust the server certificate via `NODE_EXTRA_CA_CERTS` or the system trust store
   - Or use a certificate from your company PKI

3. **Rotate API keys regularly**
   - Regenerate keys periodically
   - Update all clients

4. **Restrict network access**
   - Use firewall rules
   - Bind to specific interfaces only

5. **Monitor server logs**
   - Watch for failed authentication attempts
   - Set up alerts for unusual activity

6. **Keep software updated**
   - Update MCP Server PRTG regularly
   - Update `mcp-remote` package

## Quick Setup Guide

### 1. Start MCP Server PRTG

```bash
# Windows (as service)
mcp-server-prtg.exe start

# Linux (as daemon)
sudo systemctl start mcp-server-prtg

# Or run in foreground for testing
./mcp-server-prtg run
```

### 2. Get Your MCP Server API Key

```bash
# Windows (PowerShell)
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"

# Linux
sudo mcp-server-prtg config show-api-key --config /etc/mcp-server-prtg/config.yaml
```

### 3. Get the Client Configuration

```bash
# Windows
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config client --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"

# Linux
sudo mcp-server-prtg config client --config /etc/mcp-server-prtg/config.yaml
```

Or open `https://localhost:8443/admin/`, sign in, and copy the block from the **AI clients** tab.

### 4. Create Claude Desktop Config

**Windows:**
```powershell
# Create directory
New-Item -ItemType Directory -Force -Path "$env:APPDATA\Claude"

# Edit config
notepad "$env:APPDATA\Claude\claude_desktop_config.json"
```

**macOS:**
```bash
# Create directory
mkdir -p ~/Library/Application\ Support/Claude

# Edit config
nano ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Linux:**
```bash
# Create directory
mkdir -p ~/.config/Claude

# Edit config
nano ~/.config/Claude/claude_desktop_config.json
```

### 5. Paste Configuration

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
        "MCP_PRTG_API_KEY": "PASTE_YOUR_KEY_HERE",
        "NODE_EXTRA_CA_CERTS": "C:\\path\\to\\server.crt"
      }
    }
  }
}
```

Replace `PASTE_YOUR_KEY_HERE` with your MCP Server API key from step 2, and set `NODE_EXTRA_CA_CERTS` to the path of the copied `server.crt`.

### 6. Test Configuration

1. Save the file
2. Quit Claude Desktop completely
3. Restart Claude Desktop
4. Open new conversation
5. Ask: "What PRTG tools are available?"

## Common Mistakes

### ❌ Wrong: Using PRTG API v2 Token

```json
"env": {
  "MCP_PRTG_API_KEY": "5SIPLYZQND7TS4C4G32AVLNPS2XR6XWD64AOT7UYWE======"
}
```

This is the PRTG API v2 token (from PRTG Core Server). It won't work!

### ✅ Correct: Using MCP Server API Key

```json
"env": {
  "MCP_PRTG_API_KEY": "abc123def456ghi789"
}
```

This is the key printed by `config show-api-key`.

### ❌ Wrong: Connecting to PRTG Core Server

```json
"mcp-remote",
"https://prtg.example.com:443/mcp"
```

This tries to connect to PRTG Core Server, not MCP Server PRTG!

### ✅ Correct: Connecting to MCP Server PRTG

```json
"mcp-remote",
"https://mcp-server-host:8443/mcp"
```

This connects to where MCP Server PRTG is running.

## Support

For issues or questions:

1. Check MCP server logs: `logs/mcp-server-prtg.log`
2. Verify server health: `curl https://localhost:8443/health`
3. Test authentication: `curl -H "Authorization: Bearer KEY" https://localhost:8443/status`
4. Review: [docs/TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
5. Open [GitHub issue](https://github.com/senhub-io/mcp-server-prtg/issues) with logs (redact API keys!)

---

**Last Updated**: 2026-01-15
**Protocol**: Streamable HTTP (MCP 2025-03-26)
**Endpoint**: `/mcp`
**Client**: `mcp-remote` (npm package)
```
