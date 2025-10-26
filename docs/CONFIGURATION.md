# Configuration Guide

This guide covers all configuration options for MCP Server PRTG.

## Table of Contents

- [Configuration File Location](#configuration-file-location)
- [Configuration File Structure](#configuration-file-structure)
- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [Logging Configuration](#logging-configuration)
- [Environment Variables](#environment-variables)
- [TLS/HTTPS Setup](#tlshttps-setup)
- [Hot-Reload](#hot-reload)
- [Advanced Configuration](#advanced-configuration)

## Configuration File Location

The configuration file is named `config.yaml` and is located by default in the same directory as the executable.

You can specify a custom location using the `--config` flag:

```bash
./mcp-server-prtg run --config /path/to/config.yaml
```

On first installation, the configuration file is automatically generated with:
- A unique API key (UUID v4)
- Self-signed TLS certificates (if TLS is enabled)
- Default PostgreSQL configuration

## Configuration File Structure

The configuration file uses YAML format with the following structure:

```yaml
config_version: 1

server:
  api_key: "your-generated-api-key"
  bind_address: "0.0.0.0"
  port: 8443
  public_url: ""
  enable_tls: true
  cert_file: "./certs/server.crt"
  key_file: "./certs/server.key"
  read_timeout: 10
  write_timeout: 10

database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: ""
  sslmode: "disable"

logging:
  level: "info"
  file: "logs/mcp-server-prtg.log"
  max_size_mb: 10
  max_backups: 5
  max_age_days: 30
  compress: true
```

## Server Configuration

### api_key

**Type:** `string`
**Required:** Yes
**Description:** Bearer token used for authentication (RFC 6750).

Automatically generated during installation as a UUID v4. This key must be provided by clients in the `Authorization` header:

```
Authorization: Bearer your-generated-api-key
```

**Security Note:** Keep this key secure. File permissions are automatically set to `0600` (owner read/write only).

### bind_address

**Type:** `string`
**Default:** `"0.0.0.0"`
**Description:** IP address to bind the server to.

- `0.0.0.0` - Bind to all network interfaces (allows remote connections)
- `127.0.0.1` - Bind to localhost only (local connections only)
- Specific IP - Bind to a specific network interface

### port

**Type:** `integer`
**Default:** `8443`
**Description:** Port number for the HTTPS/HTTP server.

Common ports:
- `8443` - Standard alternative HTTPS port
- `443` - Standard HTTPS port (requires root/administrator privileges)
- `8080` - Standard HTTP port

### public_url

**Type:** `string`
**Optional:** Yes
**Description:** Public URL for the SSE endpoint.

If your server is behind a reverse proxy or has a public domain, specify it here:

```yaml
public_url: "https://prtg.example.com:8443"
```

If not specified, the server will construct a URL from `bind_address` and `port`.

### enable_tls

**Type:** `boolean`
**Default:** `true`
**Description:** Enable HTTPS with TLS encryption.

**Recommendation:** Always use `true` in production. TLS encrypts all communication including API keys.

### cert_file

**Type:** `string`
**Default:** `"./certs/server.crt"`
**Description:** Path to TLS certificate file (PEM format).

Automatically generated on first run if `enable_tls` is true and no certificate exists.

### key_file

**Type:** `string`
**Default:** `"./certs/server.key"`
**Description:** Path to TLS private key file (PEM format).

Automatically generated on first run if `enable_tls` is true and no key exists.

**Security Note:** File permissions are automatically set to `0600` (owner read/write only).

### read_timeout / write_timeout

**Type:** `integer` (seconds)
**Default:** `10`
**Description:** HTTP timeouts for regular endpoints.

**Note:** SSE endpoints use infinite timeouts (0) to maintain long-lived connections.

## Database Configuration

### host

**Type:** `string`
**Default:** `"localhost"`
**Description:** PostgreSQL server hostname or IP address.

Examples:
- `localhost` - Local PostgreSQL server
- `192.168.1.100` - Remote server by IP
- `pgsql.example.com` - Remote server by hostname

### port

**Type:** `integer`
**Default:** `5432`
**Description:** PostgreSQL server port.

### name

**Type:** `string`
**Default:** `"prtg_data_exporter"`
**Description:** Database name containing PRTG monitoring data.

This database should be created by the [PRTG Data Exporter](https://github.com/senhub-io/prtg-data-exporter).

### user

**Type:** `string`
**Default:** `"prtg_reader"`
**Description:** PostgreSQL username for read-only access.

**Security Best Practice:** Use a read-only user with minimal permissions:

```sql
CREATE USER prtg_reader WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE prtg_data_exporter TO prtg_reader;
GRANT USAGE ON SCHEMA public TO prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO prtg_reader;
```

### password

**Type:** `string`
**Required:** Yes
**Description:** PostgreSQL user password.

**Security Note:** File permissions are automatically set to `0600` to protect the password.

**Tip:** You can also use the `PRTG_DB_PASSWORD` environment variable (see [Environment Variables](#environment-variables)).

### sslmode

**Type:** `string`
**Default:** `"disable"`
**Description:** PostgreSQL SSL mode.

Options:
- `disable` - No SSL (not recommended for remote connections)
- `require` - Require SSL (recommended for remote connections)
- `verify-ca` - Require SSL and verify CA certificate
- `verify-full` - Require SSL and verify server certificate

**Production Recommendation:** Use `require` or higher for remote database connections.

## Logging Configuration

MCP Server PRTG uses structured logging with rotation support (via [lumberjack](https://github.com/natefinch/lumberjack)).

### level

**Type:** `string`
**Default:** `"info"`
**Description:** Minimum log level to record.

Levels (from most to least verbose):
- `debug` - Detailed diagnostic information (SQL queries, request/response details)
- `info` - General informational messages (startup, shutdown, connections)
- `warn` - Warning messages (non-critical issues)
- `error` - Error messages (failures, exceptions)

**Development:** Use `debug` for troubleshooting
**Production:** Use `info` or `warn`

### file

**Type:** `string`
**Default:** `"logs/mcp-server-prtg.log"`
**Description:** Log file path.

- Relative paths are relative to the executable directory
- Absolute paths are supported
- Directory is created automatically if it doesn't exist

### max_size_mb

**Type:** `integer`
**Default:** `10`
**Description:** Maximum log file size in megabytes before rotation.

When the log file reaches this size, it's rotated (renamed with a timestamp) and a new file is created.

### max_backups

**Type:** `integer`
**Default:** `5`
**Description:** Maximum number of old log files to retain.

Older files are automatically deleted when this limit is exceeded.

### max_age_days

**Type:** `integer`
**Default:** `30`
**Description:** Maximum age of log files in days before deletion.

Log files older than this are automatically deleted.

### compress

**Type:** `boolean`
**Default:** `true`
**Description:** Compress rotated log files with gzip.

Rotated files are renamed from `.log` to `.log.gz`, saving disk space.

## Environment Variables

Environment variables can be used to override configuration file settings. This is useful for Docker containers or CI/CD pipelines.

### PRTG_DB_PASSWORD

**Description:** Database password (overrides `database.password` in config file)

```bash
export PRTG_DB_PASSWORD="secure_password"
./mcp-server-prtg run
```

**Note:** Currently, only the database password supports environment variable override. Other settings must be specified in the configuration file.

## TLS/HTTPS Setup

### Automatic Self-Signed Certificates

On first installation with `enable_tls: true`, the server automatically generates self-signed certificates:

- **Certificate:** `./certs/server.crt`
- **Private Key:** `./certs/server.key`
- **Validity:** 1 year
- **Subject:** CN=localhost, O=MCP Server PRTG
- **SANs:** localhost, 127.0.0.1, ::1

Self-signed certificates are suitable for:
- Development and testing
- Internal networks where clients can trust the certificate
- Use with mcp-proxy (which supports self-signed certificates)

### Using Custom Certificates

For production with public domains, use certificates from a trusted Certificate Authority (Let's Encrypt, etc.):

1. Obtain your certificate and private key files
2. Update the configuration:

```yaml
server:
  enable_tls: true
  cert_file: "/path/to/your/certificate.crt"
  key_file: "/path/to/your/private.key"
```

3. Restart the service

### Certificate Requirements

- **Format:** PEM encoded
- **Type:** X.509 certificate
- **Key Type:** RSA 2048-bit or higher (recommended)

### Disabling TLS (Not Recommended)

For development only:

```yaml
server:
  enable_tls: false
  port: 8080  # Use standard HTTP port
```

**Warning:** All communication including API keys will be transmitted in plain text.

## Hot-Reload

MCP Server PRTG supports hot-reload of the configuration file using file system watching (via [fsnotify](https://github.com/fsnotify/fsnotify)).

### How It Works

1. The server watches `config.yaml` for changes
2. When the file is modified and saved, the server automatically reloads the configuration
3. Changes take effect immediately without restarting the service

### What Can Be Hot-Reloaded

- Logging level (`logging.level`)
- Database connection settings (new connections use new settings)
- API key (requires clients to reconnect with new key)

### What Requires a Restart

- Server bind address (`server.bind_address`)
- Server port (`server.port`)
- TLS enable/disable (`server.enable_tls`)
- Certificate files (changes require restart)

### Example

```bash
# Server is running
./mcp-server-prtg run

# In another terminal, change log level
sed -i 's/level: info/level: debug/' config.yaml

# Changes are applied automatically (visible in logs)
```

## Advanced Configuration

### Database Connection Pooling

The server uses [pgx](https://github.com/jackc/pgx) for database connections with automatic connection pooling:

- **Default Pool Size:** 25 connections
- **Max Idle Time:** 30 minutes
- **Max Lifetime:** 1 hour

These settings are optimized for typical workloads and cannot currently be configured.

### Query Timeouts

Database queries have a 30-second timeout to prevent long-running queries from blocking the server. This is hard-coded and cannot be changed.

### Security Headers

The server automatically sets security headers for HTTP responses:

- `WWW-Authenticate: Bearer realm="MCP Server PRTG"` (on 401 responses)

### SSE Transport Configuration

SSE (Server-Sent Events) configuration is optimized for long-lived connections:

- **Read Timeout:** 0 (infinite)
- **Write Timeout:** 0 (infinite)
- **Idle Timeout:** 0 (infinite)
- **Architecture:** v2 with internal server + authentication proxy

### File Permissions

The server automatically sets secure file permissions:

- `config.yaml`: 0600 (owner read/write only)
- `server.key`: 0600 (owner read/write only)
- `server.crt`: 0600 (owner read/write only)
- Log files: 0640 (owner read/write, group read)

## Configuration Examples

### Minimal Configuration (Local Development)

```yaml
config_version: 1
server:
  api_key: "dev-key-123"
  bind_address: "127.0.0.1"
  port: 8443
  enable_tls: false
database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: "password"
  sslmode: "disable"
logging:
  level: "debug"
  file: "logs/mcp-server-prtg.log"
```

### Production Configuration (Remote Access)

```yaml
config_version: 1
server:
  api_key: "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
  bind_address: "0.0.0.0"
  port: 8443
  public_url: "https://prtg-mcp.example.com:8443"
  enable_tls: true
  cert_file: "/etc/ssl/certs/server.crt"
  key_file: "/etc/ssl/private/server.key"
  read_timeout: 10
  write_timeout: 10
database:
  host: "pgsql.internal.example.com"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: "strong_random_password_here"
  sslmode: "require"
logging:
  level: "info"
  file: "/var/log/mcp-server-prtg/server.log"
  max_size_mb: 50
  max_backups: 10
  max_age_days: 90
  compress: true
```

### Docker Configuration

```yaml
config_version: 1
server:
  api_key: "docker-api-key"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
  cert_file: "/certs/server.crt"
  key_file: "/certs/server.key"
database:
  host: "postgres"  # Docker service name
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: ""  # Use PRTG_DB_PASSWORD env var
  sslmode: "disable"
logging:
  level: "info"
  file: "/logs/mcp-server-prtg.log"
  max_size_mb: 10
  max_backups: 3
  max_age_days: 7
  compress: true
```

## Troubleshooting

### Configuration File Not Found

**Error:** `failed to load configuration: failed to read config file`

**Solution:** Ensure `config.yaml` exists in the executable directory or specify path with `--config`.

### Invalid YAML Syntax

**Error:** `failed to parse config file: yaml: unmarshal errors`

**Solution:** Validate YAML syntax. Common issues:
- Incorrect indentation (use spaces, not tabs)
- Missing quotes around special characters
- Incorrect data types (string vs integer)

### Permission Denied

**Error:** `failed to write config file: permission denied`

**Solution:** Ensure the user running the service has write permissions to the config directory.

### Hot-Reload Not Working

**Issue:** Changes to config file don't take effect

**Solutions:**
- Check logs for file watcher errors
- Ensure file system supports inotify/FSEvents/ReadDirectoryChangesW
- Try restarting the service
- Verify file permissions allow reading the config file

## See Also

- [Installation Guide](INSTALLATION.md)
- [Usage Guide](USAGE.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
