# Claude Desktop Configuration

This guide explains how to configure Claude Desktop to connect to your MCP Server PRTG instance using `mcp-remote`.

## Architecture Overview

```
Claude Desktop → mcp-remote → MCP Server PRTG → PostgreSQL (PRTG Data Exporter)
                                                → PRTG API v2 (PRTG Core Server)
```

**Important:**
- MCP Server PRTG must be running as a service or daemon
- Claude Desktop connects via HTTP/HTTPS using `mcp-remote`
- You cannot use "stdio mode" because MCP Server PRTG requires PostgreSQL and PRTG API access

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
        "mcp-remote",
        "https://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "your-mcp-server-api-key-from-config-yaml"
      }
    }
  }
}
```

**Replace:**
- `<MCP_SERVER_HOST>`: IP or hostname where MCP Server PRTG is running
- `your-mcp-server-api-key-from-config-yaml`: API key from MCP Server PRTG `config.yaml` (NOT PRTG API v2 token!)

### HTTP without TLS (Development/Testing Only)

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
        "MCP_SERVER_API_KEY": "your-mcp-server-api-key-from-config-yaml"
      }
    }
  }
}
```

⚠️ **Warning:** HTTP without TLS should only be used for local development/testing.

## Configuration Examples

### Example 1: Local Server (Same Machine)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "abc123def456ghi789",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0"
      }
    }
  }
}
```

**Note:** `NODE_TLS_REJECT_UNAUTHORIZED=0` is needed for self-signed certificates (development only).

### Example 2: Remote Server with Custom Port

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-mcp.company.com:9443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "prod-api-key-xyz"
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
        "mcp-remote",
        "https://prtg-server.local:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "prtg-server-key"
      }
    },
    "other-mcp": {
      "command": "npx",
      "args": [
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

The API key is located in your **MCP Server PRTG** `config.yaml` file:

```yaml
server:
  api_key: "your-mcp-server-api-key-here"
```

**Important:** This is the MCP Server authentication key, NOT the PRTG API v2 token!

### Key Distinction

```yaml
server:
  api_key: "abc123..."        # ← THIS is for Claude Desktop (MCP_SERVER_API_KEY)

prtg:
  api_token: "xyz789..."      # ← This is for PRTG Core Server (internal use only)
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
  "version": "1.2.2",
  "transport": "streamable-http",
  "protocol": "2025-03-26",
  "uptime": "1m30s",
  "database": "connected",
  "prtg_api": "enabled"
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
- Verify you're using the `server.api_key` from `config.yaml`
- NOT the `prtg.api_token`
- Check for extra spaces in the key

### "TLS Certificate Error"

**Causes:**
- Self-signed certificate not trusted
- Certificate expired
- Hostname mismatch

**Solutions:**

**For Development (Self-Signed Cert):**
```json
"env": {
  "MCP_SERVER_API_KEY": "your-key",
  "NODE_TLS_REJECT_UNAUTHORIZED": "0"
}
```

**For Production:**
- Use a proper TLS certificate from a trusted CA
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
   - Generate proper TLS certificates
   - Or use reverse proxy (nginx, Caddy)

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

# Linux/macOS (as daemon)
sudo systemctl start mcp-server-prtg

# Or run in foreground for testing
./mcp-server-prtg run
```

### 2. Get Your MCP Server API Key

```bash
# Windows (PowerShell)
type config.yaml | Select-String "api_key"

# Linux/macOS
grep "api_key" config.yaml
```

Output example:
```yaml
  api_key: "abc123def456ghi789"
```

### 3. Create Claude Desktop Config

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

### 4. Paste Configuration

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "PASTE_YOUR_KEY_HERE",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0"
      }
    }
  }
}
```

Replace `PASTE_YOUR_KEY_HERE` with your MCP Server API key from step 2.

### 5. Test Configuration

1. Save the file
2. Quit Claude Desktop completely
3. Restart Claude Desktop
4. Open new conversation
5. Ask: "What PRTG tools are available?"

## Common Mistakes

### ❌ Wrong: Using PRTG API v2 Token

```json
"env": {
  "MCP_SERVER_API_KEY": "5SIPLYZQND7TS4C4G32AVLNPS2XR6XWD64AOT7UYWE======"
}
```

This is the PRTG API v2 token (from PRTG Core Server). It won't work!

### ✅ Correct: Using MCP Server API Key

```json
"env": {
  "MCP_SERVER_API_KEY": "abc123def456ghi789"
}
```

This is from `config.yaml` → `server.api_key`.

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

**Last Updated**: 2025-11-03
**Protocol**: Streamable HTTP (MCP 2025-03-26)
**Endpoint**: `/mcp`
**Client**: `mcp-remote` (npm package)
