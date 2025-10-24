# PRTG MCP Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that exposes PRTG monitoring data from PostgreSQL to Large Language Models (LLMs). This server allows LLMs like Claude or Mistral to query PRTG sensor data, device status, and alerts in natural language.

## Overview

This server connects to a PostgreSQL database populated by [PRTG Data Exporter](https://www.paessler.com/manuals/prtg/data_exporter) and provides structured access to PRTG monitoring metadata and current sensor status through MCP tools.

### Architecture

```
[PRTG Core] → [Data Exporter] → [PostgreSQL]
                                      ↑
                                 [MCP Server Go]
                                      ↑
                               [LLM via MCP Client]
```

## Features

### Available Tools

The server exposes 6 MCP tools for querying PRTG data:

1. **prtg_get_sensors** - Retrieve sensors with optional filters (device, name, status, tags)
2. **prtg_get_sensor_status** - Get detailed current status of a specific sensor
3. **prtg_get_alerts** - Retrieve sensors in alert state (Warning/Down/Error)
4. **prtg_device_overview** - Complete device overview with all sensors and statistics
5. **prtg_top_sensors** - Top sensors ranked by uptime, downtime, or alert frequency
6. **prtg_query_sql** - Execute custom SQL queries (SELECT only, for advanced use)

### Important Note

This server exposes **metadata and current status** from PRTG, not historical time-series data. The PRTG Data Exporter focuses on exporting the monitoring structure (groups, devices, sensors) and their current state, not historical channel values.

## Prerequisites

- **Go** 1.21 or higher
- **PostgreSQL** database with PRTG Data Exporter configured
- **PRTG Network Monitor** with Data Exporter pushing to PostgreSQL
- Database user with read access to PRTG tables

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/matthieu/mcp-server-prtg.git
cd mcp-server-prtg

# Download dependencies
go mod download

# Build for current OS
make build

# Or build for all platforms
make build-all
```

### Pre-built Binaries

Check the [releases page](https://github.com/matthieu/mcp-server-prtg/releases) for pre-built binaries.

## Configuration

### Environment Variables

The server is configured primarily through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PRTG_DB_HOST` | PostgreSQL host | `localhost` |
| `PRTG_DB_PORT` | PostgreSQL port | `5432` |
| `PRTG_DB_NAME` | Database name | `prtg_data_exporter` |
| `PRTG_DB_USER` | Database user | `prtg_reader` |
| `PRTG_DB_PASSWORD` | Database password | *(required)* |
| `PRTG_DB_SSLMODE` | SSL mode | `disable` |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` |

### Configuration File (Optional)

You can also use a YAML configuration file:

```bash
# Copy example config
cp configs/config.example.yaml configs/config.yaml

# Edit with your settings
nano configs/config.yaml
```

**Note:** Environment variables take precedence over the config file.

## Usage

### Running the Server

```bash
# Using environment variables
export PRTG_DB_PASSWORD="your_password"
./build/mcp-server-prtg

# Or with config file
./build/mcp-server-prtg -config configs/config.yaml

# Or using make (development)
export PRTG_DB_PASSWORD="your_password"
make run
```

### Integration with Claude Desktop

Add this server to your Claude Desktop configuration:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "prtg": {
      "command": "/path/to/mcp-server-prtg",
      "env": {
        "PRTG_DB_HOST": "your-postgres-host",
        "PRTG_DB_PORT": "5432",
        "PRTG_DB_NAME": "prtg_data_exporter",
        "PRTG_DB_USER": "prtg_reader",
        "PRTG_DB_PASSWORD": "your_password",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

Restart Claude Desktop after updating the configuration.

### Example Queries

Once integrated, you can ask Claude natural language questions like:

- "Show me all sensors that are currently down"
- "What's the status of sensors on device 'web-server-01'?"
- "Give me an overview of the 'mail-server' device"
- "Which sensors have the most downtime in the last 24 hours?"
- "List all ping sensors with warning status"

## Development

### Project Structure

```
mcp-server-prtg/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── server/
│   │   └── server.go         # MCP server initialization
│   ├── database/
│   │   ├── db.go            # Database connection pool
│   │   └── queries.go       # SQL queries
│   ├── handlers/
│   │   └── tools.go         # MCP tool implementations
│   ├── types/
│   │   └── models.go        # Data models
│   └── config/
│       └── config.go        # Configuration management
├── configs/
│   └── config.example.yaml  # Example configuration
├── scripts/
│   └── build.sh            # Multi-platform build script
├── Makefile                # Build automation
├── go.mod
└── README.md
```

### Building

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Build for current OS
make build

# Build for all platforms
make build-all

# Run all checks
make verify
```

### Database Schema

The server queries these main tables:

- `prtg_sensor` - Sensor metadata and current status
- `prtg_device` - Device information
- `prtg_group` - Group/probe hierarchy
- `prtg_sensor_path` - Sensor full paths
- `prtg_device_path` - Device full paths
- `prtg_tag` - Tags for filtering
- `prtg_sensor_tag` / `prtg_device_tag` / `prtg_group_tag` - Tag associations

### Adding New Tools

To add a new MCP tool:

1. Add the tool definition in `internal/handlers/tools.go` (`RegisterTools` method)
2. Implement the handler function in the same file
3. Add any required database queries in `internal/database/queries.go`
4. Update the README with the new tool description

## Security

### SQL Injection Protection

- All queries use parameterized statements
- The `prtg_query_sql` tool strictly validates input:
  - Only `SELECT` queries allowed
  - Blocks `DROP`, `DELETE`, `UPDATE`, `INSERT`, `ALTER`, etc.
  - Enforces result limits

### Database Permissions

It's recommended to create a read-only database user:

```sql
-- Create read-only user
CREATE USER prtg_reader WITH PASSWORD 'secure_password';

-- Grant connect permission
GRANT CONNECT ON DATABASE prtg_data_exporter TO prtg_reader;

-- Grant schema usage
GRANT USAGE ON SCHEMA public TO prtg_reader;

-- Grant SELECT on all tables
GRANT SELECT ON ALL TABLES IN SCHEMA public TO prtg_reader;

-- Grant SELECT on future tables (optional)
ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT ON TABLES TO prtg_reader;
```

## Performance

### Connection Pool

The server uses a connection pool with these defaults:

- Max open connections: 25
- Max idle connections: 5
- Connection max lifetime: 5 minutes
- Query timeout: 30 seconds

### Query Optimization

- All queries include `LIMIT` clauses to prevent large result sets
- Indexes on PRTG tables are preserved from the Data Exporter schema
- Queries use `INNER JOIN` for optimal performance

## Troubleshooting

### Connection Issues

```bash
# Test database connectivity
psql -h localhost -U prtg_reader -d prtg_data_exporter -c "SELECT COUNT(*) FROM prtg_sensor;"

# Check server logs (stderr)
./mcp-server-prtg 2> server.log
```

### Common Issues

**"Database password is required"**
- Ensure `PRTG_DB_PASSWORD` environment variable is set

**"Failed to ping database"**
- Verify PostgreSQL is running and accessible
- Check firewall rules and PostgreSQL `pg_hba.conf`

**"Sensor not found"**
- Verify PRTG Data Exporter is running and exporting data
- Check that the sensor ID exists in the database

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make verify` to ensure quality
5. Submit a pull request

## License

[MIT License](LICENSE)

## Support

For issues and questions:
- GitHub Issues: https://github.com/matthieu/mcp-server-prtg/issues
- PRTG Documentation: https://www.paessler.com/manuals/prtg
- MCP Documentation: https://modelcontextprotocol.io

## Acknowledgments

- Built with [mcp-go](https://github.com/mark3labs/mcp-go)
- Designed for [PRTG Network Monitor](https://www.paessler.com/prtg)
- Inspired by the [Model Context Protocol](https://modelcontextprotocol.io)
