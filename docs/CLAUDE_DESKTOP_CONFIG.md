# Claude Desktop Configuration

## Configuration File Location

### Windows
```
%APPDATA%\Claude\claude_desktop_config.json
```
Full path example: `C:\Users\YourUsername\AppData\Roaming\Claude\claude_desktop_config.json`

### macOS
```
~/Library/Application Support/Claude/claude_desktop_config.json
```

### Linux
```
~/.config/Claude/claude_desktop_config.json
```

## Configuration Modes

There are two ways to configure the MCP server in Claude Desktop:

1. **Command Mode (Recommended)**: Claude Desktop launches and manages the server automatically
2. **URL Mode**: You run the server manually and Claude Desktop connects to it

## Configuration Examples

### Example 1: Command Mode - Auto-Launch (Recommended)

**Windows:**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "C:\\path\\to\\mcp-server-prtg.exe",
      "args": ["run", "--config", "C:\\path\\to\\config.yaml"],
      "env": {
        "PRTG_DB_PASSWORD": "your-database-password"
      }
    }
  }
}
```

**macOS/Linux:**
```json
{
  "mcpServers": {
    "prtg": {
      "command": "/path/to/mcp-server-prtg",
      "args": ["run", "--config", "/path/to/config.yaml"],
      "env": {
        "PRTG_DB_PASSWORD": "your-database-password"
      }
    }
  }
}
```

**Benefits:**
- ✅ Claude Desktop starts/stops the server automatically
- ✅ No need to manage server lifecycle manually
- ✅ Server runs only when Claude Desktop is running
- ✅ Logs are captured by Claude Desktop

**Note:** With command mode, the server runs in stdio mode and doesn't need HTTP/TLS configuration.

### Example 2: URL Mode - Manual Server (Remote Access)

This mode requires you to start the server manually first.

**HTTPS with TLS (Recommended for Production):**

```json
{
  "mcpServers": {
    "prtg": {
      "url": "https://your-server-address:8443/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key-from-config-yaml"
      }
    }
  }
}
```

**Notes:**
- Replace `your-server-address` with your actual server hostname or IP
- Replace `your-api-key-from-config-yaml` with the `api_key` value from your `config.yaml`
- Port `8443` is the default, change if you configured differently
- The `/mcp` endpoint is mandatory (Streamable HTTP protocol)

### Example 2: HTTP without TLS (Development/Testing Only)

```json
{
  "mcpServers": {
    "prtg": {
      "url": "http://localhost:8443/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key-from-config-yaml"
      }
    }
  }
}
```

**⚠️ Warning:** HTTP without TLS should only be used for local development/testing.

### Example 3: Remote Server with Custom Port

```json
{
  "mcpServers": {
    "prtg": {
      "url": "https://prtg-mcp.example.com:9443/mcp",
      "headers": {
        "Authorization": "Bearer abc123def456ghi789"
      }
    }
  }
}
```

### Example 4: Multiple MCP Servers

```json
{
  "mcpServers": {
    "prtg": {
      "url": "https://prtg-server.local:8443/mcp",
      "headers": {
        "Authorization": "Bearer your-prtg-api-key"
      }
    },
    "other-mcp-server": {
      "url": "https://other-server.local:8080/mcp",
      "headers": {
        "Authorization": "Bearer other-api-key"
      }
    }
  }
}
```

## Finding Your API Key

The API key is located in your server's `config.yaml` file:

```yaml
server:
  api_key: "your-generated-api-key-here"
```

If you don't have a config file yet, run:

```bash
# Windows
mcp-server-prtg.exe run --config config.yaml

# Linux/macOS
./mcp-server-prtg run --config config.yaml
```

This will generate a new `config.yaml` with a random API key.

## Verifying Configuration

### Step 1: Check Server is Running

Open a terminal and test the health endpoint:

```bash
# Windows (PowerShell)
curl http://localhost:8443/health

# Linux/macOS
curl http://localhost:8443/health
```

Expected response: `{"status":"ok"}`

### Step 2: Test Authentication

Test the status endpoint with your API key:

```bash
# Windows (PowerShell)
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8443/status

# Linux/macOS
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8443/status
```

Expected response:
```json
{
  "version": "1.1.0",
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
4. Test with a query like: "Can you list the available MCP tools?"

## Troubleshooting

### "Connection Refused" Error

**Possible causes:**
1. Server is not running
2. Wrong port in configuration
3. Firewall blocking the connection

**Solution:**
- Verify server is running: `curl http://localhost:8443/health`
- Check server logs for errors
- Verify port matches `server.address` in `config.yaml`

### "Unauthorized" Error

**Possible causes:**
1. Wrong API key
2. Missing `Authorization` header
3. API key contains spaces or special characters

**Solution:**
- Verify API key matches exactly (no extra spaces)
- Ensure you're using `Bearer YOUR_API_KEY` format
- Check server logs for authentication attempts

### "TLS Certificate Error"

**Possible causes:**
1. Self-signed certificate not trusted
2. Certificate expired
3. Hostname mismatch

**Solution:**
- For testing, use HTTP without TLS (local only)
- For production, use a proper TLS certificate
- Regenerate certificates if expired

### Server Not Appearing in Claude Desktop

**Solution:**
1. Verify JSON syntax is correct (use a JSON validator)
2. Ensure file is saved as `claude_desktop_config.json`
3. Restart Claude Desktop completely
4. Check Claude Desktop logs for errors

## Advanced Configuration

### Custom Timeouts (if supported by Claude Desktop)

```json
{
  "mcpServers": {
    "prtg": {
      "url": "https://prtg-server.local:8443/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key"
      },
      "timeout": 30000
    }
  }
}
```

### Environment-Specific Configuration

You can maintain separate configurations for different environments:

**Development:**
```json
{
  "mcpServers": {
    "prtg-dev": {
      "url": "http://localhost:8443/mcp",
      "headers": {
        "Authorization": "Bearer dev-api-key"
      }
    }
  }
}
```

**Production:**
```json
{
  "mcpServers": {
    "prtg-prod": {
      "url": "https://prtg.company.com:8443/mcp",
      "headers": {
        "Authorization": "Bearer prod-api-key"
      }
    }
  }
}
```

## Complete Example with Comments

```json
{
  "mcpServers": {
    // Server name (appears in Claude Desktop)
    "prtg": {
      // Server URL - MUST end with /mcp
      // Format: https://hostname:port/mcp
      "url": "https://prtg-server.local:8443/mcp",

      // Headers sent with each request
      "headers": {
        // Bearer token authentication
        // Get this from your config.yaml file
        "Authorization": "Bearer your-api-key-here"
      }
    }
  }
}
```

**Note:** JSON does not support comments. Remove all `//` comments before using.

## Migration from SSE to Streamable HTTP

If you were using the old SSE transport, only the endpoint needs to change:

**Old Configuration (SSE - Deprecated):**
```json
"url": "https://localhost:8443/sse"
```

**New Configuration (Streamable HTTP):**
```json
"url": "https://localhost:8443/mcp"
```

Everything else remains the same!

## Quick Setup Guide

### 1. Start the Server

**Windows:**
```powershell
cd C:\path\to\mcp-server-prtg
.\mcp-server-prtg.exe run --config config.yaml
```

**Linux/macOS:**
```bash
cd /path/to/mcp-server-prtg
./mcp-server-prtg run --config config.yaml
```

### 2. Get Your API Key

```bash
# Windows (PowerShell)
type config.yaml | Select-String "api_key"

# Linux/macOS
grep api_key config.yaml
```

### 3. Create Claude Desktop Config

**Windows:**
```powershell
# Create directory if it doesn't exist
New-Item -ItemType Directory -Force -Path "$env:APPDATA\Claude"

# Edit the config file
notepad "$env:APPDATA\Claude\claude_desktop_config.json"
```

**macOS:**
```bash
# Create directory if it doesn't exist
mkdir -p ~/Library/Application\ Support/Claude

# Edit the config file
nano ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Linux:**
```bash
# Create directory if it doesn't exist
mkdir -p ~/.config/Claude

# Edit the config file
nano ~/.config/Claude/claude_desktop_config.json
```

### 4. Paste This Configuration

```json
{
  "mcpServers": {
    "prtg": {
      "url": "http://localhost:8443/mcp",
      "headers": {
        "Authorization": "Bearer PASTE_YOUR_API_KEY_HERE"
      }
    }
  }
}
```

Replace `PASTE_YOUR_API_KEY_HERE` with your actual API key from step 2.

### 5. Restart Claude Desktop

- Quit Claude Desktop completely
- Start Claude Desktop
- Open a new conversation
- Test: "What MCP tools are available?"

## Security Best Practices

1. **Never commit API keys to version control**
2. **Use HTTPS in production** (generate proper TLS certificates)
3. **Rotate API keys regularly**
4. **Use firewall rules** to restrict access to the server
5. **Monitor server logs** for suspicious activity
6. **Keep the server updated** with latest releases

## Support

For issues or questions:

1. Check server logs: `logs/mcp-server-prtg.log`
2. Verify configuration: `curl http://localhost:8443/health`
3. Review: `docs/TROUBLESHOOTING.md`
4. Open GitHub issue with logs and configuration (redact API keys!)

---

**Last Updated**: 2025-10-28
**Protocol**: Streamable HTTP (MCP 2025-03-26)
**Endpoint**: `/mcp`
