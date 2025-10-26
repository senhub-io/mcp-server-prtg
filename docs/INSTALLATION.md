# Installation Guide

Complete installation guide for MCP Server PRTG on different platforms.

## Prerequisites

- **PostgreSQL** - PRTG Data Exporter database
- **MCP Server PRTG Binary** - Downloaded from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases)
- **Python 3.8+** and **pip** - For mcp-proxy (MCP Client client)

## Windows Installation

### 1. Download Binary

Download `mcp-server-prtg_windows_amd64.exe` from [GitHub releases](https://github.com/senhub-io/mcp-server-prtg/releases).

### 2. Create Installation Directory

```powershell
# Create application directory
mkdir "C:\Program Files\mcp-server-prtg"
cd "C:\Program Files\mcp-server-prtg"

# Copy binary
copy "C:\Users\<user>\Downloads\mcp-server-prtg_windows_amd64.exe" .\mcp-server-prtg.exe
```

### 3. Install as Windows Service

```powershell
# Install service (automatically generates config.yaml)
.\mcp-server-prtg.exe install

# Service is now installed but not started
```

### 4. Configure Service

Edit the generated `config.yaml` file:

```yaml
version: 1
server:
  api_key: "<your-generated-api-key>"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: ""  # Can be provided via PRTG_DB_PASSWORD env var
  sslmode: "disable"
logging:
  level: "info"
```

**Important:** Note the generated API key - you'll need it for MCP Client configuration.

### 5. Configure Database Password

```powershell
# Option 1: System environment variable
setx PRTG_DB_PASSWORD "your-password" /M

# Option 2: Session variable
$env:PRTG_DB_PASSWORD = "your-password"

# Option 3: Directly in config.yaml (less secure)
# Edit config.yaml and set database.password
```

### 6. Start Service

```powershell
# Start service
.\mcp-server-prtg.exe start

# Check status
.\mcp-server-prtg.exe status
```

### 7. Verify Logs

```powershell
# Logs are in
type .\logs\mcp-server-prtg.log

# For troubleshooting, enable debug in config.yaml
# logging.level: debug
```

### Windows Service Management

```powershell
# Start
.\mcp-server-prtg.exe start

# Stop
.\mcp-server-prtg.exe stop

# Detailed status
.\mcp-server-prtg.exe status

# Uninstall (also removes config + logs + certs)
.\mcp-server-prtg.exe uninstall
```

## Linux Installation

### 1. Download Binary

```bash
# Download for your architecture
wget https://github.com/senhub-io/mcp-server-prtg/releases/latest/download/mcp-server-prtg_linux_amd64

# Make executable
chmod +x mcp-server-prtg_linux_amd64
mv mcp-server-prtg_linux_amd64 mcp-server-prtg
```

### 2. Create Installation Directory

```bash
# Install in /opt
sudo mkdir -p /opt/mcp-server-prtg
sudo mv mcp-server-prtg /opt/mcp-server-prtg/
cd /opt/mcp-server-prtg
```

### 3. Install as systemd Service

```bash
# Install (generates config.yaml + certificates + systemd unit)
sudo ./mcp-server-prtg install

# Service is installed and enabled at boot
```

### 4. Configure Service

```bash
# Edit configuration
sudo nano /opt/mcp-server-prtg/config.yaml
```

Minimal configuration:

```yaml
version: 1
server:
  api_key: "<api-key-generated-during-install>"
  bind_address: "0.0.0.0"
  port: 8443
  enable_tls: true
database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  sslmode: "require"  # Recommended for production
logging:
  level: "info"
```

### 5. Configure Database Password

```bash
# Create environment file for service
sudo mkdir -p /etc/systemd/system/mcp-server-prtg.service.d
sudo nano /etc/systemd/system/mcp-server-prtg.service.d/environment.conf
```

Content:

```ini
[Service]
Environment="PRTG_DB_PASSWORD=your-password"
```

Reload systemd:

```bash
sudo systemctl daemon-reload
```

### 6. Start Service

```bash
# Start
sudo ./mcp-server-prtg start

# Check status
sudo ./mcp-server-prtg status

# Or via systemctl
sudo systemctl status mcp-server-prtg
```

### 7. Verify Logs

```bash
# Service logs
sudo journalctl -u mcp-server-prtg -f

# Application logs
sudo tail -f /opt/mcp-server-prtg/logs/mcp-server-prtg.log
```

### Linux Service Management

```bash
# Via binary
sudo ./mcp-server-prtg start
sudo ./mcp-server-prtg stop
sudo ./mcp-server-prtg status

# Via systemctl
sudo systemctl start mcp-server-prtg
sudo systemctl stop mcp-server-prtg
sudo systemctl status mcp-server-prtg
sudo systemctl restart mcp-server-prtg

# Uninstall
sudo ./mcp-server-prtg uninstall
```

## macOS Installation

### 1. Download Binary

```bash
# For Apple Silicon (M1/M2/M3)
curl -LO https://github.com/senhub-io/mcp-server-prtg/releases/latest/download/mcp-server-prtg_darwin_arm64

# For Intel
curl -LO https://github.com/senhub-io/mcp-server-prtg/releases/latest/download/mcp-server-prtg_darwin_amd64

# Make executable
chmod +x mcp-server-prtg_darwin_arm64
mv mcp-server-prtg_darwin_arm64 mcp-server-prtg
```

### 2. Create Installation Directory

```bash
# Install in /usr/local
sudo mkdir -p /usr/local/mcp-server-prtg
sudo mv mcp-server-prtg /usr/local/mcp-server-prtg/
cd /usr/local/mcp-server-prtg
```

### 3. Install as launchd Service

```bash
# Install (generates config.yaml + certificates + launchd plist)
sudo ./mcp-server-prtg install
```

### 4. Configuration

Same process as Linux - edit `config.yaml`:

```bash
sudo nano /usr/local/mcp-server-prtg/config.yaml
```

### 5. Environment Variable for Password

```bash
# Edit plist file
sudo nano /Library/LaunchDaemons/io.senhub.mcp-server-prtg.plist
```

Add in `<dict>` section:

```xml
<key>EnvironmentVariables</key>
<dict>
    <key>PRTG_DB_PASSWORD</key>
    <string>your-password</string>
</dict>
```

### 6. Start Service

```bash
# Via binary
sudo ./mcp-server-prtg start

# Via launchctl
sudo launchctl load /Library/LaunchDaemons/io.senhub.mcp-server-prtg.plist
```

## Console Mode (No Service)

For testing or debugging without installing as service:

```bash
# Windows
.\mcp-server-prtg.exe run

# Linux / macOS
./mcp-server-prtg run

# With options
./mcp-server-prtg run --config /path/to/config.yaml

# Stop with Ctrl+C
```

## Install mcp-proxy (Client)

For use with MCP Client, install mcp-proxy:

```bash
# Via pip
pip install mcp-proxy

# Or with pip3
pip3 install mcp-proxy

# Verify installation
mcp-proxy --help
```

## Installation Verification

### Test SSE Connection

```bash
# Test with curl (replace API_KEY and URL)
curl -k -N \
  -H "Authorization: Bearer your-api-key" \
  https://your-server:8443/sse

# You should receive SSE events
```

### Test with mcp-proxy

```bash
# Command line test
mcp-proxy https://your-server:8443/sse \
  --headers Authorization "Bearer your-api-key"

# Should display MCP connection
```

### Verify Database

The detailed status automatically tests the connection:

```bash
# Windows
.\mcp-server-prtg.exe status

# Linux / macOS
sudo ./mcp-server-prtg status
```

Expected output:

```
═══════════════════════════════════════════════════════════════
  MCP Server PRTG - Service Status
═══════════════════════════════════════════════════════════════

✅ Service Status:  Running
📋 Service Name:    mcp-server-prtg

Configuration:
  Config File:  /opt/mcp-server-prtg/config.yaml
  Log File:     /opt/mcp-server-prtg/logs/mcp-server-prtg.log
  Log Level:    info

Server:
  Address:      0.0.0.0:8443
  Public URL:   https://your-server:8443
  TLS Enabled:  true
  Certificate:  /opt/mcp-server-prtg/certs/server.crt

Database:
  Host:     localhost:5432
  Database: prtg_data_exporter
  User:     prtg_reader
  SSL Mode: require
  Status:   ✅ Connected

═══════════════════════════════════════════════════════════════
```

## Next Steps

1. **Configure MCP Client** - See [USAGE.md](USAGE.md)
2. **Customize configuration** - See [CONFIGURATION.md](CONFIGURATION.md)
3. **Test MCP tools** - See [TOOLS.md](TOOLS.md)

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for installation issues.
