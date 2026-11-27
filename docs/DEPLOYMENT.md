# Deployment and Updates

This guide covers deploying, configuring, updating and rolling back MCP Server PRTG on Windows (MSI), Linux (systemd) and in stdio mode. It is aimed at operators who install and maintain the server.

The server exposes PRTG monitoring data over Streamable HTTP at `https://<host>:8443/mcp`. Data sources are the PostgreSQL **target** database of PRTG Data Exporter 2026.x (default name `prtg_data_exporter_collection`) and, optionally, PRTG API v2.

## Table of contents

- [Choose a deployment mode](#choose-a-deployment-mode)
- [Prerequisites](#prerequisites)
- [Read-only PostgreSQL role](#read-only-postgresql-role)
- [Enabling the PRTG API v2](#enabling-the-prtg-api-v2)
- [Secrets and secret.key](#secrets-and-secretkey)
- [Windows (MSI)](#windows-msi)
- [Linux (systemd)](#linux-systemd)
- [stdio mode (npm)](#stdio-mode-npm)
- [Connect an MCP client](#connect-an-mcp-client)
- [Upgrade checklist](#upgrade-checklist)
- [See also](#see-also)

## Choose a deployment mode

| Mode | When to use | Data sources | Update method |
|---|---|---|---|
| Windows service (MSI) | Shared server, typically next to PRTG (recommended) | PostgreSQL + optional PRTG API v2 | Run a newer MSI |
| Linux service (systemd) | Shared server on Linux | PostgreSQL + optional PRTG API v2 | Replace the binary |
| stdio mode (npm) | Single user's MCP client | PRTG API v2 only (no database) | `npx` pulls the latest; pin a version |

## Prerequisites

For the HTTP service modes (Windows and Linux):

- Network access to the PRTG Data Exporter PostgreSQL **target** database (`prtg_data_exporter_collection`), with the dedicated read-only role below.
- Optionally, a PRTG API v2 token (PRTG base URL with API port, e.g. `https://prtg.example.com:1616`).
- Inbound TCP 8443 open from the MCP clients.

For stdio mode, no database is required; see [stdio mode (npm)](#stdio-mode-npm).

## Read-only PostgreSQL role

Create a dedicated read-only role on the **target** database (not the Data Exporter configuration database `prtg_data_exporter`, which also stores PRTG credentials):

```sql
CREATE ROLE mcp_prtg_reader LOGIN;
\password mcp_prtg_reader
GRANT CONNECT ON DATABASE prtg_data_exporter_collection TO mcp_prtg_reader;
-- then, inside prtg_data_exporter_collection:
GRANT USAGE ON SCHEMA public TO mcp_prtg_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO mcp_prtg_reader;
ALTER DEFAULT PRIVILEGES FOR ROLE prtg_data_exporter_user_td IN SCHEMA public GRANT SELECT ON TABLES TO mcp_prtg_reader;
```

## Enabling the PRTG API v2

The PRTG API v2 must be activated on the PRTG core server (PRTG Application Server, ports 1616/1615 and 23580). It is enabled by default on new installs since 25.2.106 and automatically on update to 26.4.126 when enough memory is free; otherwise use Setup → Activate, then restart the core server.

In the server configuration, the optional `prtg:` section enables it: `enabled`, `base_url` (scheme and port, e.g. `https://prtg.example.com:1616`), `api_token` (a PRTG API key of type Scripting with read access), `timeout` (seconds), `verify_ssl`.

## Secrets and secret.key

`server.api_key`, `database.password` and `prtg.api_token` are stored encrypted in config.yaml as `enc:dpapi:...` (Windows, DPAPI machine scope: only that machine can decrypt) or `enc:aes:...` (Linux, AES-256-GCM with the key in `secret.key` next to config.yaml, mode 0600). You may type a clear-text value in config.yaml: it is encrypted in place at the next service start. A config file copied to another machine cannot be decrypted there: re-enter the secrets.

On Linux, back up `/etc/mcp-server-prtg/secret.key` together with config.yaml. Losing it means re-entering the secrets.

## Windows (MSI)

### Download and verify

1. Go to https://github.com/senhub-io/mcp-server-prtg/releases and download `mcp-server-prtg-<version>-amd64.msi` and `mcp-server-prtg-<version>-amd64.msi.sha256`.
2. Verify the checksum:

   ```powershell
   (Get-FileHash .\mcp-server-prtg-<version>-amd64.msi -Algorithm SHA256).Hash
   ```

   Compare the result with the value in the `.sha256` file.
3. Production releases (2.0.0 and later) are code-signed. Verify the signature:

   ```powershell
   Get-AuthenticodeSignature .\mcp-server-prtg-<version>-amd64.msi
   Get-AuthenticodeSignature .\mcp-server-prtg.exe
   ```

   The status must be `Valid`. Beta pre-releases are not signed.

### Install (wizard)

Run the MSI as an administrator. The installer shows a welcome page, the MIT licence, the install folder (default `C:\Program Files\Sensor Factory\MCP Server PRTG\`), then on a **fresh install** two settings pages:

- **Data Exporter database**: host (default `localhost`), port (5432), database name (`prtg_data_exporter_collection`), user (`mcp_prtg_reader`), password, SSL mode (`disable`/`require`/`verify-ca`/`verify-full`), MCP HTTPS port (8443; ports validated 1–65535).
- **PRTG API v2 (optional)**: enable, URL (must start with `http://` or `https://`), token (required when enabled), verify certificate.

A checkbox (on by default) "Show the configuration for AI clients when setup completes" controls whether setup ends by opening an elevated console with the client configuration.

The MSI writes `config.yaml` through `config init` with secrets encrypted, installs `C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe`, registers the service `mcp-server-prtg` (LocalSystem, automatic start, restart on failure) and starts it. Data lives in `C:\ProgramData\Sensor Factory\MCP Server PRTG\` (config.yaml, certs, logs; ACL SYSTEM + Administrators only; kept on uninstall). The Start Menu folder **MCP Server PRTG** contains two shortcuts: **MCP Server PRTG - Administration** (opens `https://localhost:<port>/admin/`) and **MCP Server PRTG - AI client configuration** (runs `config client --window`).

**Upgrades and repairs skip the settings pages and keep config.yaml.**

Limitation: a password containing a double quote cannot go through the wizard; set it afterwards with `config init`.

### Silent install

For GPO, Intune or SCCM, with properties:

```
msiexec /i mcp-server-prtg-<version>-amd64.msi /qn DBPASSWORD=... PRTGENABLE=1 PRTGURL=https://prtg:1616 PRTGTOKEN=... /l*v install.log
```

Properties: `DBHOST`, `DBPORT`, `DBNAME`, `DBUSER`, `DBPASSWORD`, `DBSSLMODE`, `MCPPORT`, `PRTGENABLE=1`, `PRTGURL`, `PRTGTOKEN`, `PRTGVERIFYSSL=1`, `ADOPT=1`.

### After install

From an elevated prompt:

```powershell
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml"
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config client [--host <name clients use>]
```

Both commands find the installed configuration without `--config` (ProgramData on Windows).

To change settings later, run `config init` with only the settings to change (it keeps the API key), then:

```powershell
Restart-Service mcp-server-prtg
```

### Certificates

The self-signed certificate (`certs\server.crt` next to config.yaml) covers `localhost`, the machine name and its IP addresses; it is regenerated automatically at start when missing (delete the `certs` folder and restart to regenerate, e.g. after renaming the server).

Clients must trust this certificate instead of disabling verification. Copy `C:\ProgramData\Sensor Factory\MCP Server PRTG\certs\server.crt` to each client computer:

- Claude Desktop / mcp-remote: set `NODE_EXTRA_CA_CERTS` to the path of the copied certificate.
- Claude Code, Cursor, VS Code: import the certificate into the computer's trusted root certificates (Windows: `certlm.msc` → Trusted Root Certification Authorities, or `Import-Certificate -FilePath server.crt -CertStoreLocation Cert:\LocalMachine\Root`; macOS: Keychain Access, "Always Trust").

Do not use `NODE_TLS_REJECT_UNAUTHORIZED=0`; it exposes the API key. A certificate from the company PKI can be used instead (`server.cert_file` / `server.key_file`).

### Administration page

The service serves an administration page at `https://localhost:<port>/admin/` (8443 by default), reachable **from the server itself only** (loopback; requests carrying proxy forwarding headers are refused). Sign in with the MCP API key. Sessions are kept in memory (8 h, 30 min idle), so a restart asks to sign in again.

Tabs:

- **Overview**: service version/uptime/tools, database status and detected schema features, PRTG API configured or not (the page never calls PRTG by itself), channel history coverage, Data Exporter collection profiles.
- **Settings**: database, PRTG API v2, MCP port, log level. **Test connection** buttons for the database and PRTG test the values typed before saving. Secrets are never shown: leave a secret field empty to keep it; a stored secret is only reused for the same database host/port/user without weakening the SSL mode, or the same PRTG URL without turning off certificate verification — otherwise it must be typed again. **Save** and **Save and restart** (the service exits and the service manager starts it again).
- **AI clients**: ready-to-paste configuration with a Copy button.

The page uses a strict Content-Security-Policy, an anti-CSRF header and rate-limited login failures.

### Verify

```powershell
Invoke-RestMethod https://localhost:8443/health -SkipCertificateCheck
```

`/health` returns `{"status":"ok"}` without authentication. Check version, uptime and database state:

```bash
curl -k -H "Authorization: Bearer <api_key>" https://localhost:8443/status
```

### Update

Run a newer MSI, interactively or silently. The upgrade replaces the program, restarts the service and never touches the data folder (config.yaml is kept). The same version can be reinstalled (repair). Read the release notes for breaking changes before upgrading, and back up `config.yaml` and the certs first.

### Roll back

Installing an older version is refused with "A newer version ... is already installed". To roll back, uninstall the current version, then install the older MSI. The data folder is kept.

### Uninstall

Via Settings > Apps, or silently:

```
msiexec /x <msi> /qn
```

Uninstalling removes the program and the service but keeps the data folder. Delete `C:\ProgramData\Sensor Factory\MCP Server PRTG\` manually to remove the configuration and secrets.

### Migrate from a manual install

If the service was created earlier with the built-in `install` command, migrate as follows:

1. Stop the old service:

   ```
   mcp-server-prtg.exe stop
   ```

   or

   ```
   sc stop mcp-server-prtg
   ```

2. Copy the old `config.yaml` (and its certs folder) into `C:\ProgramData\Sensor Factory\MCP Server PRTG\`.
3. Edit `cert_file` / `key_file` in `config.yaml` to point to the new certs location (YAML on Windows needs doubled backslashes inside double quotes).
4. Install with adoption:

   ```
   msiexec /i <msi> ADOPT=1
   ```

   This removes the old service registration and installs the managed one.
5. Delete the old program folder.

## Linux (systemd)

Release assets: `mcp-server-prtg-<version>-linux-amd64.tar.gz`, `mcp-server-prtg-<version>-linux-arm64.tar.gz` (archives contain the executable, README.md, LICENSE, config.example.yaml).

### Install

1. Download and extract the tar.gz for your architecture.
2. Install the binary and create the config directory:

   ```bash
   sudo install -m 0755 mcp-server-prtg /usr/local/bin/
   sudo mkdir -p /etc/mcp-server-prtg
   ```

3. Create the configuration (secrets may also come from environment variables such as `PRTG_DB_PASSWORD`, `PRTG_URL`, `PRTG_API_TOKEN`):

   ```bash
   sudo PRTG_DB_PASSWORD=... mcp-server-prtg config init --config /etc/mcp-server-prtg/config.yaml --https --db-host <host> --db-name prtg_data_exporter_collection --db-user mcp_prtg_reader
   ```

4. Install and start the systemd service:

   ```bash
   sudo mcp-server-prtg install --config /etc/mcp-server-prtg/config.yaml
   sudo mcp-server-prtg start
   ```

   Manage it with `sudo mcp-server-prtg stop|status` or `sudo systemctl restart mcp-server-prtg`.

Back up `/etc/mcp-server-prtg/secret.key` with config.yaml.

Verify with `/health` and `/status` as described under [Windows (MSI) > Verify](#verify).

### Update

1. Read the release notes for breaking changes.
2. Back up `/etc/mcp-server-prtg/config.yaml` and `secret.key`.
3. Stop the service and keep a copy of the old binary:

   ```bash
   sudo mcp-server-prtg stop
   sudo cp /usr/local/bin/mcp-server-prtg /usr/local/bin/mcp-server-prtg.previous
   ```

4. Extract the new archive and replace the binary, then start:

   ```bash
   sudo install -m 0755 mcp-server-prtg /usr/local/bin/
   sudo mcp-server-prtg start
   ```

5. Check the version at `/status`. config.yaml, secret.key and certs are kept.

### Roll back

```bash
sudo mcp-server-prtg stop
sudo cp /usr/local/bin/mcp-server-prtg.previous /usr/local/bin/mcp-server-prtg
sudo mcp-server-prtg start
```

### Uninstall

```bash
sudo mcp-server-prtg uninstall --config /etc/mcp-server-prtg/config.yaml
```

Back up config.yaml and secret.key first.

## stdio mode (npm)

Run locally for a single user's MCP client. Uses PRTG API v2 only; no database.

```
npx -y @senhub-io/mcp-server-prtg
```

`npx` without a version always fetches the latest release. Pin a version for reproducible behaviour:

```
npx -y @senhub-io/mcp-server-prtg@<version>
```

Environment variables:

| Variable | Meaning | Default |
|---|---|---|
| `PRTG_URL` | PRTG base URL with the API port, e.g. `https://prtg.example.com:1616` | required |
| `PRTG_API_TOKEN` | PRTG API v2 token | required |
| `PRTG_VERIFY_SSL` | Verify the PRTG TLS certificate | `true` |
| `PRTG_TIMEOUT` | Request timeout in seconds | `30` |
| `PRTG_MAX_CONCURRENT` | Maximum PRTG API requests in flight, to protect the PRTG core server | `2` |
| `PRTG_CACHE_DISABLED` | Disable the response cache | `false` |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` |

## Connect an MCP client

The endpoint is `https://<host>:8443/mcp`, authenticated with the header `Authorization: Bearer <api_key>`.

Run `mcp-server-prtg config client [--host <name>] [--config <path>]` to print the configurations for:

- **Claude Desktop**: `npx -y mcp-remote <url> --header "Authorization:Bearer ${MCP_PRTG_API_KEY}"` with env `MCP_PRTG_API_KEY` and `NODE_EXTRA_CA_CERTS` set to the path of a copy of the server certificate.
- **Claude Code**: `claude mcp add --transport http prtg <url> --header "Authorization: Bearer <key>"`.
- **Cursor**: `~/.cursor/mcp.json` with `url` and `headers`.
- **VS Code**: `.vscode/mcp.json` with `"type": "http"`, `url`, `headers`.

Example for Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "prtg": {
      "command": "npx",
      "args": [
        "mcp-remote",
        "https://<MCP_SERVER_HOST>:8443/mcp",
        "--header",
        "Authorization:Bearer ${MCP_PRTG_API_KEY}"
      ],
      "env": {
        "MCP_PRTG_API_KEY": "<api_key from config show-api-key>",
        "NODE_EXTRA_CA_CERTS": "<path to a copy of server.crt>"
      }
    }
  }
}
```

Trust the server certificate on each client as described under [Windows (MSI) > Certificates](#certificates). Client-specific guides: [Claude Desktop](CLAUDE_DESKTOP_CONFIG.md), [Cursor](CURSOR_CONFIG.md), [Continue](CONTINUE_CONFIG.md), [Cline](CLINE_CONFIG.md).

## Upgrade checklist

Applies to all modes.

1. Read the release notes for breaking changes before upgrading.
2. Back up `config.yaml` (and the certs; on Linux also `secret.key`).
3. Upgrade.
4. Check `/health` and `/status` (version and database state).
5. Run one tool from an MCP client.
6. Keep the previous installer or binary for rollback.

## See also

- [Configuration reference](CONFIGURATION.md): every setting
- [Troubleshooting](TROUBLESHOOTING.md): diagnosing problems
- [Tools](TOOLS.md): what each MCP tool returns
- [RELEASING.md](RELEASING.md): release procedure for maintainers
