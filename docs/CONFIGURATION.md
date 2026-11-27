# Configuration Guide

This guide covers all configuration options for MCP Server PRTG.

## Table of Contents

- [Configuration File Location](#configuration-file-location)
- [Editing the Configuration](#editing-the-configuration)
- [Configuration File Structure](#configuration-file-structure)
- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [PRTG API v2 Configuration](#prtg-api-v2-configuration)
- [Logging Configuration](#logging-configuration)
- [Encrypted Secrets](#encrypted-secrets)
- [Environment Variables](#environment-variables)
- [config Commands](#config-commands)
- [TLS/HTTPS Setup](#tlshttps-setup)
- [Configuration Examples](#configuration-examples)

## Configuration File Location

The configuration file is named `config.yaml`.

- **Windows (MSI):** `C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml`
- **Linux:** `/etc/mcp-server-prtg/config.yaml`

You can specify a custom location using the `--config` flag:

```bash
mcp-server-prtg run --config /path/to/config.yaml
```

## Editing the Configuration

The recommended way to edit the configuration is the administration page:

```
https://localhost:<port>/admin/
```

Sign in with the MCP API key. The **Settings** tab lets you change the database settings, the PRTG API v2 settings, the MCP port and the log level. Use **Test connection** to check the database or PRTG values before saving. Secrets are never shown: leave a secret field empty to keep the stored value. A stored secret is only reused for the same database host/port/user without weakening the SSL mode, or for the same PRTG URL without turning off certificate verification; otherwise it must be typed again. Use **Save** or **Save and restart** (the service exits and the service manager starts it again).

Alternatively, use `config init` (see [config Commands](#config-commands)) and then restart the service.

## Configuration File Structure

The configuration file uses YAML format with the following structure:

```yaml
config_version: 1

server:
  api_key: "enc:dpapi:..."        # or enc:aes:... on Linux; see Encrypted Secrets
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
  cert_file: "./certs/server.crt"
  key_file: "./certs/server.key"
  read_timeout: 10
  write_timeout: 10
  allow_custom_queries: false

database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter_collection"
  user: "mcp_prtg_reader"
  password: "enc:dpapi:..."        # or enc:aes:... on Linux
  sslmode: "disable"

prtg:
  enabled: false
  base_url: "https://prtg.example.com:1616"
  api_token: "enc:dpapi:..."       # or enc:aes:... on Linux
  timeout: 30
  verify_ssl: true

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

Only the `Authorization` header is accepted; the key cannot be passed as a URL query parameter.

**Requirements and protections:**
- The server refuses to start if `api_key` is empty or shorter than 16 characters (the generated key is a 36-character UUID).
- The key is compared in constant time.
- Failed authentication attempts are rate limited per client address (TCP peer address; IPv6 addresses are grouped by /64; `X-Forwarded-For` and `X-Real-IP` are ignored): after 5 failures within one minute, the address receives HTTP 429 for 5 minutes. Behind a reverse proxy, all clients share the proxy address and therefore the same limit.

**Security Note:** Keep this key secure. File permissions are automatically set to `0600` (owner read/write only). The value is stored encrypted (see [Encrypted Secrets](#encrypted-secrets)).

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

### enable_tls

**Type:** `boolean`
**Default:** `true`
**Description:** Enable HTTPS with TLS encryption.

**Recommendation:** Always use `true` in production. TLS encrypts all communication including API keys.

### cert_file

**Type:** `string`
**Default:** `certs/server.crt` in a `certs` folder next to the configuration file
**Description:** Path to TLS certificate file (PEM format).

Automatically generated at start when missing if `enable_tls` is true.

### key_file

**Type:** `string`
**Default:** `certs/server.key` in a `certs` folder next to the configuration file
**Description:** Path to TLS private key file (PEM format).

**Security Note:** File permissions are automatically set to `0600` (owner read/write only).

### read_timeout / write_timeout

**Type:** `integer` (seconds)
**Default:** `10`
**Description:** HTTP timeouts for regular endpoints.

**Note:** The `/mcp` endpoint uses optimized timeouts for Streamable HTTP:
- ReadTimeout: 0 (no timeout for streaming connections)
- WriteTimeout: 0 (no timeout for streaming connections)
- IdleTimeout: 60 minutes (close inactive connections)

### allow_custom_queries

**Type:** `boolean`
**Default:** `false`
**Description:** Enable or disable the `prtg_query_sql` tool for custom SQL queries.

**SECURITY WARNING:**
- **Default is `false` (DISABLED)** for security reasons
- Enabling this feature allows execution of arbitrary SELECT queries on your database
- **NEVER enable in production environments** unless absolutely necessary

**Protections in place:**
- Only SELECT queries allowed (INSERT, UPDATE, DELETE, DROP blocked)
- Queries run in a read-only transaction (read-only sandbox) with a 15-second statement timeout
- Results capped at 1 to 1000 rows (default 100), regardless of any `LIMIT` in the query
- Multiple statements and SQL comments are rejected
- References to PRTG Data Exporter configuration and credential tables (`prtg_server_address`, `target_database`, `grafana_*`, `licensing_*`, `setting`, `audit_trail`) are rejected
- Functions `pg_*`, `dblink*`, `lo_*`, `current_setting` and `set_config` are rejected

## Database Configuration

The server reads the PostgreSQL **target** database of PRTG Data Exporter 2026.x, default name `prtg_data_exporter_collection`. This is NOT the Data Exporter configuration database `prtg_data_exporter`, which also stores PRTG credentials and must stay out of reach. Older Data Exporter layouts still work: the schema is detected at start-up (soft delete, tree depth, channel values, collection status, per-sensor last fetch).

### host

**Type:** `string`
**Default:** `"localhost"`
**Description:** PostgreSQL server hostname or IP address.

### port

**Type:** `integer`
**Default:** `5432`
**Description:** PostgreSQL server port.

### name

**Type:** `string`
**Default:** `"prtg_data_exporter_collection"`
**Description:** Name of the PRTG Data Exporter 2026.x target database containing the monitoring data.

### user

**Type:** `string`
**Default:** `"mcp_prtg_reader"`
**Description:** PostgreSQL username for read-only access.

**Recommended read-only role on the target database:**

```sql
CREATE ROLE mcp_prtg_reader LOGIN;
\password mcp_prtg_reader
GRANT CONNECT ON DATABASE prtg_data_exporter_collection TO mcp_prtg_reader;
-- in the prtg_data_exporter_collection database:
GRANT USAGE ON SCHEMA public TO mcp_prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO mcp_prtg_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE prtg_data_exporter_user_td IN SCHEMA public GRANT SELECT ON TABLES TO mcp_prtg_reader;
```

### password

**Type:** `string`
**Required:** Yes
**Description:** PostgreSQL user password. Stored encrypted (see [Encrypted Secrets](#encrypted-secrets)).

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

## PRTG API v2 Configuration

PRTG API v2 access is **optional**. If not configured, only the database-backed tools are available.

**PRTG API v2 activation:** The PRTG API v2 must be activated on the PRTG core server (PRTG Application Server, ports 1616/1615 and 23580; default on new installs since 25.2.106, automatic on update to 26.4.126 when enough memory is free; otherwise Setup → Activate, then restart the core server).

**Request guard:** Every PRTG API call goes through a request guard: responses are cached, identical concurrent requests are coalesced into a single call, and at most 2 requests are in flight at a time.

**Configuration block:**
```yaml
prtg:
  enabled: true
  base_url: "https://prtg.example.com:1616"
  api_token: "your-prtg-api-key"
  timeout: 30
  verify_ssl: true
```

### enabled

**Type:** `boolean`
**Default:** `false`
**Description:** Enable/disable PRTG API v2 access.

### base_url

**Type:** `string`
**Required:** Yes (if `enabled: true`)
**Description:** PRTG Core Server URL including scheme and port.

**Important:**
- Include the protocol (http:// or https://) and the API v2 port (typically **1616**, NOT 443)
- Do NOT include a path (like /api/v2)

**Examples:**
```yaml
base_url: "https://prtg.example.com:1616"
base_url: "https://192.168.1.100:1616"
base_url: "http://localhost:1616"
```

### api_token

**Type:** `string`
**Required:** Yes (if `enabled: true`)
**Description:** A PRTG API key of type **Scripting** with read access.

**How to get the token:**
1. Log in to the PRTG Web Interface
2. Go to **Setup** → **Account Settings** → **My Account**
3. Navigate to **API Keys & Tokens**
4. Create a new API key of type Scripting with read access
5. Copy the token value

**Note:** This is different from the MCP Server `server.api_key`. The token is stored encrypted (see [Encrypted Secrets](#encrypted-secrets)).

### timeout

**Type:** `integer`
**Default:** `30`
**Description:** HTTP request timeout in seconds for PRTG API calls.

### verify_ssl

**Type:** `boolean`
**Default:** `true`
**Description:** Verify SSL/TLS certificates when connecting to the PRTG API.

- `true` - Verify certificates (recommended for production)
- `false` - Skip verification (use for self-signed certificates)

## Logging Configuration

MCP Server PRTG uses structured logging with rotation support (via [lumberjack](https://github.com/natefinch/lumberjack)). Secrets and tokens are masked in log output.

### level

**Type:** `string`
**Default:** `"info"`
**Description:** Minimum log level to record: `debug`, `info`, `warn`, `error`.

### file

**Type:** `string`
**Default:** `"logs/mcp-server-prtg.log"`
**Description:** Log file path. Relative paths are relative to the executable directory; absolute paths are supported.

### max_size_mb

**Type:** `integer`
**Default:** `10`
**Description:** Maximum log file size in megabytes before rotation.

### max_backups

**Type:** `integer`
**Default:** `5`
**Description:** Maximum number of old log files to retain.

### max_age_days

**Type:** `integer`
**Default:** `30`
**Description:** Maximum age of log files in days before deletion.

### compress

**Type:** `boolean`
**Default:** `true`
**Description:** Compress rotated log files with gzip.

## Encrypted Secrets

The secrets `server.api_key`, `database.password` and `prtg.api_token` are stored encrypted in `config.yaml`:

- **Windows:** `enc:dpapi:...` - DPAPI machine scope; only that machine can decrypt.
- **Linux:** `enc:aes:...` - AES-256-GCM with the key in `secret.key` next to `config.yaml` (mode 0600). Back up `secret.key` together with `config.yaml`; losing it means re-entering the secrets.

You may type a clear-text value in `config.yaml`: it is encrypted in place at the next service start. A config file copied to another machine cannot be decrypted there: re-enter the secrets on that machine.

## Environment Variables

These environment variables feed the same options as the configuration file and are useful to keep secrets off the command line:

| Variable | Maps to |
|----------|---------|
| `PRTG_DB_HOST` | `database.host` |
| `PRTG_DB_PORT` | `database.port` |
| `PRTG_DB_NAME` | `database.name` |
| `PRTG_DB_USER` | `database.user` |
| `PRTG_DB_PASSWORD` | `database.password` |
| `PRTG_DB_SSLMODE` | `database.sslmode` |
| `PRTG_URL` | `prtg.base_url` |
| `PRTG_API_TOKEN` | `prtg.api_token` |

```bash
export PRTG_DB_PASSWORD="secure_password"
mcp-server-prtg run
```

## config Commands

```bash
mcp-server-prtg config init --config <path> [--https] [--port N] [--db-host H] [--db-port N] [--db-name N] [--db-user U] [--db-password P] [--db-sslmode M] [--prtg-url URL] [--prtg-token T] [--prtg-verify-ssl=true|false] [--prtg-timeout S] [--prtg-disable] [--api-key K] [--log-file F]
```

Creates the configuration, or updates only the given settings of an existing one (an existing API key is kept unless `--api-key` is given). Secrets passed on the command line are written encrypted.

```bash
mcp-server-prtg config show-api-key --config <path>
```

Prints the decrypted MCP API key.

```bash
mcp-server-prtg config client [--host <name>] [--config <path>]
```

Prints the ready-to-paste configurations for Claude Desktop, Claude Code, Cursor and VS Code. `config show-api-key` and `config client` find the installed configuration without `--config` (ProgramData on Windows, `/etc/mcp-server-prtg` on Linux).

The service commands are unchanged: `install`, `uninstall`, `start`, `stop`, `status`, `run`.

## TLS/HTTPS Setup

### Automatic Self-Signed Certificates

When `enable_tls: true` and no certificate exists, the server generates a self-signed certificate at start in a `certs` folder next to the configuration file:

- **Certificate:** `certs/server.crt`
- **Private Key:** `certs/server.key`

The certificate covers `localhost`, the machine name and its IP addresses. It is regenerated automatically at start when missing (delete the `certs` folder and restart to regenerate, for example after renaming the server).

### Trusting the Certificate

Clients must trust `server.crt` instead of disabling verification. Copy `server.crt` to each client computer:

- **Claude Desktop / mcp-remote:** set `NODE_EXTRA_CA_CERTS` to the path of a copy of the certificate.
- **Claude Code, Cursor, VS Code:** import into the computer's trusted root certificates (Windows: `certlm.msc` → Trusted Root Certification Authorities, or `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`; macOS: Keychain Access, "Always Trust").

Do NOT set `NODE_TLS_REJECT_UNAUTHORIZED=0`: it exposes the API key.

### Using Custom Certificates

A certificate from the company PKI can be used instead: set `server.cert_file` and `server.key_file` and restart the service.

### Disabling TLS (Not Recommended)

For development only: set `enable_tls: false`. All communication including API keys will be transmitted in plain text.

## Configuration Examples

### Minimal Configuration (Local Development)

```yaml
config_version: 1
server:
  api_key: "dev-key-0123456789abcdef"  # At least 16 characters; encrypted in place at next start
  bind_address: "127.0.0.1"
  port: 8443
  enable_tls: false
  allow_custom_queries: false
database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter_collection"
  user: "mcp_prtg_reader"
  password: "password"
  sslmode: "disable"
prtg:
  enabled: false
logging:
  level: "debug"
  file: "logs/mcp-server-prtg.log"
```

### Production Configuration

```yaml
config_version: 1
server:
  api_key: "a1b2c3d4-e5f6-4789-a0b1-c2d3e4f5a6b7"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
  cert_file: "/etc/ssl/certs/server.crt"
  key_file: "/etc/ssl/private/server.key"
  read_timeout: 10
  write_timeout: 10
  allow_custom_queries: false
database:
  host: "pgsql.internal.example.com"
  port: 5432
  name: "prtg_data_exporter_collection"
  user: "mcp_prtg_reader"
  password: "strong_random_password_here"
  sslmode: "require"
prtg:
  enabled: true
  base_url: "https://prtg.internal.example.com:1616"
  api_token: "your-prtg-scripting-api-key"
  timeout: 30
  verify_ssl: true
logging:
  level: "info"
  file: "/var/log/mcp-server-prtg/server.log"
  max_size_mb: 50
  max_backups: 10
  max_age_days: 90
  compress: true
```

## See Also

- [Installation Guide](INSTALLATION.md)
- [Usage Guide](USAGE.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
