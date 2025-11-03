# MCP Tools Reference

Complete reference documentation for all 15 MCP tools provided by MCP Server PRTG.

## Table of Contents

- [Overview](#overview)
- [Status Codes](#status-codes)
- [PostgreSQL-Based Tools (12)](#postgresql-based-tools)
  - [prtg_get_sensors](#prtg_get_sensors)
  - [prtg_get_sensor_status](#prtg_get_sensor_status)
  - [prtg_get_alerts](#prtg_get_alerts)
  - [prtg_device_overview](#prtg_device_overview)
  - [prtg_top_sensors](#prtg_top_sensors)
  - [prtg_get_hierarchy](#prtg_get_hierarchy)
  - [prtg_search](#prtg_search)
  - [prtg_get_groups](#prtg_get_groups)
  - [prtg_get_tags](#prtg_get_tags)
  - [prtg_get_business_processes](#prtg_get_business_processes)
  - [prtg_get_statistics](#prtg_get_statistics)
  - [prtg_query_sql](#prtg_query_sql)
- [PRTG API v2 Tools (3)](#prtg-api-v2-tools)
  - [prtg_get_channel_current_values](#prtg_get_channel_current_values)
  - [prtg_get_sensor_timeseries](#prtg_get_sensor_timeseries)
  - [prtg_get_sensor_history_custom](#prtg_get_sensor_history_custom)
- [Database Schema](#database-schema)
- [Common Patterns](#common-patterns)

## Overview

MCP Server PRTG exposes 15 tools through the Model Context Protocol:
- **12 PostgreSQL-based tools** - Query sensor status, configuration, and hierarchy from PRTG Data Exporter database
- **3 PRTG API v2 tools** - Query historical metrics and real-time channel data directly from PRTG Core Server

All tools return JSON responses with consistent visual formatting including markdown tables and complete JSON data.

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

### prtg_get_hierarchy

Navigate PRTG hierarchy tree structure (groups, devices, sensors).

#### Description

Returns the hierarchical structure of PRTG objects starting from a specific group or the root. Useful for understanding infrastructure organization.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `group_name` | string | No | - | Starting group name (partial match, case-insensitive). If not provided, starts from root |
| `include_sensors` | boolean | No | false | Include sensors in the hierarchy |
| `max_depth` | integer | No | 3 | Maximum depth to traverse (1-10) |

#### Examples

**Get full hierarchy from root:**
```json
{
  "name": "prtg_get_hierarchy",
  "arguments": {
    "max_depth": 2
  }
}
```

**Get hierarchy for specific group with sensors:**
```json
{
  "name": "prtg_get_hierarchy",
  "arguments": {
    "group_name": "Production",
    "include_sensors": true,
    "max_depth": 3
  }
}
```

#### Notes

- Returns nested JSON structure representing the hierarchy
- Visual formatting shows groups, devices, and optionally sensors
- Includes probe status and tree depth information
- Limited to max_depth to prevent excessive data retrieval

---

### prtg_search

Universal search across PRTG groups, devices, and sensors.

#### Description

Search for PRTG objects by name across all object types. Returns matching groups, devices, and sensors in a single query.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `search_term` | string | **Yes** | - | Search term (partial match, case-insensitive) |
| `limit` | integer | No | 50 | Maximum results per object type |

#### Examples

**Search for "web" objects:**
```json
{
  "name": "prtg_search",
  "arguments": {
    "search_term": "web",
    "limit": 20
  }
}
```

**Search for production resources:**
```json
{
  "name": "prtg_search",
  "arguments": {
    "search_term": "prod"
  }
}
```

#### Response Format

Returns results grouped by type:
```json
{
  "groups": [ /* matching groups */ ],
  "devices": [ /* matching devices */ ],
  "sensors": [ /* matching sensors */ ]
}
```

#### Notes

- Searches across all PRTG object types simultaneously
- Uses case-insensitive partial matching
- Results are limited per object type
- Visual formatting shows breakdown by object type with counts

---

### prtg_get_groups

List PRTG groups and probes with filtering options.

#### Description

Returns groups and probes (root-level groups) from PRTG. Groups organize devices hierarchically, and probes are special groups representing PRTG probe nodes.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `group_name` | string | No | - | Filter by group name (partial match, case-insensitive) |
| `parent_id` | integer | No | - | Filter by parent group ID (shows direct children) |
| `limit` | integer | No | 100 | Maximum number of results |

#### Examples

**List all groups:**
```json
{
  "name": "prtg_get_groups",
  "arguments": {
    "limit": 50
  }
}
```

**Find production groups:**
```json
{
  "name": "prtg_get_groups",
  "arguments": {
    "group_name": "production"
  }
}
```

**Get children of specific group:**
```json
{
  "name": "prtg_get_groups",
  "arguments": {
    "parent_id": 100
  }
}
```

#### Response Format

Visual table showing groups with:
- Group ID and name
- Type (Probe or Group) with emoji indicators
- Tree depth and full path
- Breakdown statistics (probe count vs group count)

#### Notes

- Probes are indicated with 📡 emoji
- Regular groups use 📁 emoji
- Results include full hierarchy path
- Tree depth shows nesting level in PRTG structure

---

### prtg_get_tags

List PRTG tags with usage statistics.

#### Description

Returns all tags defined in PRTG with their usage count. Tags are labels applied to sensors for organization and filtering.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `tag_name` | string | No | - | Filter by tag name (partial match, case-insensitive) |
| `limit` | integer | No | 100 | Maximum number of results |

#### Examples

**List all tags:**
```json
{
  "name": "prtg_get_tags",
  "arguments": {}
}
```

**Find production tags:**
```json
{
  "name": "prtg_get_tags",
  "arguments": {
    "tag_name": "prod"
  }
}
```

#### Response Format

Visual table showing:
- Tag ID and name
- Number of sensors using the tag
- Total sensor associations
- Average sensors per tag

#### Notes

- Includes tags with zero usage
- Shows sensor count for each tag
- Useful for tag management and cleanup
- Can identify most/least used tags

---

### prtg_get_business_processes

Query PRTG Business Process sensors.

#### Description

Returns Business Process sensors, which are special sensors that aggregate status from multiple source sensors to monitor complete business workflows.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `process_name` | string | No | - | Filter by process name (partial match, case-insensitive) |
| `status` | integer | No | - | Filter by status (3=Up, 4=Warning, 5=Down, etc.) |
| `limit` | integer | No | 100 | Maximum number of results |

#### Examples

**List all business processes:**
```json
{
  "name": "prtg_get_business_processes",
  "arguments": {}
}
```

**Find failing processes:**
```json
{
  "name": "prtg_get_business_processes",
  "arguments": {
    "status": 5
  }
}
```

**Search for specific process:**
```json
{
  "name": "prtg_get_business_processes",
  "arguments": {
    "process_name": "E-commerce"
  }
}
```

#### Response Format

Visual table showing:
- Process ID and name
- Current status with emoji indicators
- Priority level
- Device and last check time
- Status message
- Status breakdown statistics (up/warning/down counts)

#### Notes

- Business Process sensors aggregate status from source sensors
- Results ordered by priority (highest first)
- Shows complete process health overview
- Useful for high-level business monitoring

---

### prtg_get_statistics

Get server-wide aggregated PRTG statistics.

#### Description

Returns comprehensive statistics about the entire PRTG installation, including counts, status breakdown, and sensor type distribution.

#### Parameters

No parameters required - returns global statistics.

#### Examples

**Get PRTG statistics:**
```json
{
  "name": "prtg_get_statistics",
  "arguments": {}
}
```

#### Response Format

Returns comprehensive statistics:
```json
{
  "total_sensors": 1234,
  "total_devices": 156,
  "total_groups": 45,
  "total_tags": 89,
  "total_probes": 3,
  "avg_sensors_per_device": 7.9,
  "sensors_by_status": {
    "Up": 1100,
    "Warning": 50,
    "Down": 34,
    "Paused (User)": 50
  },
  "top_sensor_types": [
    {"type": "ping", "count": 234},
    {"type": "http", "count": 189},
    {"type": "snmp", "count": 156}
  ]
}
```

Visual output includes:
- Overall infrastructure counts
- Sensor status breakdown with percentages
- Top 15 sensor types with distribution
- Average sensors per device metric

#### Notes

- Provides health overview of entire PRTG installation
- Status breakdown shows percentage distribution
- Sensor type distribution helps identify monitoring focus
- No parameters needed - always returns global stats
- Useful for capacity planning and health monitoring

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

## PRTG API v2 Tools

These tools query data directly from PRTG Core Server via API v2. They require PRTG API v2 configuration in `config.yaml` (see [CONFIGURATION.md](CONFIGURATION.md)).

### prtg_get_channel_current_values

**PRIMARY TOOL for checking sensor current state and discovering available channels.**

#### Description

Returns ALL channels of a sensor with their current values, names, units, and last update timestamp. This is the main tool for understanding a sensor's current state.

Each PRTG sensor has multiple channels (individual measurements). Examples:
- **SSL sensors**: 'Days to Expiration', 'Response Time'
- **Server sensors**: 'CPU Load', 'Memory Usage', 'Disk Space'
- **Network sensors**: 'Traffic In', 'Traffic Out', 'Packet Loss'

**Always use this tool first** when asked about a sensor's current state, values, or status.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sensor_id` | integer | **Yes** | - | PRTG sensor ID (use `prtg_get_sensors` to find sensor IDs) |

#### Examples

**Get all current channel values for a sensor:**
```json
{
  "name": "prtg_get_channel_current_values",
  "arguments": {
    "sensor_id": 12345
  }
}
```

#### Response Format

Returns a markdown table with current channel values:

```
# Current Channel Values - Sensor 12345

Total channels: 5

| Channel | Value | Unit | Timestamp |
|---------|-------|------|----------|
| Response Time | 45.23 | ms | 2025-10-26 10:30:00 |
| Days to Expiration | 89.00 | days | 2025-10-26 10:30:00 |
| Traffic In | 1234567.89 | kbit/s | 2025-10-26 10:30:00 |
| Traffic Out | 987654.32 | kbit/s | 2025-10-26 10:30:00 |
| Downtime | 0.00 | % | 2025-10-26 10:30:00 |
```

#### Notes

- This tool queries PRTG API v2 in real-time (not PostgreSQL database)
- Returns the most recent measurement for each channel
- Channel names, units, and values come directly from PRTG
- Use this instead of `prtg_get_sensor_status` when you need detailed channel data
- For historical trends, use `prtg_get_sensor_timeseries`

---

### prtg_get_sensor_timeseries

Retrieve **HISTORICAL** time series data for analyzing trends over time.

#### Description

Returns time-stamped measurements showing how channel values evolved over predefined time periods. Use this to analyze performance trends, identify when issues started, compare metrics between time periods, and detect patterns in historical data.

**For CURRENT values, use `prtg_get_channel_current_values` instead.**

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sensor_id` | integer | **Yes** | - | PRTG sensor ID |
| `time_type` | string | **Yes** | - | Time period: `live`, `short`, `medium`, or `long` |

#### Time Periods

| Time Type | Coverage | Typical Use Case |
|-----------|----------|------------------|
| `live` | Last few minutes | Real-time monitoring, immediate troubleshooting |
| `short` | Last 24 hours | Daily trend analysis, recent performance issues |
| `medium` | Last 7 days | Weekly trends, recurring issues |
| `long` | Last 30+ days | Monthly trends, long-term capacity planning |

#### Examples

**Analyze last 24 hours of sensor data:**
```json
{
  "name": "prtg_get_sensor_timeseries",
  "arguments": {
    "sensor_id": 12345,
    "time_type": "short"
  }
}
```

**Check recent activity (live data):**
```json
{
  "name": "prtg_get_sensor_timeseries",
  "arguments": {
    "sensor_id": 54321,
    "time_type": "live"
  }
}
```

**Analyze weekly trends:**
```json
{
  "name": "prtg_get_sensor_timeseries",
  "arguments": {
    "sensor_id": 67890,
    "time_type": "medium"
  }
}
```

#### Response Format

Returns a markdown table with time-stamped measurements:

```
# Time Series Data - Sensor 12345 (short)

Total data points: 145
Channels: Response Time, Traffic In, Traffic Out

## Measurements

| Timestamp | Response Time | Traffic In | Traffic Out |
|-----------|---------------|------------|-------------|
| 2025-10-26 10:30:00 | 45.23 | 1234567.89 | 987654.32 |
| 2025-10-26 10:25:00 | 43.12 | 1198765.43 | 965432.10 |
| 2025-10-26 10:20:00 | 48.56 | 1345678.90 | 1023456.78 |
| 2025-10-26 10:15:00 | 42.34 | 1123456.78 | 945678.90 |
| ... | ... | ... | ... |
| 2025-10-25 10:35:00 | 44.67 | 1267890.12 | 978901.23 |
```

**Note:** If more than 15 data points exist, the table shows the first 10 and last 5 points with "..." indicating truncation.

#### Notes

- This tool queries PRTG API v2 for historical data
- Data granularity depends on PRTG's configured averaging intervals
- Not all channels may be available for all time periods
- For custom date ranges, use `prtg_get_sensor_history_custom`
- Large datasets may be truncated in display (summary provided)

---

### prtg_get_sensor_history_custom

Retrieve **HISTORICAL** data for a specific date/time range.

#### Description

Returns time series data for a custom time window. Use this when you need to analyze a specific incident timeframe (e.g., "what happened last Tuesday between 2pm-4pm").

**For CURRENT values, use `prtg_get_channel_current_values` instead.**

Useful for:
- Incident investigation and root cause analysis
- Comparing specific time windows
- Generating reports for past periods
- Analyzing scheduled events (maintenance windows, batch jobs)

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `sensor_id` | integer | **Yes** | - | PRTG sensor ID |
| `start_time` | string | **Yes** | - | Start time in RFC3339 format (e.g., `2025-10-30T00:00:00Z`) |
| `end_time` | string | **Yes** | - | End time in RFC3339 format (e.g., `2025-10-31T23:59:59Z`) |

#### Time Format

Use RFC3339 format for timestamps:
- **UTC**: `2025-10-30T14:00:00Z`
- **With timezone**: `2025-10-30T14:00:00+02:00`
- **Date only** (midnight UTC): `2025-10-30T00:00:00Z`

#### Examples

**Analyze specific incident window:**
```json
{
  "name": "prtg_get_sensor_history_custom",
  "arguments": {
    "sensor_id": 12345,
    "start_time": "2025-10-29T14:00:00Z",
    "end_time": "2025-10-29T16:00:00Z"
  }
}
```

**Full day analysis:**
```json
{
  "name": "prtg_get_sensor_history_custom",
  "arguments": {
    "sensor_id": 54321,
    "start_time": "2025-10-25T00:00:00Z",
    "end_time": "2025-10-25T23:59:59Z"
  }
}
```

**Compare before/after maintenance:**
```json
{
  "name": "prtg_get_sensor_history_custom",
  "arguments": {
    "sensor_id": 67890,
    "start_time": "2025-10-20T22:00:00Z",
    "end_time": "2025-10-21T02:00:00Z"
  }
}
```

#### Response Format

Returns a markdown table with time-stamped measurements:

```
# Time Series Data - Sensor 12345
Period: 2025-10-29 14:00:00 to 2025-10-29 16:00:00

Total data points: 48
Channels: CPU Load, Memory Usage, Disk I/O

## Measurements

| Timestamp | CPU Load | Memory Usage | Disk I/O |
|-----------|----------|--------------|----------|
| 2025-10-29 14:00:00 | 45.23 | 78.90 | 1234.56 |
| 2025-10-29 14:05:00 | 48.12 | 79.45 | 1298.76 |
| 2025-10-29 14:10:00 | 52.34 | 81.23 | 1456.78 |
| ... | ... | ... | ... |
| 2025-10-29 15:55:00 | 43.21 | 77.65 | 1198.90 |
```

#### Error Responses

**Invalid time format:**
```json
{
  "error": "Invalid start_time format (use RFC3339): parsing time \"2025-10-30\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"\" as \"T\""
}
```

**End before start:**
```json
{
  "error": "end_time must be after start_time"
}
```

#### Notes

- This tool queries PRTG API v2 for historical data
- Time range validation ensures end_time is after start_time
- Data granularity depends on PRTG's configured averaging intervals
- Very large time ranges may result in truncated display (summary provided)
- For predefined periods (24h, 7d, 30d), use `prtg_get_sensor_timeseries` instead

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
