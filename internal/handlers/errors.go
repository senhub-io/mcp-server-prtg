// internal/handlers/errors.go
package handlers

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ErrorGuidance returns an MCP result with a pedagogical error message
func ErrorGuidance(title, explanation string, suggestions []string) *mcp.CallToolResult {
	var sb strings.Builder
	sb.WriteString("ERROR: ")
	sb.WriteString(title)
	sb.WriteString("\n\n")
	sb.WriteString(explanation)
	sb.WriteString("\n\n")

	if len(suggestions) > 0 {
		sb.WriteString("How to resolve:\n")
		for i, s := range suggestions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: sb.String(),
			},
		},
	}
}

// Predefined errors with guidance
var (
	ErrInvalidSensorID = func() *mcp.CallToolResult {
		return ErrorGuidance(
			"Invalid sensor_id",
			"The sensor_id parameter must be a positive integer corresponding to an existing PRTG sensor.",
			[]string{
				"Search by name: `prtg_search search_term=\"sensor_name\"`",
				"List alerts (with IDs): `prtg_get_alerts`",
				"Explore a device: `prtg_device_overview device_name=\"device_name\"`",
			},
		)
	}

	ErrDeviceNotFound = func(name string) *mcp.CallToolResult {
		return ErrorGuidance(
			fmt.Sprintf("Device '%s' not found", name),
			"No device matches this name in the PRTG database.",
			[]string{
				fmt.Sprintf("Check spelling and search: `prtg_search search_term=\"%s\"`", name),
				"List all devices: `prtg_get_sensors limit=1` then check device_name",
				"Explore hierarchy: `prtg_get_hierarchy`",
			},
		)
	}

	ErrSensorNotFound = func(id int) *mcp.CallToolResult {
		return ErrorGuidance(
			fmt.Sprintf("Sensor ID %d not found", id),
			"This sensor_id does not exist or has been deleted in PRTG.",
			[]string{
				"Data may be out of sync. Verify that the Data Exporter is active.",
				"Search sensor by name: `prtg_search search_term=\"name\"`",
				"List active sensors: `prtg_get_sensors limit=50`",
			},
		)
	}

	ErrEmptySearchTerm = func() *mcp.CallToolResult {
		return ErrorGuidance(
			"Empty search term",
			"The search_term parameter is required to perform a search.",
			[]string{
				"Search for a device: `prtg_search search_term=\"server\"`",
				"Search by type: `prtg_search search_term=\"ping\"`",
				"Search by group: `prtg_search search_term=\"production\"`",
			},
		)
	}

	ErrInvalidTimeRange = func() *mcp.CallToolResult {
		return ErrorGuidance(
			"Invalid time range",
			"end_time must be after start_time. Expected format: RFC3339 (e.g., 2025-01-15T14:00:00Z)",
			[]string{
				"Example for last 24h: start_time=2025-01-14T00:00:00Z, end_time=2025-01-15T00:00:00Z",
				"Use time_type instead for standard periods: `prtg_get_sensor_timeseries sensor_id=X time_type=short`",
				"Available time_type: live (minutes), short (24h), medium (7d), long (30d)",
			},
		)
	}

	ErrCustomQueriesDisabled = func() *mcp.CallToolResult {
		return ErrorGuidance(
			"Custom SQL queries disabled",
			"For security reasons, prtg_query_sql is disabled by default.",
			[]string{
				"Use predefined tools: prtg_get_sensors, prtg_get_alerts, etc.",
				"To enable: set allow_custom_queries=true in config.yaml (not recommended in production)",
				"Contact your administrator if needed",
			},
		)
	}
)
