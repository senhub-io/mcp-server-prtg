# Troubleshooting Guide

Common issues and their solutions for MCP Server PRTG.

## Table of Contents

- [Service Issues](#service-issues)
- [Database Connection Issues](#database-connection-issues)
- [MCP Client Integration Issues](#claude-desktop-integration-issues)
- [Connection Issues](#connection-issues)
- [Authentication Issues](#authentication-issues)
- [TLS/Certificate Issues](#tls-certificate-issues)
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
./mcp-server-prtg status
```

**Check logs:**
```bash
# Linux/macOS
cat logs/mcp-server-prtg.log

# Windows (PowerShell)
Get-Content logs/mcp-server-prtg.log -Tail 50
```

#### Common Causes and Solutions

**1. Port Already in Use**

Error: `bind: address already in use` or `listen tcp :8443: bind: address already in use`

Solution:
```bash
# Linux/macOS - Find process using port 8443
sudo lsof -i :8443
sudo netstat -tlnp | grep 8443

# Windows (PowerShell)
netstat -ano | findstr :8443

# Kill the conflicting process or change port in config.yaml
```

Change port in `config.yaml`:
```yaml
server:
  port: 8444  # Use different port
```

**2. Database Connection Failed**

Error: `failed to connect to database` or `pq: password authentication failed`

Solution: See [Database Connection Issues](#database-connection-issues)

**3. Missing Configuration File**

Error: `failed to load configuration: failed to read config file`

Solution:
```bash
# Check if config.yaml exists
ls -la config.yaml

# If missing, run install to regenerate
./mcp-server-prtg install
```

**4. Permission Denied**

Error: `permission denied` or `failed to create log directory`

Solution:
```bash
# Linux/macOS - Grant permissions
sudo chown -R $USER:$USER .
chmod 755 .
chmod 600 config.yaml

# Windows - Run as Administrator
```

**5. Certificate Files Not Found**

Error: `failed to load certificate` or `no such file or directory`

Solution:
```bash
# Check if certificate files exist
ls -la certs/

# If missing, regenerate by removing and reinstalling
rm -rf certs/
./mcp-server-prtg install
```

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

Edit `config.yaml`:
```yaml
logging:
  level: debug
```

Restart service:
```bash
./mcp-server-prtg restart
```

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
# Linux/macOS
sudo killall mcp-server-prtg

# Windows (PowerShell) - Find and kill process
Get-Process mcp-server-prtg | Stop-Process -Force
```

**Check for hung connections:**
```bash
# Linux/macOS
netstat -an | grep 8443

# Windows (PowerShell)
netstat -ano | findstr :8443
```

## Database Connection Issues

### Connection Failed

#### Symptom
Error: `failed to connect to database` or `connection refused`

#### Diagnosis

**Test database connectivity:**
```bash
# Using psql
psql -h localhost -p 5432 -U prtg_reader -d prtg_data_exporter

# Using telnet
telnet localhost 5432
```

#### Solutions

**1. PostgreSQL Not Running**

Check PostgreSQL status:
```bash
# Linux
sudo systemctl status postgresql

# Windows
Get-Service postgresql*

# macOS
brew services list | grep postgresql
```

Start PostgreSQL:
```bash
# Linux
sudo systemctl start postgresql

# Windows
Start-Service postgresql-x64-14

# macOS
brew services start postgresql
```

**2. Wrong Host or Port**

Verify configuration in `config.yaml`:
```yaml
database:
  host: "localhost"  # Try 127.0.0.1 if localhost doesn't work
  port: 5432
```

**3. Database Doesn't Exist**

Error: `pq: database "prtg_data_exporter" does not exist`

Solution:
```bash
# Create database (or use PRTG Data Exporter)
createdb -U postgres prtg_data_exporter
```

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
listen_addresses = '*'  # Or specific IP

# pg_hba.conf - Add client IP
host    prtg_data_exporter    prtg_reader    192.168.1.0/24    md5
```

### Authentication Failed

#### Symptom
Error: `pq: password authentication failed for user "prtg_reader"`

#### Solutions

**1. Wrong Password**

Update password in `config.yaml`:
```yaml
database:
  password: "correct_password_here"
```

Or use environment variable:
```bash
export PRTG_DB_PASSWORD="correct_password"
./mcp-server-prtg run
```

**2. User Doesn't Exist**

Create user:
```sql
-- Connect as postgres superuser
psql -U postgres

-- Create user
CREATE USER prtg_reader WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE prtg_data_exporter TO prtg_reader;
GRANT USAGE ON SCHEMA public TO prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO prtg_reader;
```

**3. pg_hba.conf Configuration**

Check PostgreSQL client authentication:
```bash
# Find pg_hba.conf
psql -U postgres -c "SHOW hba_file;"

# Edit pg_hba.conf - Add line
host    prtg_data_exporter    prtg_reader    127.0.0.1/32    md5

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
MCP tools don't appear in MCP Client's tools panel.

#### Diagnosis

**1. Check mcp-remote is accessible:**
```bash
npx mcp-remote --version
```

**2. Check MCP Client configuration:**
```bash
# macOS (Claude Desktop)
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Windows (PowerShell)
Get-Content $env:APPDATA\Claude\claude_desktop_config.json

# Linux
cat ~/.config/Claude/claude_desktop_config.json
```

**3. Check server is running:**
```bash
curl https://localhost:8443/health
```

#### Solutions

**1. Verify Node.js and npm**
```bash
node --version
npm --version
# mcp-remote will be automatically installed via npx
```

**2. Fix Configuration File**

Ensure valid JSON syntax:
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
        "MCP_SERVER_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

**Common JSON errors:**
- Missing commas between items
- Trailing comma after last item (not allowed)
- Wrong quotes (use `"` not `'`)
- Missing closing braces

**3. Restart MCP Client Completely**
```bash
# macOS - Quit completely (Claude Desktop)
killall Claude

# Windows - Task Manager
# End all Claude.exe processes

# Then restart MCP Client
```

**4. Check API Key**

Get API key from config:
```bash
grep api_key config.yaml
```

Ensure it matches in `claude_desktop_config.json` (or your MCP client config).

### Tools Appear But Don't Work

#### Symptom
Tools are visible but return errors when used.

#### Diagnosis

Check MCP Client logs:
```bash
# macOS (Claude Desktop)
cat ~/Library/Logs/Claude/mcp*.log

# Windows
Get-Content $env:APPDATA\Claude\logs\mcp*.log

# Linux
cat ~/.config/Claude/logs/mcp*.log
```

#### Solutions

**1. Authentication Error**

Error: `Unauthorized` or `401`

Solution: Verify API key matches between `config.yaml` and `claude_desktop_config.json` (or your MCP client config)

**2. Certificate Error (Self-Signed)**

Error: `SSL certificate verify failed` or `certificate verification failed`

Solution: Add `NODE_TLS_REJECT_UNAUTHORIZED=0` to env (development only):
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
        "MCP_SERVER_API_KEY": "your-api-key",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0"
      }
    }
  }
}
```

**Note:** Not recommended for production. Use trusted certificates instead.

**3. Database Connection Error**

Check server logs and verify database connectivity.

## Connection Issues

### Connection Refused

#### Symptom
Error: `connection refused` or `unable to connect`

#### Solutions

**1. Verify Server is Running**
```bash
./mcp-server-prtg status

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

#### Diagnosis

Check logs for patterns:
```bash
tail -f logs/mcp-server-prtg.log
```

#### Solutions

**1. Network Issues**
- Check network stability
- Increase TCP keepalive settings
- Use wired connection instead of WiFi

**2. Proxy or Firewall Interference**
- Configure proxy to allow long-lived connections
- Whitelist server IP/domain
- Ensure proxies don't interfere with heartbeat mechanism (30s interval)

**3. Resource Exhaustion**
- Check server CPU/memory usage
- Increase system resources

**4. Heartbeat Issues**
The server sends heartbeats every 30 seconds to keep connections alive. If heartbeats are being blocked:
- Check proxy configuration
- Verify firewall allows bidirectional traffic
- Review network equipment (load balancers, firewalls) timeout settings

## Authentication Issues

### Unauthorized (401) Errors

#### Symptom
Error: `Unauthorized: Missing or invalid Bearer token`

#### Solutions

**1. Verify API Key**

Get from config:
```bash
grep api_key config.yaml
```

Test manually:
```bash
curl -H "Authorization: Bearer your-api-key" \
     https://localhost:8443/status
```

**2. Check Header Format**

Correct format:
```
Authorization: Bearer your-api-key
```

**NOT:**
- `Authorization: your-api-key` (missing "Bearer")
- `Authorization: bearer your-api-key` (lowercase "bearer")
- `Authorization: Bearer: your-api-key` (extra colon)

**3. API Key Changed**

If config was hot-reloaded with new API key:
- Update client configuration
- Restart MCP Client
- Reconnect clients

### WWW-Authenticate Header Issues

#### Symptom
Repeated authentication prompts or errors.

#### Solution

This is normal behavior when authentication fails. The server sends:
```
WWW-Authenticate: Bearer realm="MCP Server PRTG"
```

Check logs for specific authentication failure reason.

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

#### Solutions

**Option 1: Disable Certificate Verification (Development Only)**

Add `NODE_TLS_REJECT_UNAUTHORIZED=0` to environment variables:

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
        "MCP_SERVER_API_KEY": "your-api-key",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0"
      }
    }
  }
}
```

**⚠️ WARNING:** This disables certificate validation. Only use for development with self-signed certificates. Never use in production.

**Option 2: Use HTTP Instead of HTTPS (Development Only)**

Disable TLS in server configuration:

1. Edit `config.yaml`:
```yaml
server:
  enable_tls: false  # Disable TLS
  port: 8443
```

2. Restart the server:
```bash
./mcp-server-prtg restart
```

3. Update MCP Client configuration to use `http://`:
```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "http://localhost:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "your-api-key"
      }
    }
  }
}
```

**Note:** Even with HTTP, your API key authentication is still secure via the Bearer token.

**Option 3: Use Trusted CA Certificates (Production - Recommended)**

For production environments, use certificates from a trusted Certificate Authority (e.g., Let's Encrypt):

```yaml
server:
  enable_tls: true
  cert_file: "/etc/letsencrypt/live/domain.com/fullchain.pem"
  key_file: "/etc/letsencrypt/live/domain.com/privkey.pem"
```

Then use HTTPS without disabling verification:
```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://your-domain.com:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "your-api-key"
      }
    }
  }
}
```

**Option 3: Add Self-Signed Certificate to System Trust Store (Advanced)**

You can configure your operating system to trust the self-signed certificate:

**macOS:**
```bash
# Export server certificate
openssl s_client -connect localhost:8443 -showcerts < /dev/null 2>/dev/null | \
  openssl x509 -outform PEM > server.crt

# Add to system keychain
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain server.crt
```

**Linux:**
```bash
# Copy certificate to trusted store
sudo cp certs/server.crt /usr/local/share/ca-certificates/mcp-server-prtg.crt
sudo update-ca-certificates
```

**Windows:**
```powershell
# Import certificate to Trusted Root
Import-Certificate -FilePath "certs\server.crt" -CertStoreLocation Cert:\LocalMachine\Root
```

After adding the certificate to the system trust store, restart your MCP Client.

**Testing with curl:**

```bash
# HTTP (no SSL)
curl http://localhost:8443/health

# HTTPS with self-signed cert (bypass verification for testing)
curl -k https://localhost:8443/health
```

### Certificate Expired

#### Symptom
Error: `certificate has expired`

#### Solution

**For self-signed certificates:**

Regenerate certificates:
```bash
# Remove old certificates
rm -rf certs/

# Edit config to trigger regeneration
# Change enable_tls to false, save, then back to true
# Or manually regenerate:
./mcp-server-prtg install  # Regenerates certificates
```

**For trusted certificates:**

Renew certificate with your CA:
```bash
# Example with Let's Encrypt
sudo certbot renew
```

### Wrong Host Name in Certificate

#### Symptom
Error: `certificate is valid for localhost, not for 192.168.1.100`

#### Solution

**Option 1: Use hostname in URL**
```
https://localhost:8443/mcp  # Instead of IP address
```

**Option 2: Generate certificate with correct SANs**

Regenerate certificate with additional Subject Alternative Names including your server's IP or hostname.

## Performance Issues

### Slow Query Responses

#### Symptom
Queries take a long time to return results.

#### Diagnosis

Enable debug logging:
```yaml
logging:
  level: debug
```

Check logs for query durations:
```bash
grep "query_duration_ms" logs/mcp-server-prtg.log
```

#### Solutions

**1. Reduce Result Set Size**

Use smaller limits:
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "limit": 50  // Instead of 1000
  }
}
```

**2. Add More Specific Filters**

Filter by device or sensor name:
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "device_name": "web-prod",
    "limit": 100
  }
}
```

**3. Database Performance**

Check PostgreSQL performance:
```sql
-- Connect to database
psql -U prtg_reader -d prtg_data_exporter

-- Check slow queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active'
ORDER BY duration DESC;

-- Check table sizes
SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

Add indexes if needed:
```sql
-- Example indexes
CREATE INDEX idx_sensor_device_id ON prtg_sensor(prtg_device_id);
CREATE INDEX idx_sensor_status ON prtg_sensor(status);
CREATE INDEX idx_sensor_name ON prtg_sensor(name);
```

**4. Increase Query Timeout**

Currently hard-coded to 30 seconds. If you need longer timeouts, you'll need to modify the source code.

### High Memory Usage

#### Symptom
Server uses excessive memory or runs out of memory.

#### Solutions

**1. Reduce Connection Pool Size**

This requires code modification. Default is 25 connections.

**2. Limit Result Sets**

Always use `limit` parameters to prevent large result sets.

**3. Increase Server Memory**

Add more RAM or adjust system limits:
```bash
# Linux - Check limits
ulimit -a

# Increase memory limit
ulimit -v unlimited
```

### High CPU Usage

#### Symptom
Server uses 100% CPU continuously.

#### Solutions

**1. Check for Query Loops**

Review queries that might be executing repeatedly.

**2. Database Connection Issues**

Failed database connections with retries can cause high CPU usage.

**3. Restart Service**

Sometimes a restart resolves temporary issues:
```bash
./mcp-server-prtg restart
```

## Configuration Issues

### Invalid YAML Syntax

#### Symptom
Error: `yaml: unmarshal errors` or `failed to parse config file`

#### Solutions

**Validate YAML syntax:**
```bash
# Online: https://www.yamllint.com/

# Using Python
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Using yq (if installed)
yq eval config.yaml
```

**Common YAML errors:**
- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted special characters
- Wrong data types

**Example of common mistakes:**
```yaml
# ❌ Wrong - tabs instead of spaces
server:
	port: 8443

# ✅ Correct - spaces for indentation
server:
  port: 8443

# ❌ Wrong - missing quotes around special characters
api_key: a1b2c3:d4e5

# ✅ Correct - quoted
api_key: "a1b2c3:d4e5"

# ❌ Wrong - string instead of integer
port: "8443"

# ✅ Correct - integer
port: 8443
```

### Hot-Reload Not Working

#### Symptom
Changes to `config.yaml` don't take effect.

#### Diagnosis

Check logs for file watcher errors:
```bash
grep "watcher" logs/mcp-server-prtg.log
```

#### Solutions

**1. File System Doesn't Support File Watching**

Some network file systems (NFS, SMB) don't support inotify/FSEvents.

Solution: Restart service manually after changes:
```bash
./mcp-server-prtg restart
```

**2. Config File Was Replaced (Not Modified)**

If you delete and recreate the file, the watcher loses track.

Solution: Always edit the file in-place instead of replacing it.

**3. Changes Require Restart**

Some settings can't be hot-reloaded:
- `server.port`
- `server.bind_address`
- `server.enable_tls`

Solution: Restart the service.

## Logging and Diagnostics

### Enable Debug Logging

**Temporary (until restart):**
Edit `config.yaml`:
```yaml
logging:
  level: debug
```

Hot-reload will apply changes immediately.

**Permanent:**
Keep debug level in config file.

### Check Logs

**View recent logs:**
```bash
# Linux/macOS
tail -f logs/mcp-server-prtg.log

# Windows (PowerShell)
Get-Content logs/mcp-server-prtg.log -Wait -Tail 50
```

**Search logs:**
```bash
# Find errors
grep "error" logs/mcp-server-prtg.log

# Find specific sensor ID
grep "sensor_id.*12345" logs/mcp-server-prtg.log

# Find authentication failures
grep "Unauthorized" logs/mcp-server-prtg.log
```

### Log Rotation Issues

#### Symptom
Log files grow too large or disk fills up.

#### Solutions

**1. Adjust Rotation Settings**

In `config.yaml`:
```yaml
logging:
  max_size_mb: 10      # Rotate after 10 MB
  max_backups: 5       # Keep 5 old files
  max_age_days: 30     # Delete files older than 30 days
  compress: true       # Compress old logs
```

**2. Manually Clean Old Logs**
```bash
# Remove old logs
rm logs/*.log.gz

# Remove all but current log
ls logs/*.log | grep -v mcp-server-prtg.log | xargs rm
```

### Collect Diagnostic Information

When reporting issues, collect:

```bash
# 1. Version
./mcp-server-prtg --version

# 2. Status
./mcp-server-prtg status

# 3. Configuration (REDACT PASSWORD and API KEY)
cat config.yaml | grep -v password | grep -v api_key

# 4. Recent logs
tail -n 100 logs/mcp-server-prtg.log

# 5. System info
uname -a
go version

# 6. Network status
netstat -an | grep 8443

# 7. Database connectivity
psql -h localhost -U prtg_reader -d prtg_data_exporter -c "SELECT version();"
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

4. **Community support:**
   - Check GitHub Discussions
   - Review closed issues for similar problems

## See Also

- [Configuration Guide](CONFIGURATION.md)
- [Usage Guide](USAGE.md)
- [Tools Reference](TOOLS.md)
- [Architecture](ARCHITECTURE.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
