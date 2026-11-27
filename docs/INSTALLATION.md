# Installation Guide

End-user installation guide for MCP Server PRTG on Windows and Linux. For operational details (silent install, updates, rollback, service management), see [DEPLOYMENT.md](DEPLOYMENT.md).

## Prerequisites

- **PRTG Data Exporter 2026.x** — provides the PostgreSQL target database (default name `prtg_data_exporter_collection`) that the server reads in HTTP mode.
- **PRTG Core Server** — required for PRTG API v2 access (optional, for real-time metrics).
- **MCP Server PRTG release** — download from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases). Supported service platforms: Windows amd64 (MSI or zip) and Linux amd64/arm64 (tar.gz).
- **Node.js** and **npm** — for mcp-remote when connecting MCP clients such as Claude Desktop.

## Windows

### MSI installer (recommended)

Download `mcp-server-prtg-<version>-amd64.msi` from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases) and run it as administrator.

On a fresh install the wizard asks for:

- **Data Exporter database**: host (default `localhost`), port (5432), database name (`prtg_data_exporter_collection`), user (`mcp_prtg_reader`), password, SSL mode (`disable`/`require`/`verify-ca`/`verify-full`), MCP HTTPS port (8443).
- **PRTG API v2 (optional)**: enable, URL (must start with `http://` or `https://`), token, verify certificate.

The installer writes `C:\ProgramData\Sensor Factory\MCP Server PRTG\config.yaml` with secrets encrypted, installs `C:\Program Files\Sensor Factory\MCP Server PRTG\mcp-server-prtg.exe`, registers the Windows service `mcp-server-prtg` and starts it. Upgrades and repairs skip the wizard and keep the existing configuration.

After installation, open `https://localhost:8443/admin/` from the server itself and sign in with the API key (shown by `config show-api-key`), or use the Start Menu shortcuts **MCP Server PRTG - Administration** and **MCP Server PRTG - AI client configuration**.

See [DEPLOYMENT.md](DEPLOYMENT.md#windows-msi) for silent installation, updates, rollback and migration.

### Manual installation

Manual installation with `mcp-server-prtg.exe install` remains possible on hosts where the MSI cannot be used.

1. Download `mcp-server-prtg-<version>-windows-amd64.zip` from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases) and extract `mcp-server-prtg.exe`.
2. Copy the executable to a program folder, for example `C:\Program Files\Sensor Factory\MCP Server PRTG\`.
3. Run `mcp-server-prtg.exe install` to register the service and generate a configuration.
4. Edit `config.yaml` (or use `config init`) to set the database host, name `prtg_data_exporter_collection`, user `mcp_prtg_reader` and password (or the `PRTG_DB_PASSWORD` environment variable).
5. Run `mcp-server-prtg.exe start` and check `mcp-server-prtg.exe status`.

## Linux

1. Download `mcp-server-prtg-<version>-linux-amd64.tar.gz` (or `-linux-arm64.tar.gz`) from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases) and extract.
2. Install the binary and create the configuration directory:

   ```bash
   sudo install -m 0755 mcp-server-prtg /usr/local/bin/
   sudo mkdir -p /etc/mcp-server-prtg
   ```

3. Create the configuration (secrets may come from environment variables such as `PRTG_DB_PASSWORD`):

   ```bash
   sudo PRTG_DB_PASSWORD=... mcp-server-prtg config init --config /etc/mcp-server-prtg/config.yaml --https --db-host <host> --db-name prtg_data_exporter_collection --db-user mcp_prtg_reader
   ```

4. Install and start the systemd service:

   ```bash
   sudo mcp-server-prtg install --config /etc/mcp-server-prtg/config.yaml
   sudo mcp-server-prtg start
   ```

Back up `/etc/mcp-server-prtg/secret.key` together with `config.yaml`.

See [DEPLOYMENT.md](DEPLOYMENT.md#linux-systemd) for updates, rollback and uninstall.

## Connect an MCP client

Run `mcp-server-prtg config client` (elevated on Windows) to print ready-to-paste configurations for Claude Desktop, Claude Code, Cursor and VS Code, or see [DEPLOYMENT.md](DEPLOYMENT.md#connect-an-mcp-client).

## Next steps

- [DEPLOYMENT.md](DEPLOYMENT.md) — operations: silent install, certificates, updates, rollback
- [CONFIGURATION.md](CONFIGURATION.md) — all configuration options
- [TOOLS.md](TOOLS.md) — tool reference
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — troubleshooting
