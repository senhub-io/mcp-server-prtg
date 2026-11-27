# Troubleshooting Guide

Common issues and their solutions for MCP Server PRTG.

## Table of Contents

- [Service Issues](#service-issues)
- [Administration Page Issues](#administration-page-issues)
- [Database Connection Issues](#database-connection-issues)
- [MCP Client Integration Issues](#mcp-client-integration-issues)
- [Connection Issues](#connection-issues)
- [Authentication Issues](#authentication-issues)
- [TLS/Certificate Issues](#tls-certificate-issues)
- [PRTG API v2 Issues](#prtg-api-v2-issues)
- [Data Exporter Coverage Issues](#data-exporter-coverage-issues)
- [Error Handling and Validation](#error-handling-and-validation)
- [Performance Issues](#performance-issues)
- [Configuration Issues](#configuration-issues)
- [Logging and Diagnostics](#logging-and-diagnostics)

## Service Issues

### Service Won't Start

#### Symptom
Service fails to start or exits immediately after starting.

#### Diagnosis

**Check service status:**
```bash
mcp-server-prtg status
```

**Check logs:**
```bash
# Linux
cat /var/log/mcp-server-prtg/mcp-server-prtg.log
# or, with the default relative path, from the configuration directory:
cat logs/mcp-server-prtg.log

# Windows (PowerShell)
Get-Content "C:\ProgramData\Sensor Factory\MCP Server PRTG\logs\mcp-server-prtg.log" -Tail 50
```

#### Common Causes and Solutions

**1. Port Already in Use**

Error: `bind: address already in use` or `listen tcp :8443: bind: address already in use`

Solution:
```bash
# Linux - Find process using port 8443
sudo lsof -i :8443
sudo netstat -tlnp | grep 8443

# Windows (PowerShell)
netstat -ano | findstr :8443

# Kill the conflicting process or change the port in the admin page Settings tab or config.yaml
```

**2. Database Connection Failed**

Error: `failed to connect to database` or `pq: password authentication failed`

Solution: See [Database Connection Issues](#database-connection-issues). Check that the database password is correct; if it was changed in the admin page or `config init`, restart the service.

**3. Missing Configuration File**

Error: `failed to load configuration: failed to read config file`

Solution:
```bash
# Check if config.yaml exists at the expected location
# Windows: C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml
# Linux: /etc/mcp-server-prtg/config.yaml

# If missing, create it:
mcp-server-prtg config init --config <path>
```

**4. Permission Denied**

Error: `permission denied` or `failed to create log directory`

Solution:
```bash
# Linux - Grant permissions
sudo chown -R root:root /etc/mcp-server-prtg
chmod 755 /etc/mcp-server-prtg
chmod 600 /etc/mcp-server-prtg/config.yaml

# Windows - Run as Administrator
```

**5. Certificate Files Not Found**

Error: `failed to load certificate` or `no such file or directory`

The self-signed certificate is regenerated automatically at start when missing. Delete the `certs` folder next to `config.yaml` and restart the service to regenerate it.

**6. API Key Missing or Too Short**

The server refuses to start if `server.api_key` is empty or shorter than 16 characters.

Solution: set `server.api_key` in `config.yaml` to a value of at least 16 characters (for example a UUID, which is what the installer generates), or run `config init --api-key <key>`.

### Service Crashes or Stops Unexpectedly

#### Diagnosis

**Check system logs:**
```bash
# Linux - systemd
sudo journalctl -u mcp-server-prtg -n 100

# Windows - Event Viewer
eventvwr.msc
# Navigate to: Windows Logs > Application
```

**Enable debug logging:**

In the admin page Settings tab, or in `config.yaml`:
```yaml
logging:
  level: debug
```

Then restart the service.

#### Common Causes

1. **Database Connection Lost**: Check database availability
2. **Out of Memory**: Monitor system resources
3. **Disk Full**: Check available disk space for logs
4. **Configuration Error**: Validate YAML syntax

### Service Won't Stop

#### Symptom
Service doesn't respond to stop command or takes too long.

#### Solution

**Force stop:**
```bash
# Linux
sudo systemctl stop mcp-server-prtg
# or
sudo killall mcp-server-prtg

# Windows (PowerShell) - Find and kill process
Get-Process mcp-server-prtg | Stop-Process -Force
```

### Windows SmartScreen Warning

SmartScreen may warn on beta pre-releases (`X.Y.Z-beta.N`), which are not signed. The production release 2.0.0 is code-signed (Authenticode, certificate "SENSOR FACTORY SAS" issued by Certum, RFC 3161 timestamp). Verify with:

```powershell
Get-AuthenticodeSignature mcp-server-prtg.exe
# Status should be: Valid
```

## Administration Page Issues

### Admin Page Unreachable from Another Machine

The administration page `https://localhost:<port>/admin/` is reachable from the server itself only (loopback). This is by design. Requests carrying proxy forwarding headers are refused. Open the page on the server, or use `config init` / `config client` from the command line.

### Admin Page Unreachable Behind a Proxy

If you place the server behind a reverse proxy, the admin page still only accepts direct loopback connections; proxy forwarding headers are refused. Access the admin page locally on the server.

### Login Refused / Too Many Attempts

Login failures are rate limited. After too many failed attempts, wait a few minutes before trying again with the correct MCP API key (get it with `config show-api-key`).

### "Restarting" Then Sign In Again

Sessions are kept in memory (8 hours, 30 minutes idle). After a restart the service asks to sign in again.

### "Enter the Password Again" When Changing Settings

In the Settings tab, a stored secret is only reused for the same database host/port/user without weakening the SSL mode, or for the same PRTG URL without turning off certificate verification. If you change the host, user, SSL mode or PRTG URL, type the password/token again.

## Database Connection Issues

### Connection Failed

#### Symptom
Error: `failed to connect to database` or `connection refused`

#### Diagnosis

**Test database connectivity:**
```bash
psql -h localhost -p 5432 -U mcp_prtg_reader -d prtg_data_exporter_collection
```

#### Solutions

**1. PostgreSQL Not Running**

Check PostgreSQL status:
```bash
# Linux
sudo systemctl status postgresql

# Windows
Get-Service postgresql*
```

Start PostgreSQL:
```bash
# Linux
sudo systemctl start postgresql

# Windows
Start-Service postgresql-x64-14
```

**2. Wrong Host or Port**

Verify configuration in the admin page Settings tab or `config.yaml`:
```yaml
database:
  host: "localhost"
  port: 5432
```

**3. Database Doesn't Exist**

The target database is `prtg_data_exporter_collection` (the PRTG Data Exporter 2026.x target database), not `prtg_data_exporter`.

**4. Firewall Blocking Connection**

For remote databases:
```bash
# Linux - Allow PostgreSQL port
sudo ufw allow 5432/tcp

# Check PostgreSQL is listening on correct interface
sudo netstat -tlnp | grep 5432
```

Edit PostgreSQL configuration:
```bash
# postgresql.conf
listen_addresses = '*'

# pg_hba.conf - Add client IP
host    prtg_data_exporter_collection    mcp_prtg_reader    192.168.1.0/24    md5
```

### Authentication Failed

#### Symptom
Error: `pq: password authentication failed for user "mcp_prtg_reader"`

#### Solutions

**1. Wrong Password**

Update the password in the admin page Settings tab or `config.yaml`, or use the environment variable:
```bash
export PRTG_DB_PASSWORD="correct_password"
```

**2. User Doesn't Exist or Lacks Rights**

Create the recommended read-only role on the target database:
```sql
CREATE ROLE mcp_prtg_reader LOGIN;
\password mcp_prtg_reader
GRANT CONNECT ON DATABASE prtg_data_exporter_collection TO mcp_prtg_reader;
-- in the prtg_data_exporter_collection database:
GRANT USAGE ON SCHEMA public TO mcp_prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO mcp_prtg_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE prtg_data_exporter_user_td IN SCHEMA public GRANT SELECT ON TABLES TO mcp_prtg_reader;
```

**3. pg_hba.conf Configuration**

Check PostgreSQL client authentication:
```bash
psql -U postgres -c "SHOW hba_file;"
# Edit pg_hba.conf - Add line
host    prtg_data_exporter_collection    mcp_prtg_reader    127.0.0.1/32    md5
# Reload PostgreSQL
sudo systemctl reload postgresql
```

### SSL/TLS Connection Issues

#### Symptom
Error: `pq: SSL is not enabled on the server`

#### Solutions

**Option 1: Disable SSL (for local connections)**
```yaml
database:
  sslmode: "disable"
```

**Option 2: Enable SSL in PostgreSQL**
```bash
# postgresql.conf
ssl = on
ssl_cert_file = '/path/to/server.crt'
ssl_key_file = '/path/to/server.key'

# Restart PostgreSQL
sudo systemctl restart postgresql
```

Update config:
```yaml
database:
  sslmode: "require"  # or "verify-ca" or "verify-full"
```

## MCP Client Integration Issues

### MCP Client Doesn't See Tools

#### Symptom
MCP tools don't appear in the MCP client's tools panel.

#### Diagnosis

**1. Check the client configuration printed by:**
```bash
mcp-server-prtg config client
```

**2. Check server is running:**
```bash
curl -k https://localhost:8443/health
```

#### Solutions

**1. Use the printed configuration**

`config client` prints ready-to-paste configurations for Claude Desktop (via `npx -y mcp-remote <url> --header "Authorization:Bearer ${MCP_PRTG_API_KEY}"` with `MCP_PRTG_API_KEY` and `NODE_EXTRA_CA_CERTS`), Claude Code, Cursor and VS Code.

**2. Restart the MCP client completely**

**3. Check the API key**

Get the key with:
```bash
mcp-server-prtg config show-api-key
```

### Tools Appear But Don't Work

#### Symptom
Tools are visible but return errors when used.

#### Solutions

**1. Authentication Error**

Error: `Unauthorized` or `401`

Verify the API key matches between the server and the client configuration.

**2. Certificate Error (Self-Signed)**

Error: `SSL certificate verify failed` or `certificate verification failed`

Do NOT set `NODE_TLS_REJECT_UNAUTHORIZED=0`. Instead, trust the server certificate: copy `certs/server.crt` to the client computer and import it into the trusted root certificates (see [TLS/Certificate Issues](#tls-certificate-issues)).

**3. Database Connection Error**

Check server logs and verify database connectivity.

## Connection Issues

### Connection Refused

#### Symptom
Error: `connection refused` or `unable to connect`

#### Solutions

**1. Verify Server is Running**
```bash
mcp-server-prtg status

# Check if port is listening
netstat -an | grep 8443
```

**2. Check Firewall**
```bash
# Linux - Allow port 8443
sudo ufw allow 8443/tcp

# Windows - Add firewall rule
netsh advfirewall firewall add rule name="MCP Server PRTG" dir=in action=allow protocol=TCP localport=8443
```

**3. Check Bind Address**

In `config.yaml`:
```yaml
server:
  bind_address: "0.0.0.0"  # Not "127.0.0.1" for remote access
```

### Connection Drops Frequently

#### Symptom
Streamable HTTP connection disconnects and reconnects repeatedly.

#### Solutions

**1. Network Issues**
- Check network stability
- Use wired connection instead of WiFi

**2. Proxy or Firewall Interference**
- Configure proxy to allow long-lived connections
- Ensure proxies don't interfere with heartbeat mechanism (30s interval)

**3. Resource Exhaustion**
- Check server CPU/memory usage

## Authentication Issues

### Unauthorized (401) Errors

#### Symptom
Error: HTTP 401 with the body `Unauthorized`

#### Solutions

**1. Verify API Key**

Get from the server:
```bash
mcp-server-prtg config show-api-key
```

Test manually:
```bash
curl -k -H "Authorization: Bearer your-api-key" \
     https://localhost:8443/status
```

**2. Check Header Format**

Correct format:
```
Authorization: Bearer your-api-key
```

**NOT:**
- `Authorization: your-api-key` (missing "Bearer")
- `Authorization: Bearer: your-api-key` (extra colon)
- `https://server:8443/mcp?token=your-api-key` (the `token` query parameter is not accepted; use the header)

**3. API Key Changed**

If the API key was changed, update the client configuration and restart the MCP client.

### Too Many Requests (429) Errors

#### Symptom
Requests are rejected with HTTP 429, even with the correct API key.

#### Cause
Failed authentication attempts are rate limited per client address: after 5 failures within one minute, the address is blocked for 5 minutes. The address is the TCP peer address (IPv6 addresses are grouped by /64); `X-Forwarded-For` and `X-Real-IP` headers are ignored.

#### Solutions
- Fix the API key in the client configuration, then wait 5 minutes before retrying
- Behind a reverse proxy, all clients share the proxy address: one misconfigured client can block the others. Fix that client, or let clients connect directly to the server

## TLS/Certificate Issues

### Certificate Verification Failed

#### Symptom
Error when using HTTPS with self-signed certificates:
```
Error: unable to verify the first certificate
```

Or:
```
Error: self signed certificate
```

#### Solution: Trust the Server Certificate

The self-signed certificate (`certs/server.crt` next to `config.yaml`) covers `localhost`, the machine name and its IP addresses. Clients must trust it instead of disabling verification.

Copy `server.crt` to each client computer:

- **Claude Desktop / mcp-remote:** set `NODE_EXTRA_CA_CERTS` to the path of a copy of the certificate.
- **Claude Code, Cursor, VS Code:** import into the computer's trusted root certificates:
  - **Windows:** `certlm.msc` → Trusted Root Certification Authorities, or `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`
  - **macOS:** Keychain Access, "Always Trust"
  - **Linux:** copy to `/usr/local/share/ca-certificates/` and run `sudo update-ca-certificates`

Do NOT set `NODE_TLS_REJECT_UNAUTHORIZED=0`: it exposes the API key.

### Certificate Covers Wrong Names After Renaming the Server

The certificate covers `localhost`, the machine name and its IP addresses. After renaming the server, delete the `certs` folder next to `config.yaml` and restart the service to regenerate the certificate.

### Using a Company PKI Certificate

Set `server.cert_file` and `server.key_file` to a certificate from your PKI and restart the service.

### Disabling TLS (Development Only)

For development only: set `enable_tls: false` and use `http://` in the client configuration. All communication including API keys will be transmitted in plain text.

## PRTG API v2 Issues

### PRTG API v2 Not Active

If PRTG API calls fail, check that the PRTG API v2 is activated on the PRTG core server (PRTG Application Server, ports 1616/1615 and 23580; default on new installs since 25.2.106, automatic on update to 26.4.126 when enough memory is free; otherwise Setup → Activate, then restart the core server).

Use the **Test connection** button in the admin page Settings tab to verify the PRTG URL (including scheme and port, typically 1616) and token.

## Data Exporter Coverage Issues

### Stale Sensors

If a sensor's Data Exporter export is older than 24 hours, the metrics tools fall back to the PRTG API when configured; otherwise the answer states when the export stopped. Check the Data Exporter collection profiles in `prtg_get_statistics` (last/next collection, errors, paused flag).

### Profile Flagged Paused

Data Exporter sets a paused flag while throttling. `prtg_get_alerts` excludes paused sensors unless a status filter is given.

### Export Lag

Check the channel-history coverage in `prtg_get_statistics` (up to date, stale, never fetched, without data, not tracked) and the collection profiles for errors.

## Error Handling and Validation

### Understanding Pedagogical Errors

MCP Server PRTG uses pedagogical error messages designed to guide users toward resolution. These errors are not bugs - they're helpful guidance.

#### Symptom
A message like:
```
Invalid sensor_id

The sensor_id parameter must be a positive integer corresponding to an existing PRTG sensor.

How to resolve:
1. Search by name: prtg_search search_term="sensor_name"
2. List alerts (with IDs): prtg_get_alerts
3. Explore a device: prtg_device_overview device_name="device_name"
```

#### What This Means
This is working as designed. The error message:
- Explains what went wrong in plain English
- Provides numbered action items to resolve the issue
- Includes concrete examples of correct usage

#### Common Pedagogical Errors

**ErrInvalidSensorID**
- **Trigger**: sensor_id parameter is ≤0 or invalid
- **Self-Resolution**: use prtg_search or prtg_get_alerts to discover valid IDs

**ErrDeviceNotFound**
- **Trigger**: Device name doesn't match any devices
- **Self-Resolution**: use prtg_search to find similarly named devices

**ErrEmptySearchTerm**
- **Trigger**: search_term parameter is missing or empty
- **Self-Resolution**: provide a search term

**ErrCustomQueriesDisabled**
- **Trigger**: Attempting prtg_query_sql when allow_custom_queries=false
- **Self-Resolution**: use predefined tools (prtg_get_sensors, prtg_get_alerts, etc.)

### Validation Errors vs. System Errors

**Validation Errors (Pedagogical):**
- Returned as tool output (not JSON-RPC errors)
- Include suggestions
- Can be acted on directly
- Examples: invalid parameters, missing fields, resource not found

**System Errors (Technical):**
- Returned as JSON-RPC error responses
- Indicate server/infrastructure issues
- Require human intervention
- Examples: database connection failed, timeout, internal server error

#### When to Report Issues

**DO NOT report as bugs:**
- Invalid sensor_id - This is validation guidance
- Device not found - This is search assistance
- Empty search term - This is parameter validation
- Custom queries disabled - This is security configuration

**DO report as bugs:**
- Database connection errors
- Server crashes or panics
- Incorrect data returned
- Timeouts or performance issues
- Authentication failures (when credentials are correct)

## Performance Issues

### Slow Query Responses

#### Symptom
Queries take a long time to return results.

#### Solutions

**1. Reduce Result Set Size**

Use smaller limits:
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "limit": 50
  }
}
```

**2. Add More Specific Filters**

Filter by device or sensor name.

**3. Database Performance**

Check PostgreSQL performance:
```sql
psql -U mcp_prtg_reader -d prtg_data_exporter_collection

-- Check slow queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC;
```

Database queries have a 30-second timeout (15 seconds for custom queries run through `prtg_query_sql`).

### High Memory Usage

#### Solutions

**1. Limit Result Sets**

Always use `limit` parameters to prevent large result sets.

**2. Increase Server Memory**

Add more RAM or adjust system limits.

### High CPU Usage

#### Solutions

**1. Check for Query Loops**

Review queries that might be executing repeatedly.

**2. Database Connection Issues**

Failed database connections with retries can cause high CPU usage.

**3. Restart Service**

```bash
# Linux
sudo systemctl restart mcp-server-prtg

# Windows
Restart-Service mcp-server-prtg
```

## Configuration Issues

### Invalid YAML Syntax

#### Symptom
Error: `yaml: unmarshal errors` or `failed to parse config file`

#### Solutions

**Validate YAML syntax:**
```bash
python -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

**Common YAML errors:**
- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted special characters
- Wrong data types

### Changes Require Restart

Some settings require a restart: bind address, port, TLS settings and certificate files. Use **Save and restart** in the admin page, or restart the service.

## Logging and Diagnostics

### Enable Debug Logging

In the admin page Settings tab, or in `config.yaml`:
```yaml
logging:
  level: debug
```

Then restart the service.

### Check Logs

**View recent logs:**
```bash
# Linux
tail -f /var/log/mcp-server-prtg/mcp-server-prtg.log

# Windows (PowerShell)
Get-Content "C:\ProgramData\Sensor Factory\MCP Server PRTG\logs\mcp-server-prtg.log" -Wait -Tail 50
```

**Search logs:**
```bash
grep "error" logs/mcp-server-prtg.log
grep "Unauthorized" logs/mcp-server-prtg.log
```

### Log Rotation Issues

#### Symptom
Log files grow too large or disk fills up.

#### Solutions

Adjust rotation settings in `config.yaml`:
```yaml
logging:
  max_size_mb: 10
  max_backups: 5
  max_age_days: 30
  compress: true
```

### Collect Diagnostic Information

When reporting issues, collect:

```bash
# 1. Version
mcp-server-prtg --version

# 2. Status
mcp-server-prtg status

# 3. Configuration (REDACT PASSWORD and API KEY)
cat config.yaml | grep -v password | grep -v api_key

# 4. Recent logs
tail -n 100 logs/mcp-server-prtg.log

# 5. System info
uname -a

# 6. Network status
netstat -an | grep 8443

# 7. Database connectivity
psql -h localhost -U mcp_prtg_reader -d prtg_data_exporter_collection -c "SELECT version();"
```

## Getting Help

If you can't resolve your issue:

1. **Check documentation:**
   - [Configuration Guide](CONFIGURATION.md)
   - [Usage Guide](USAGE.md)
   - [Architecture](ARCHITECTURE.md)
   - [Tools Reference](TOOLS.md)

2. **Search existing issues:**
   - [GitHub Issues](https://github.com/senhub-io/mcp-server-prtg/issues)

3. **Open a new issue:**
   - Include diagnostic information (see above)
   - Describe steps to reproduce
   - Include relevant log excerpts
   - Specify your environment (OS, version, etc.)

## See Also

- [Configuration Guide](CONFIGURATION.md)
- [Usage Guide](USAGE.md)
- [Tools Reference](TOOLS.md)
- [Architecture](ARCHITECTURE.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
