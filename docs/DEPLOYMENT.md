# Deployment and Updates

This guide covers deploying, configuring, updating and rolling back MCP Server PRTG on Windows, Linux and in stdio mode. It is aimed at operators who install and maintain the server.

The server exposes PRTG monitoring data over Streamable HTTP at `https://<host>:8443/mcp`. Data sources are a PostgreSQL database filled by the Paessler PRTG Data Exporter (primary) and, optionally, PRTG API v2.

## Table of contents

- [Choose a deployment mode](#choose-a-deployment-mode)
- [Prerequisites](#prerequisites)
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

- Network access to the PRTG Data Exporter PostgreSQL target database, with a dedicated read-only role that has `SELECT` on the `prtg_*` tables only.
- Optionally, a PRTG API v2 token (PRTG base URL with API port, e.g. `https://prtg.example.com:1616`).
- Inbound TCP 8443 open from the MCP clients.

For stdio mode, no database is required; see [stdio mode (npm)](#stdio-mode-npm).

## Windows (MSI)

### Download and verify

1. Go to https://github.com/senhub-io/mcp-server-prtg/releases and download `mcp-server-prtg-<version>-amd64.msi` and `mcp-server-prtg-<version>-amd64.msi.sha256`.
2. Verify the checksum:

   ```powershell
   (Get-FileHash .\mcp-server-prtg-<version>-amd64.msi -Algorithm SHA256).Hash
   ```

   Compare the result with the value in the `.sha256` file.

The MSI may be unsigned in some releases. If Windows SmartScreen warns, complete the verification above before choosing "More info > Run anyway".

### Install (interactive, silent)

Interactive: run the MSI as an administrator. It installs the program, registers and starts the Windows service `mcp-server-prtg` (display name "MCP Server PRTG"), automatic start, running as LocalSystem. The service restarts automatically after a crash (twice, 10 s delay).

Silent install, for use with GPO, Intune or SCCM:

```
msiexec /i mcp-server-prtg-<version>-amd64.msi /qn /l*v install.log
```

On first start the service generates `config.yaml` with a random API key (UUID) and a self-signed TLS certificate. The database being unreachable does not stop the service; tools report errors until the database is configured.

### What gets installed

| Path | Contents |
|---|---|
| `C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe` | Program binary |
| `C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml` | Configuration |
| `C:\ProgramData\Sensor Factory\MCP Server PRTG\certs\server.crt` | TLS certificate |
| `C:\ProgramData\Sensor Factory\MCP Server PRTG\certs\server.key` | TLS private key |
| `C:\ProgramData\Sensor Factory\MCP Server PRTG\logs\mcp-server-prtg.log` | Log file |

Access to the data folder is restricted to SYSTEM and Administrators because `config.yaml` holds the API key and the database password.

### Configure

1. Edit `C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml`. Example:

   ```yaml
   config_version: 1
   server:
     api_key: <random UUID generated on first start>
     bind_address: "0.0.0.0"
     port: 8443
     enable_tls: true
     cert_file: "C:\\ProgramData\\Sensor Factory\\MCP Server PRTG\\certs\\server.crt"
     key_file: "C:\\ProgramData\\Sensor Factory\\MCP Server PRTG\\certs\\server.key"
     allow_custom_queries: false
   database:
     host: <db host>
     port: 5432
     name: "prtg_data_exporter"
     user: "prtg_reader"
     password: <password>
     sslmode: "require"
   prtg:
     enabled: false
     base_url: "https://prtg.example.com:1616"
     api_token: <token>
     timeout: 30
     verify_ssl: true
   logging:
     level: info
     file: <generated path, keep it>
     max_size_mb: 10
     max_backups: 5
     max_age_days: 30
     compress: true
   ```

   - `sslmode: "require"` is recommended in production.
   - To enable PRTG API v2, set `prtg.enabled: true` and fill in `base_url` and `api_token`.
   - `logging.level` accepts `debug`, `info`, `warn`, `error`.
   - The API key must be at least 16 characters (the generated one is 36); the server refuses to start otherwise.

2. Alternatively, provide the database password through the `PRTG_DB_PASSWORD` environment variable. For a Windows service, set it as a system-wide variable.

3. Restart the service after editing `config.yaml`:

   ```powershell
   Restart-Service mcp-server-prtg
   ```

4. Retrieve the API key from `config.yaml` for the MCP clients. Clients authenticate only with the header `Authorization: Bearer <api_key>`; there is no query-string token.

5. Open the firewall port:

   ```powershell
   New-NetFirewallRule -DisplayName "MCP Server PRTG" -Direction Inbound -Protocol TCP -LocalPort 8443 -Action Allow
   ```

Service management:

```powershell
Get-Service mcp-server-prtg
Restart-Service mcp-server-prtg
```

Logs are in the data folder under `logs\`.

### Verify

```powershell
Invoke-RestMethod https://localhost:8443/health -SkipCertificateCheck
```

or

```bash
curl -k https://localhost:8443/health
```

`/health` returns `{"status":"ok"}` without authentication.

Check version, uptime and database state:

```bash
curl -k -H "Authorization: Bearer <api_key>" https://localhost:8443/status
```

The database field is `"connected"`, `"error"` or `"not_configured"`.

### Update

Run a newer MSI, interactively or silently. The upgrade replaces the program, restarts the service and never touches the data folder. The same version can be reinstalled (repair).

Read the release notes for breaking changes before upgrading, and back up `config.yaml` and the certs first.

### Roll back

Installing an older version is refused with "A newer version ... is already installed". To roll back, uninstall the current version, then install the older MSI. The data folder is kept.

### Deploy to many servers (GPO / Intune / SCCM)

Use the silent install command:

```
msiexec /i mcp-server-prtg-<version>-amd64.msi /qn /l*v install.log
```

### Migrate from a manual install

If the service was created earlier with the built-in `install` command, the MSI refuses to install and shows a message. Migrate as follows:

1. Stop the old service:

   ```
   mcp-server-prtg.exe stop
   ```

   or

   ```
   sc stop mcp-server-prtg
   ```

2. Copy the old `config.yaml` (and its certs folder) into `C:\ProgramData\Sensor Factory\MCP Server PRTG\`.
3. Edit `cert_file` / `key_file` in `config.yaml` to point to the new certs location. YAML on Windows needs doubled backslashes inside double quotes, e.g.:

   ```yaml
   cert_file: "C:\\ProgramData\\Sensor Factory\\MCP Server PRTG\\certs\\server.crt"
   ```

4. Install with adoption:

   ```
   msiexec /i <msi> ADOPT=1
   ```

   This removes the old service registration and installs the managed one.
5. Delete the old program folder.

### Uninstall

Via Settings > Apps, or silently:

```
msiexec /x <msi> /qn
```

Uninstalling removes the program and the service but keeps the data folder. Delete `C:\ProgramData\Sensor Factory\MCP Server PRTG\` manually to remove the configuration and secrets.

## Linux (systemd)

Release assets per platform: `mcp-server-prtg_linux_amd64.zip`, `mcp-server-prtg_linux_arm64.zip`, `mcp-server-prtg_windows_amd64.zip`, `mcp-server-prtg_darwin_amd64.zip`, `mcp-server-prtg_darwin_arm64.zip`.

### Install

1. Download and unzip the release zip for your architecture.
2. Place the binary at `/opt/mcp-server-prtg/mcp-server-prtg` and make it executable:

   ```bash
   chmod +x /opt/mcp-server-prtg/mcp-server-prtg
   ```

3. Install the service:

   ```bash
   cd /opt/mcp-server-prtg
   sudo ./mcp-server-prtg install
   ```

   This generates `/opt/mcp-server-prtg/config.yaml` (with an API key), a self-signed certificate in `/opt/mcp-server-prtg/certs` and a systemd unit. Edit `config.yaml` as described in [Windows (MSI) > Configure](#configure): the same keys apply, and `cert_file` / `key_file` already point at the generated certificate.

4. Manage the service:

   ```bash
   sudo ./mcp-server-prtg start
   sudo ./mcp-server-prtg stop
   sudo ./mcp-server-prtg status
   ```

   or

   ```bash
   sudo systemctl restart mcp-server-prtg
   ```

Verify with `/health` and `/status` as described under [Windows (MSI) > Verify](#verify).

### Update

1. Read the release notes for breaking changes.
2. Back up `config.yaml` and the certs.
3. Download the new zip and verify it.
4. Stop the service and keep a copy of the old binary:

   ```bash
   cd /opt/mcp-server-prtg
   sudo ./mcp-server-prtg stop
   sudo cp mcp-server-prtg mcp-server-prtg.previous
   ```

5. Replace the binary, then start:

   ```bash
   unzip -o mcp-server-prtg_linux_amd64.zip
   sudo install -m 0755 mcp-server-prtg_linux_amd64 /opt/mcp-server-prtg/mcp-server-prtg
   sudo ./mcp-server-prtg start
   ```

6. Check the version at `/status`. `config.yaml` and certs are kept.

### Roll back

```bash
cd /opt/mcp-server-prtg
sudo ./mcp-server-prtg stop
sudo cp mcp-server-prtg.previous mcp-server-prtg
sudo ./mcp-server-prtg start
```

### Uninstall

```bash
sudo ./mcp-server-prtg uninstall
```

This removes the service, `config.yaml`, the certs and the default logs folder. Back up `config.yaml` first.

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
        "MCP_SERVER_API_KEY": "<api_key from config.yaml>"
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
2. Back up `config.yaml` (and the certs).
3. Upgrade.
4. Check `/health` and `/status` (version and database state).
5. Run one tool from an MCP client.
6. Keep the previous installer or binary for rollback.

## See also

- [Configuration reference](CONFIGURATION.md): every setting
- [Troubleshooting](TROUBLESHOOTING.md): diagnosing problems
- [Tools](TOOLS.md): what each MCP tool returns