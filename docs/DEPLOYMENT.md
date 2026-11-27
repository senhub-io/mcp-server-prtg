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

The MSI may be unsigned in some releases. If Windows SmartScreen warns, complete the verification above before choosing "More info > Run anyway".

### Install (wizard)

Run the MSI as an administrator. On a **fresh install** the wizard asks for:

- Database: host (default `localhost`), port (5432), name (`prtg_data_exporter_collection`), user (`mcp_prtg_reader`), password, SSL mode (`disable`/`require`/`verify-ca`/`verify-full`), MCP HTTPS port (8443).
- PRTG API v2 (checkbox): URL, token, verify certificate.

It writes `config.yaml` through `config init` with secrets encrypted, then starts the service. **Upgrades and repairs skip the wizard and keep config.yaml.**

The MSI installs `C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe` and the service `mcp-server-prtg` (LocalSystem, automatic start, restart on failure). Data lives in `C:\ProgramData\Sensor Factory\MCP Server PRTG\` (config.yaml, certs, logs; ACL SYSTEM + Administrators only; kept on uninstall).

Limitation: a password containing a double quote cannot go through the wizard; set it afterwards with `config init`.

### Silent install

For GPO, Intune or SCCM, with properties:

```
msiexec /i mcp-server-prtg-<version>-amd64.msi /qn DBPASSWORD=... PRTGENABLE=1 PRTGURL=https://prtg:1616 PRTGTOKEN=... /l*v install.log
```

Properties: `DBHOST`, `DBPORT`, `DBNAME`, `DBUSER`, `DBPASSWORD`, `DBSSLMODE`, `MCPPORT`, `PRTGENABLE=1`, `PRTGURL`, `PRTGTOKEN`, `PRTGVERIFYSSL=1`, `ADOPT=1`.

### After install

When the last wizard page's "Show the configuration for AI clients" box is checked (the default), setup ends by opening an elevated console with the ready-to-paste configuration for Claude Desktop, Claude Code, Cursor and VS Code: server URL, API key and certificate instructions. The Start Menu shortcut **MCP Server PRTG - AI client configuration** opens it again at any time. From an elevated prompt:

```powershell
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config client [--host <name clients use>]
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config show-api-key
```

Both commands find the installed configuration without `--config`. The self-signed certificate covers `localhost`, the machine name and its addresses; copy `C:\ProgramData\Sensor Factory\MCP Server PRTG\certs\server.crt` to each client computer and trust it (Claude Desktop: `NODE_EXTRA_CA_CERTS`; other clients: trusted root certificates). Certificates generated by 2.0.0-beta.1 only cover `localhost`: delete the `certs` folder and restart the service to regenerate them.

To change settings later, run `config init` with only the settings to change (it keeps the API key), then:

```powershell
Restart-Service mcp-server-prtg
```

### Administration page

The service serves an administration page at `https://localhost:<port>/admin/` (8443 by default), reachable **from the server itself only**. Sign in with the MCP API key (`config show-api-key`). It shows the service, database and PRTG status, the channel history coverage and the Data Exporter collection profiles; it edits the database, PRTG API v2, port and log level settings, with **Test connection** buttons for the database and PRTG that use the values typed in the form before anything is saved; and it shows the AI client configuration with a copy button. Passwords and tokens are never sent back to the browser: leave a secret field empty to keep the stored value. **Save and restart** writes `config.yaml` (secrets encrypted) and restarts the service through the service manager.

Example:

```powershell
& "C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe" config init --config "C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml" --db-host db.internal --prtg-url https://prtg:1616 --prtg-token <token>
Restart-Service mcp-server-prtg
```

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

Versions older than 2.0.0-beta.1 cannot read encrypted secrets. After rolling back to such a version, replace `server.api_key`, `database.password` and `prtg.api_token` in config.yaml with their clear-text values, then restart the service.

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
        "Authorization:Bearer ${MCP_SERVER_API_KEY}"
      ],
      "env": {
        "MCP_SERVER_API_KEY": "<api_key from config show-api-key>"
      }
    }
  }
}
```

With the generated self-signed certificate, Node clients need `NODE_TLS_REJECT_UNAUTHORIZED=0` in `env`. For production, replace `cert_file` / `key_file` in `config.yaml` with a certificate trusted by the clients.

Client-specific guides: [Claude Desktop](CLAUDE_DESKTOP_CONFIG.md), [Cursor](CURSOR_CONFIG.md), [Continue](CONTINUE_CONFIG.md), [Cline](CLINE_CONFIG.md).

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
