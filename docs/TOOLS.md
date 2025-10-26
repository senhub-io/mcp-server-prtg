# MCP Tools Reference

Complete reference documentation for all 6 MCP tools provided by MCP Server PRTG.

## Table of Contents

- [Overview](#overview)
- [Status Codes](#status-codes)
- [Tools](#tools)
  - [prtg_get_sensors](#prtg_get_sensors)
  - [prtg_get_sensor_status](#prtg_get_sensor_status)
  - [prtg_get_alerts](#prtg_get_alerts)
  - [prtg_device_overview](#prtg_device_overview)
  - [prtg_top_sensors](#prtg_top_sensors)
  - [prtg_query_sql](#prtg_query_sql)
- [Database Schema](#database-schema)
- [Common Patterns](#common-patterns)

## Overview

MCP Server PRTG exposes 6 tools through the Model Context Protocol. All tools return JSON responses with consistent formatting.

### Response Format

All tools return results in this format:

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found N result(s):\n\n{JSON data}"
    }
  ]
}
```

### Query Timeouts

All database queries have a 30-second timeout to prevent long-running queries from blocking the server.

### Limits

Most tools have configurable limits (default: 50-100 results) to prevent overwhelming responses. Limits can be adjusted per query.

## Status Codes

PRTG uses numeric status codes for sensors:

| Code | Status | Description |
|------|--------|-------------|
| 3 | Up | Sensor is operational and within normal parameters |
| 4 | Warning | Sensor detected a warning condition |
| 5 | Down | Sensor is down or not responding |
| 7 | Paused | Sensor monitoring is paused |
| 13 | Unknown | Sensor status is unknown |

## Tools

### prtg_get_sensors

Retrieve PRTG sensors with optional filters.

#### Description

Returns a list of sensors with their current status and metadata. Supports filtering by device name, sensor name, status, and tags.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `device_name` | string | No | - | Filter by device name (partial match, case-insensitive) |
| `sensor_name` | string | No | - | Filter by sensor name (partial match, case-insensitive) |
| `status` | integer | No | - | Filter by status code (3=Up, 4=Warning, 5=Down, 7=Paused) |
| `tags` | string | No | - | Filter by tag name (partial match) |
| `limit` | integer | No | 1000 | Maximum number of results |

#### Examples

**List all sensors:**
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "limit": 50
  }
}
```

**Find all ping sensors:**
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "sensor_name": "ping",
    "limit": 100
  }
}
```

**Find down sensors on specific device:**
```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "device_name": "web-prod",
    "status": 5,
    "limit": 50
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 2 result(s):\n\n[
  {
    \"id\": 12345,
    \"server_id\": 1,
    \"name\": \"Ping\",
    \"sensor_type\": \"ping\",
    \"device_id\": 6789,
    \"device_name\": \"web-prod-01\",
    \"scanning_interval_secs\": 60,
    \"status\": 3,
    \"status_text\": \"Up\",
    \"last_check_utc\": \"2025-10-26T10:30:00Z\",
    \"last_up_utc\": \"2025-10-26T10:30:00Z\",
    \"last_down_utc\": null,
    \"priority\": 3,
    \"message\": \"OK\",
    \"uptime_since_secs\": 86400.5,
    \"downtime_since_secs\": null,
    \"full_path\": \"PRTG Network Monitor/Production/Web Servers/web-prod-01\",
    \"tags\": \"production,webserver\"
  },
  {
    \"id\": 12346,
    \"server_id\": 1,
    \"name\": \"HTTP\",
    \"sensor_type\": \"http\",
    \"device_id\": 6789,
    \"device_name\": \"web-prod-01\",
    \"scanning_interval_secs\": 60,
    \"status\": 3,
    \"status_text\": \"Up\",
    \"last_check_utc\": \"2025-10-26T10:29:45Z\",
    \"last_up_utc\": \"2025-10-26T10:29:45Z\",
    \"last_down_utc\": null,
    \"priority\": 4,
    \"message\": \"OK (200 ms)\",
    \"uptime_since_secs\": 172800.2,
    \"downtime_since_secs\": null,
    \"full_path\": \"PRTG Network Monitor/Production/Web Servers/web-prod-01\",
    \"tags\": \"production,webserver,http\"
  }
]"
    }
  ]
}
```

#### Notes

- Tag filtering is currently disabled for performance reasons
- Results are ordered by sensor name
- Uses case-insensitive partial matching for name filters
- Null values indicate missing or not applicable data (e.g., `last_down_utc` for sensors that never went down)

---

### prtg_get_sensor_status

Get detailed current status of a specific sensor by ID.

#### Description

Returns comprehensive information about a single sensor including current values, uptime/downtime statistics, and status information.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sensor_id` | integer | **Yes** | - | The sensor ID to query |

#### Examples

**Get sensor details:**
```json
{
  "name": "prtg_get_sensor_status",
  "arguments": {
    "sensor_id": 12345
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 1 result(s):\n\n{
  \"id\": 12345,
  \"server_id\": 1,
  \"name\": \"Ping\",
  \"sensor_type\": \"ping\",
  \"device_id\": 6789,
  \"device_name\": \"web-prod-01\",
  \"scanning_interval_secs\": 60,
  \"status\": 3,
  \"status_text\": \"Up\",
  \"last_check_utc\": \"2025-10-26T10:30:00Z\",
  \"last_up_utc\": \"2025-10-26T10:30:00Z\",
  \"last_down_utc\": \"2025-10-25T08:15:30Z\",
  \"priority\": 3,
  \"message\": \"OK (15 ms)\",
  \"uptime_since_secs\": 93870.5,
  \"downtime_since_secs\": 0,
  \"full_path\": \"PRTG Network Monitor/Production/Web Servers/web-prod-01\",
  \"tags\": \"production,webserver,monitoring\"
}"
    }
  ]
}
```

#### Error Response

If sensor ID is not found:
```json
{
  "error": "failed to get sensor: sensor not found"
}
```

#### Notes

- Returns all available sensor metadata including tags
- Uptime/downtime are in seconds since last status change
- Priority indicates sensor importance (1=lowest, 5=highest)
- Message contains the last status message from PRTG

---

### prtg_get_alerts

Retrieve sensors in alert state (not Up).

#### Description

Returns all sensors that are currently in a non-operational state (Warning, Down, or other error states). Useful for monitoring and alerting.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `hours` | integer | No | 24 | Only include alerts from the last N hours (0 = all) |
| `status` | integer | No | - | Filter by specific status (4=Warning, 5=Down) |
| `device_name` | string | No | - | Filter by device name (partial match) |

#### Examples

**Get all current alerts:**
```json
{
  "name": "prtg_get_alerts",
  "arguments": {
    "hours": 24
  }
}
```

**Get only down sensors:**
```json
{
  "name": "prtg_get_alerts",
  "arguments": {
    "status": 5,
    "hours": 0
  }
}
```

**Get alerts for specific device:**
```json
{
  "name": "prtg_get_alerts",
  "arguments": {
    "device_name": "database",
    "hours": 48
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 3 result(s):\n\n[
  {
    \"id\": 54321,
    \"server_id\": 1,
    \"name\": \"Disk Free\",
    \"sensor_type\": \"snmpdisk\",
    \"device_id\": 9876,
    \"device_name\": \"db-prod-02\",
    \"scanning_interval_secs\": 300,
    \"status\": 4,
    \"status_text\": \"Warning\",
    \"last_check_utc\": \"2025-10-26T10:25:00Z\",
    \"last_up_utc\": \"2025-10-25T14:30:00Z\",
    \"last_down_utc\": null,
    \"priority\": 4,
    \"message\": \"Disk C: 85% full\",
    \"uptime_since_secs\": 0,
    \"downtime_since_secs\": 71700.0,
    \"full_path\": \"PRTG Network Monitor/Production/Database Servers/db-prod-02\",
    \"tags\": \"production,database,disk\"
  },
  {
    \"id\": 11111,
    \"server_id\": 1,
    \"name\": \"HTTP\",
    \"sensor_type\": \"http\",
    \"device_id\": 2222,
    \"device_name\": \"api-prod-01\",
    \"scanning_interval_secs\": 60,
    \"status\": 5,
    \"status_text\": \"Down\",
    \"last_check_utc\": \"2025-10-26T10:30:00Z\",
    \"last_up_utc\": \"2025-10-26T09:45:00Z\",
    \"last_down_utc\": \"2025-10-26T09:45:30Z\",
    \"priority\": 5,
    \"message\": \"HTTP/1.1 500 Internal Server Error\",
    \"uptime_since_secs\": 0,
    \"downtime_since_secs\": 2670.0,
    \"full_path\": \"PRTG Network Monitor/Production/API Servers/api-prod-01\",
    \"tags\": \"production,api,http\"
  }
]"
    }
  ]
}
```

#### Notes

- Results are ordered by priority (descending), then status, then name
- Limited to 100 results maximum
- `hours=0` returns all alerts regardless of age
- Sensors with status 3 (Up) are never included

---

### prtg_device_overview

Get a complete overview of a device including all its sensors and statistics.

#### Description

Returns comprehensive information about a device, including all its sensors and aggregated statistics (up/down/warning counts).

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `device_name` | string | **Yes** | - | Device name to query (partial match, case-insensitive) |

#### Examples

**Get device overview:**
```json
{
  "name": "prtg_device_overview",
  "arguments": {
    "device_name": "web-prod-01"
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 15 result(s):\n\n{
  \"device\": {
    \"id\": 6789,
    \"server_id\": 1,
    \"name\": \"web-prod-01\",
    \"host\": \"192.168.1.100\",
    \"group_id\": 111,
    \"group_name\": \"Web Servers\",
    \"full_path\": \"PRTG Network Monitor/Production/Web Servers\",
    \"tree_depth\": 3,
    \"sensor_count\": 15
  },
  \"sensors\": [
    {
      \"id\": 12345,
      \"name\": \"Ping\",
      \"sensor_type\": \"ping\",
      \"status\": 3,
      \"status_text\": \"Up\",
      \"message\": \"OK (15 ms)\",
      \"uptime_since_secs\": 86400.5,
      \"downtime_since_secs\": null,
      \"priority\": 3
    },
    {
      \"id\": 12346,
      \"name\": \"HTTP\",
      \"sensor_type\": \"http\",
      \"status\": 3,
      \"status_text\": \"Up\",
      \"message\": \"OK (200 ms)\",
      \"uptime_since_secs\": 172800.2,
      \"downtime_since_secs\": null,
      \"priority\": 4
    }
  ],
  \"total_sensors\": 15,
  \"up_sensors\": 13,
  \"down_sensors\": 1,
  \"warn_sensors\": 1
}"
    }
  ]
}
```

#### Error Response

If device is not found:
```json
{
  "error": "failed to get device overview: device not found"
}
```

#### Notes

- Uses case-insensitive partial matching for device name
- Returns the first matching device if multiple matches exist
- Sensors are ordered by status, then name
- Statistics include counts of sensors in each state
- Includes device hierarchy information (group, path, depth)

---

### prtg_top_sensors

Get top sensors ranked by various metrics.

#### Description

Returns sensors ranked by uptime, downtime, or alert frequency. Useful for identifying problematic or reliable sensors.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `metric` | string | No | `downtime` | Metric to rank by: `uptime`, `downtime`, or `alerts` |
| `sensor_type` | string | No | - | Filter by sensor type (e.g., `ping`, `http`) |
| `limit` | integer | No | 10 | Number of results to return |
| `hours` | integer | No | 24 | Time window in hours (not currently used in queries) |

#### Metrics

- **uptime**: Sensors with the longest uptime (most reliable)
- **downtime**: Sensors with the longest downtime (most problematic)
- **alerts**: Sensors currently in non-Up status, ordered by priority

#### Examples

**Top 10 sensors with most downtime:**
```json
{
  "name": "prtg_top_sensors",
  "arguments": {
    "metric": "downtime",
    "limit": 10
  }
}
```

**Most reliable ping sensors:**
```json
{
  "name": "prtg_top_sensors",
  "arguments": {
    "metric": "uptime",
    "sensor_type": "ping",
    "limit": 20
  }
}
```

**Top alert-generating sensors:**
```json
{
  "name": "prtg_top_sensors",
  "arguments": {
    "metric": "alerts",
    "limit": 15
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 10 result(s):\n\n[
  {
    \"id\": 99999,
    \"server_id\": 1,
    \"name\": \"Backup Service\",
    \"sensor_type\": \"exexml\",
    \"device_id\": 8888,
    \"device_name\": \"backup-server-01\",
    \"scanning_interval_secs\": 300,
    \"status\": 5,
    \"status_text\": \"Down\",
    \"last_check_utc\": \"2025-10-26T10:30:00Z\",
    \"last_up_utc\": \"2025-10-23T15:20:00Z\",
    \"last_down_utc\": \"2025-10-23T15:21:00Z\",
    \"priority\": 4,
    \"message\": \"Service not responding\",
    \"uptime_since_secs\": 0,
    \"downtime_since_secs\": 242340.0,
    \"full_path\": \"PRTG Network Monitor/Infrastructure/Backup Servers/backup-server-01\",
    \"tags\": \"backup,service\"
  }
]"
    }
  ]
}
```

#### Notes

- For `uptime` metric: NULL values are sorted last
- For `downtime` metric: NULL values are sorted last
- For `alerts` metric: Only returns sensors with status != 3 (Up)
- Results are limited to the specified `limit` parameter
- `sensor_type` uses case-insensitive partial matching

---

### prtg_query_sql

Execute a custom SQL query on the PRTG database.

#### Description

Executes custom SELECT queries for advanced analysis not covered by other tools. **SELECT only** - write operations are blocked for security.

#### Security

This tool implements multiple security measures:
- Only SELECT queries are allowed
- Forbidden keywords: DROP, DELETE, UPDATE, INSERT, ALTER, CREATE, TRUNCATE, EXEC, EXECUTE
- SQL comments (`--`, `/*`) are blocked to prevent bypass attempts
- Maximum limit enforced (1000 results)
- 30-second query timeout

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | **Yes** | - | SQL SELECT query to execute |
| `limit` | integer | No | 100 | Maximum number of results (max: 1000) |

#### Examples

**Find sensors not checked in over 1 hour:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT id, name, device_name, last_check_utc FROM prtg_sensor WHERE last_check_utc < NOW() - INTERVAL '1 hour'",
    "limit": 50
  }
}
```

**Count sensors by status:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT status, COUNT(*) as count FROM prtg_sensor GROUP BY status ORDER BY status"
  }
}
```

**Find devices with multiple down sensors:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT d.name, COUNT(*) as down_count FROM prtg_sensor s JOIN prtg_device d ON s.prtg_device_id = d.id WHERE s.status = 5 GROUP BY d.name HAVING COUNT(*) > 1"
  }
}
```

**Get sensors with longest downtime:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT name, device_name, downtime_since_secs, status FROM prtg_sensor WHERE downtime_since_secs IS NOT NULL ORDER BY downtime_since_secs DESC",
    "limit": 20
  }
}
```

#### Response Format

```json
{
  "content": [
    {
      "type": "text",
      "text": "Found 3 result(s):\n\n[
  {
    \"id\": 12345,
    \"name\": \"Ping\",
    \"status\": 3,
    \"last_check_utc\": \"2025-10-26T10:30:00Z\"
  },
  {
    \"id\": 12346,
    \"name\": \"HTTP\",
    \"status\": 3,
    \"last_check_utc\": \"2025-10-26T10:29:45Z\"
  },
  {
    \"id\": 54321,
    \"name\": \"Disk Free\",
    \"status\": 4,
    \"last_check_utc\": \"2025-10-26T10:25:00Z\"
  }
]"
    }
  ]
}
```

#### Error Responses

**Forbidden operation:**
```json
{
  "error": "query execution failed: only SELECT queries are allowed"
}
```

**Forbidden keyword:**
```json
{
  "error": "query execution failed: query contains forbidden keyword: DELETE"
}
```

#### Notes

- If query doesn't include LIMIT, the `limit` parameter is automatically appended
- Results are returned as an array of objects with column names as keys
- All PostgreSQL data types are supported
- Use fully qualified table names (see [Database Schema](#database-schema))
- Query timeout is 30 seconds

---

## Database Schema

The PRTG database contains the following main tables:

### prtg_sensor

Main sensor data table.

**Columns:**
- `id` (integer) - Sensor ID (primary key)
- `prtg_server_address_id` (integer) - Server ID
- `name` (string) - Sensor name
- `sensor_type` (string) - Sensor type (ping, http, snmp, etc.)
- `prtg_device_id` (integer) - Parent device ID
- `status` (integer) - Current status code (3=Up, 4=Warning, 5=Down, etc.)
- `priority` (integer) - Priority level (1-5)
- `message` (string) - Last status message
- `last_check_utc` (timestamp) - Last check time
- `last_up_utc` (timestamp) - Last time sensor was up
- `last_down_utc` (timestamp) - Last time sensor went down
- `scanning_interval_seconds` (integer) - Check interval
- `uptime_since_seconds` (float) - Uptime duration in seconds
- `downtime_since_seconds` (float) - Downtime duration in seconds
- `full_path` (string) - Full PRTG hierarchy path

### prtg_device

Device information table.

**Columns:**
- `id` (integer) - Device ID (primary key)
- `prtg_server_address_id` (integer) - Server ID
- `name` (string) - Device name
- `host` (string) - Device hostname or IP
- `prtg_group_id` (integer) - Parent group ID
- `tree_depth` (integer) - Hierarchy depth

### prtg_device_path

Device path/hierarchy information.

**Columns:**
- `device_id` (integer) - Device ID
- `prtg_server_address_id` (integer) - Server ID
- `path` (string) - Full device path

### prtg_sensor_path

Sensor path/hierarchy information.

**Columns:**
- `sensor_id` (integer) - Sensor ID
- `prtg_server_address_id` (integer) - Server ID
- `path` (string) - Full sensor path

### prtg_tag

Tag definitions.

**Columns:**
- `id` (integer) - Tag ID (primary key)
- `name` (string) - Tag name

### prtg_sensor_tag

Sensor-to-tag associations.

**Columns:**
- `prtg_sensor_id` (integer) - Sensor ID
- `prtg_tag_id` (integer) - Tag ID
- `prtg_server_address_id` (integer) - Server ID

### prtg_group

Group/folder information.

**Columns:**
- `id` (integer) - Group ID (primary key)
- `name` (string) - Group name
- `prtg_server_address_id` (integer) - Server ID

## Common Patterns

### Find All Down Sensors

```json
{
  "name": "prtg_get_sensors",
  "arguments": {
    "status": 5,
    "limit": 100
  }
}
```

Or with SQL:
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT id, name, device_name, message FROM prtg_sensor WHERE status = 5"
  }
}
```

### Monitor Specific Device

```json
{
  "name": "prtg_device_overview",
  "arguments": {
    "device_name": "web-prod-01"
  }
}
```

### Get Alert Summary

```json
{
  "name": "prtg_get_alerts",
  "arguments": {
    "hours": 24
  }
}
```

### Find Most Unreliable Sensors

```json
{
  "name": "prtg_top_sensors",
  "arguments": {
    "metric": "downtime",
    "limit": 20
  }
}
```

### Complex Analysis with SQL

**Sensors down for more than 4 hours:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT s.name, d.name as device_name, s.downtime_since_seconds / 3600 as hours_down FROM prtg_sensor s JOIN prtg_device d ON s.prtg_device_id = d.id WHERE s.status = 5 AND s.downtime_since_seconds > 14400 ORDER BY s.downtime_since_seconds DESC"
  }
}
```

**Count sensors by type and status:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT sensor_type, status, COUNT(*) as count FROM prtg_sensor GROUP BY sensor_type, status ORDER BY sensor_type, status"
  }
}
```

**Find devices with high alert rates:**
```json
{
  "name": "prtg_query_sql",
  "arguments": {
    "query": "SELECT d.name, COUNT(*) as alert_count FROM prtg_sensor s JOIN prtg_device d ON s.prtg_device_id = d.id WHERE s.status != 3 GROUP BY d.name ORDER BY alert_count DESC"
  }
}
```

## See Also

- [Usage Guide](USAGE.md)
- [Configuration Guide](CONFIGURATION.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [GitHub Repository](https://github.com/senhub-io/mcp-server-prtg)
