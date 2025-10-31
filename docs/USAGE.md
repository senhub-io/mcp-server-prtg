# Usage Guide

This guide covers how to use MCP Server PRTG with MCP Client, direct API access, and integration examples.

## Table of Contents

- [MCP Client Setup](#claude-desktop-setup)
- [Example Queries with LLM](#example-queries-with-claude)
- [Direct API Usage](#direct-api-usage)
- [Integration Examples](#integration-examples)
- [Best Practices](#best-practices)

## MCP Client Setup

MCP Server PRTG uses **Streamable HTTP transport** (MCP 2025-03-26), which is natively supported by [mcp-remote](https://github.com/modelcontextprotocol/mcp-remote).

### Prerequisites

- MCP Server PRTG installed and running
- Node.js and npm
- MCP Client application (e.g., Claude Desktop)

### Step 1: Install mcp-remote

```bash
# mcp-remote is automatically installed via npx
# No separate installation required
npx mcp-remote --version
```

### Step 2: Get Your API Key

Your API key is located in `config.yaml`:

```bash
# Linux/macOS
cat config.yaml | grep api_key

# Windows (PowerShell)
Select-String -Path config.yaml -Pattern "api_key"
```

Example output:
```yaml
api_key: "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
```

### Step 3: Configure MCP Client

Edit MCP Client configuration file:

**Location:**
- **macOS:** `~/Library/Application Support/LLM/mcp_client_config.json`
- **Windows:** `%APPDATA%\LLM\mcp_client_config.json`
- **Linux:** `~/.config/LLM/mcp_client_config.json`

**Configuration:**

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
      }
    }
  }
}
```

**For remote servers:**

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://prtg-mcp.example.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
      }
    }
  }
}
```

**For HTTP (development only):**

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "http://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${PRTG_API_KEY}"
      ],
      "env": {
        "PRTG_API_KEY": "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
      }
    }
  }
}
```

**Note:** For production with self-signed certificates, add `NODE_TLS_REJECT_UNAUTHORIZED=0` to `env` (not recommended) or use trusted CA certificates.

### Step 4: Restart MCP Client

1. Quit MCP Client completely
2. Restart MCP Client
3. Open the MCP tools panel (hammer icon in the bottom-right corner)
4. Verify that 6 PRTG tools are available

### Verification

In MCP Client, you should see these tools:
- prtg_get_sensors
- prtg_get_sensor_status
- prtg_get_alerts
- prtg_device_overview
- prtg_top_sensors
- prtg_query_sql

## Example Queries with LLM

Once configured, you can ask LLM natural language questions about your PRTG monitoring infrastructure.

### Basic Sensor Queries

**List all sensors:**
```
Show me all PRTG sensors
```

**Search for specific sensors:**
```
Show me all ping sensors
```

**Filter by device:**
```
Show me all sensors on server "web-prod-01"
```

**Filter by status:**
```
Show me all sensors that are currently down
```

**Combine filters:**
```
Show me all HTTP sensors on devices with "prod" in the name that are in warning state
```

### Alert Monitoring

**Current alerts:**
```
What alerts do I have right now in PRTG?
```

**Recent alerts:**
```
Show me all alerts from the last 24 hours
```

**Critical alerts only:**
```
Show me all sensors that are currently down
```

**Alerts by device:**
```
What alerts do I have on server "db-prod-01"?
```

### Device Information

**Device overview:**
```
Give me a complete overview of device "firewall-main"
```

**Device health:**
```
What's the status of all sensors on "web-prod-02"?
```

**Multiple devices:**
```
Show me the status of all devices with "database" in their name
```

### Performance Analysis

**Top problematic sensors:**
```
Which sensors have the most downtime in the last 24 hours?
```

**Uptime analysis:**
```
Show me the top 10 sensors with the best uptime
```

**Alert frequency:**
```
Which sensors are generating the most alerts?
```

### Detailed Status

**Specific sensor:**
```
Show me detailed information about sensor ID 12345
```

**Sensor history:**
```
What's the uptime and downtime for sensor "Main Website HTTP"?
```

### Advanced SQL Queries

**Custom queries:**
```
Run a SQL query to find all sensors that haven't checked in for more than 1 hour
```

**Complex analysis:**
```
Find all devices that have more than 5 sensors in down state
```

**Trend analysis:**
```
Show me all sensors that were down for more than 4 hours today
```

### Report Generation

**Summary reports:**
```
Create a summary report of my PRTG infrastructure including total sensors, alerts, and top issues
```

**Device reports:**
```
Generate a health report for all production web servers
```

**Trend reports:**
```
Analyze sensor reliability over the past week and identify problem areas
```

## Direct API Usage

You can interact with MCP Server PRTG directly using HTTP requests (useful for automation, testing, or custom integrations).

### Authentication

All requests require Bearer token authentication:

```
Authorization: Bearer your-api-key
```

### Endpoints

**MCP Endpoint:** `POST/GET /mcp` (Streamable HTTP)
**Health Check:** `GET /health`
**Status:** `GET /status`

### Health Check

```bash
curl https://localhost:8443/health
```

Response:
```json
{
  "status": "healthy",
  "timestamp": "2025-10-26T10:30:00Z"
}
```

### Status Check (Authenticated)

```bash
curl -H "Authorization: Bearer your-api-key" \
     https://localhost:8443/status
```

Response:
```json
{
  "status": "running",
  "version": "1.0.2-beta",
  "transport": "streamable_http",
  "tls_enabled": true,
  "base_url": "https://localhost:8443",
  "mcp_tools": 6,
  "database": {
    "status": "connected",
    "error": ""
  },
  "timestamp": "2025-10-26T10:30:00Z"
}
```

### MCP Tool Calls

MCP Server PRTG implements the [Model Context Protocol](https://modelcontextprotocol.io). Tool calls use JSON-RPC 2.0 over the `/mcp` endpoint (Streamable HTTP transport).

**Example: List all sensors**

```bash
curl -X POST https://localhost:8443/mcp \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "prtg_get_sensors",
      "arguments": {
        "limit": 10
      }
    }
  }'
```

**Example: Get alerts**

```bash
curl -X POST https://localhost:8443/mcp \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type": application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "prtg_get_alerts",
      "arguments": {
        "hours": 24
      }
    }
  }'
```

**Example: Device overview**

```bash
curl -X POST https://localhost:8443/mcp \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "prtg_device_overview",
      "arguments": {
        "device_name": "web-prod-01"
      }
    }
  }'
```

**Example: Custom SQL query**

```bash
curl -X POST https://localhost:8443/mcp \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "tools/call",
    "params": {
      "name": "prtg_query_sql",
      "arguments": {
        "query": "SELECT name, status FROM prtg_sensor WHERE status != 3 LIMIT 10",
        "limit": 10
      }
    }
  }'
```

### Self-Signed Certificates

For development with self-signed certificates, use the `-k` or `--insecure` flag:

```bash
curl -k https://localhost:8443/health
```

## Integration Examples

### Python Integration

```python
import requests
import json

class PRTGMCPClient:
    def __init__(self, base_url, api_key, verify_ssl=True):
        self.base_url = base_url
        self.headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        }
        self.verify_ssl = verify_ssl
        self.request_id = 0

    def call_tool(self, tool_name, arguments):
        """Call an MCP tool"""
        self.request_id += 1
        payload = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": arguments
            }
        }

        response = requests.post(
            f"{self.base_url}/mcp",
            headers=self.headers,
            json=payload,
            verify=self.verify_ssl
        )
        response.raise_for_status()
        return response.json()

    def get_sensors(self, device_name=None, sensor_name=None, limit=50):
        """Get sensors with optional filters"""
        args = {"limit": limit}
        if device_name:
            args["device_name"] = device_name
        if sensor_name:
            args["sensor_name"] = sensor_name

        return self.call_tool("prtg_get_sensors", args)

    def get_alerts(self, hours=24):
        """Get current alerts"""
        return self.call_tool("prtg_get_alerts", {"hours": hours})

    def get_device_overview(self, device_name):
        """Get device overview"""
        return self.call_tool("prtg_device_overview", {"device_name": device_name})

# Usage example
client = PRTGMCPClient(
    base_url="https://localhost:8443",
    api_key="your-api-key",
    verify_ssl=False  # For self-signed certs
)

# Get all alerts
alerts = client.get_alerts(hours=24)
print(json.dumps(alerts, indent=2))

# Get sensors for a specific device
sensors = client.get_sensors(device_name="web-prod")
print(json.dumps(sensors, indent=2))
```

### Bash/Shell Script Integration

```bash
#!/bin/bash

# Configuration
API_KEY="your-api-key"
BASE_URL="https://localhost:8443"

# Function to call MCP tools
call_tool() {
    local tool_name="$1"
    local arguments="$2"

    curl -s -k -X POST "${BASE_URL}/mcp" \
        -H "Authorization: Bearer ${API_KEY}" \
        -H "Content-Type: application/json" \
        -d "{
            \"jsonrpc\": \"2.0\",
            \"id\": 1,
            \"method\": \"tools/call\",
            \"params\": {
                \"name\": \"${tool_name}\",
                \"arguments\": ${arguments}
            }
        }"
}

# Get alerts and send to Slack
alerts=$(call_tool "prtg_get_alerts" '{"hours": 1}')
alert_count=$(echo "$alerts" | jq -r '.result.content[0].text' | grep -o 'Found [0-9]* result' | awk '{print $2}')

if [ "$alert_count" -gt 0 ]; then
    curl -X POST "https://hooks.slack.com/services/YOUR/WEBHOOK/URL" \
        -H "Content-Type: application/json" \
        -d "{\"text\": \"⚠️ PRTG Alert: ${alert_count} sensors in alert state\"}"
fi
```

### Node.js Integration

```javascript
const axios = require('axios');
const https = require('https');

class PRTGMCPClient {
  constructor(baseUrl, apiKey, verifySSL = true) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
    this.requestId = 0;

    this.client = axios.create({
      baseURL: baseUrl,
      headers: {
        'Authorization': `Bearer ${apiKey}`,
        'Content-Type': 'application/json'
      },
      httpsAgent: new https.Agent({
        rejectUnauthorized: verifySSL
      })
    });
  }

  async callTool(toolName, arguments) {
    this.requestId++;
    const payload = {
      jsonrpc: '2.0',
      id: this.requestId,
      method: 'tools/call',
      params: {
        name: toolName,
        arguments: arguments
      }
    };

    const response = await this.client.post('/mcp', payload);
    return response.data;
  }

  async getSensors(filters = {}) {
    return this.callTool('prtg_get_sensors', {
      limit: filters.limit || 50,
      ...filters
    });
  }

  async getAlerts(hours = 24) {
    return this.callTool('prtg_get_alerts', { hours });
  }

  async getDeviceOverview(deviceName) {
    return this.callTool('prtg_device_overview', { device_name: deviceName });
  }
}

// Usage
(async () => {
  const client = new PRTGMCPClient(
    'https://localhost:8443',
    'your-api-key',
    false // Don't verify SSL for self-signed certs
  );

  try {
    const alerts = await client.getAlerts(24);
    console.log(JSON.stringify(alerts, null, 2));
  } catch (error) {
    console.error('Error:', error.message);
  }
})();
```

### PowerShell Integration

```powershell
# Configuration
$ApiKey = "your-api-key"
$BaseUrl = "https://localhost:8443"

# Ignore SSL certificate validation (for self-signed certs)
add-type @"
    using System.Net;
    using System.Security.Cryptography.X509Certificates;
    public class TrustAllCertsPolicy : ICertificatePolicy {
        public bool CheckValidationResult(
            ServicePoint srvPoint, X509Certificate certificate,
            WebRequest request, int certificateProblem) {
            return true;
        }
    }
"@
[System.Net.ServicePointManager]::CertificatePolicy = New-Object TrustAllCertsPolicy

# Function to call MCP tools
function Invoke-PRTGTool {
    param(
        [string]$ToolName,
        [hashtable]$Arguments
    )

    $headers = @{
        "Authorization" = "Bearer $ApiKey"
        "Content-Type" = "application/json"
    }

    $body = @{
        jsonrpc = "2.0"
        id = 1
        method = "tools/call"
        params = @{
            name = $ToolName
            arguments = $Arguments
        }
    } | ConvertTo-Json -Depth 10

    $response = Invoke-RestMethod -Uri "$BaseUrl/mcp" `
        -Method Post `
        -Headers $headers `
        -Body $body

    return $response
}

# Get alerts
$alerts = Invoke-PRTGTool -ToolName "prtg_get_alerts" -Arguments @{ hours = 24 }
$alerts | ConvertTo-Json -Depth 10

# Get device overview
$device = Invoke-PRTGTool -ToolName "prtg_device_overview" -Arguments @{ device_name = "web-prod-01" }
$device | ConvertTo-Json -Depth 10
```

## Best Practices

### Performance

1. **Use specific filters** to reduce data transfer and query time:
   ```
   # Good
   Show me ping sensors on server "web-01"

   # Less efficient
   Show me all sensors (then filter manually)
   ```

2. **Set appropriate limits** for large result sets:
   ```
   # Good
   Show me the top 10 sensors with most downtime

   # Less efficient
   Show me all sensors with downtime
   ```

3. **Use device_overview** instead of multiple queries:
   ```
   # Good
   Give me an overview of device "web-01"

   # Less efficient
   Show me all sensors on "web-01", then count by status
   ```

### Security

1. **Protect your API key:**
   - Never commit API keys to version control
   - Use environment variables or secrets management
   - Rotate keys periodically

2. **Use HTTPS in production:**
   - Always enable TLS for remote connections
   - Use trusted certificates in production
   - Only use `--insecure` in development

3. **Implement rate limiting** in your integrations to avoid overwhelming the server

4. **Use read-only database user** with minimal permissions

### Reliability

1. **Implement error handling** in your integrations:
   ```python
   try:
       result = client.get_alerts()
   except requests.exceptions.RequestException as e:
       logging.error(f"API request failed: {e}")
       # Implement retry logic or alerting
   ```

2. **Set reasonable timeouts** to avoid hanging requests:
   ```python
   response = requests.post(url, timeout=30)  # 30 second timeout
   ```

3. **Monitor server health** before making tool calls:
   ```bash
   # Check health before running queries
   curl https://localhost:8443/health
   ```

### Optimization

1. **Cache results** when appropriate:
   ```python
   # Cache sensor list for 5 minutes
   @lru_cache(maxsize=128, ttl=300)
   def get_sensors_cached():
       return client.get_sensors()
   ```

2. **Use SQL queries** for complex filtering:
   ```
   Instead of fetching all sensors and filtering in code,
   use prtg_query_sql with a WHERE clause
   ```

3. **Batch operations** when processing multiple devices

## Troubleshooting

### MCP Client Can't See Tools

**Issue:** MCP tools don't appear in MCP Client

**Solutions:**
1. Verify mcp-remote is accessible: `npx mcp-remote --version`
2. Check MCP Client config file syntax (valid JSON)
3. Ensure the URL is correct (include `/mcp` endpoint)
4. Verify API key is correct
5. Restart MCP Client completely (quit and reopen)

### Connection Failed

**Issue:** Cannot connect to MCP server

**Solutions:**
1. Verify server is running: `curl https://localhost:8443/health`
2. Check firewall allows traffic on port 8443
3. For remote servers, ensure public_url is correctly configured
4. For self-signed certs, add `NODE_TLS_REJECT_UNAUTHORIZED=0` to env (not recommended for production)

### Authentication Failed

**Issue:** "Unauthorized: Missing or invalid Bearer token"

**Solutions:**
1. Verify API key in config.yaml matches your client configuration
2. Ensure "Bearer " prefix is included in Authorization header
3. Check for extra spaces or line breaks in API key
4. Hot-reload may have changed the API key - restart clients

### Slow Queries

**Issue:** Queries take a long time to complete

**Solutions:**
1. Reduce limit parameter in queries
2. Add more specific filters (device_name, sensor_name)
3. Check database performance and indexes
4. Increase query timeout if needed
5. Use prtg_query_sql with LIMIT clauses

## See Also

- [Configuration Guide](CONFIGURATION.md)
- [Tools Reference](TOOLS.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
- [MCP Protocol Documentation](https://modelcontextprotocol.io)
