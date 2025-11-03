// Package handlers implements MCP tools for querying PRTG historical metrics via API v2.
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matthieu/mcp-server-prtg/internal/prtg"
)

// PRTGClient interface for PRTG API operations.
type PRTGClient interface {
	GetTimeSeries(ctx context.Context, objectID int, timeType prtg.TimeSeriesType) (*prtg.TimeSeriesData, error)
	GetTimeSeriesCustom(ctx context.Context, objectID int, start, end time.Time) (*prtg.TimeSeriesData, error)
	GetChannelsBySensor(ctx context.Context, sensorID int) ([]prtg.Channel, error)
}

// MetricsToolHandler handles MCP tool requests for PRTG metrics/historical data.
type MetricsToolHandler struct {
	prtgClient PRTGClient
	handler    *ToolHandler // Reference to main handler for database queries
}

// NewMetricsToolHandler creates a new metrics tool handler.
func NewMetricsToolHandler(prtgClient PRTGClient, mainHandler *ToolHandler) *MetricsToolHandler {
	return &MetricsToolHandler{
		prtgClient: prtgClient,
		handler:    mainHandler,
	}
}

// RegisterMetricsTools registers all PRTG metrics-related MCP tools.
func (h *MetricsToolHandler) RegisterMetricsTools(s *server.MCPServer) {
	// Tool 1: prtg_get_sensor_timeseries
	s.AddTool(mcp.Tool{
		Name: "prtg_get_sensor_timeseries",
		Description: "Retrieve **HISTORICAL** time series data for analyzing trends over time. " +
			"Returns time-stamped measurements showing how channel values evolved. " +
			"**For CURRENT values, use prtg_get_channel_current_values instead.** " +
			"Use cases: analyze performance degradation over 24h, identify when an issue started, " +
			"compare metrics between time periods, detect patterns in historical data.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "PRTG sensor ID (use prtg_get_sensors to find sensor IDs)",
				},
				"time_type": map[string]interface{}{
					"type": "string",
					"enum": []string{"live", "short", "medium", "long"},
					"description": "Time period: 'live' (last minutes), 'short' (last 24h), " +
						"'medium' (last 7 days), 'long' (last 30+ days)",
				},
			},
			Required: []string{"sensor_id", "time_type"},
		},
	}, h.handleGetSensorTimeSeries)

	// Tool 2: prtg_get_sensor_history_custom
	s.AddTool(mcp.Tool{
		Name: "prtg_get_sensor_history_custom",
		Description: "Retrieve **HISTORICAL** data for a specific date/time range. " +
			"**For CURRENT values, use prtg_get_channel_current_values instead.** " +
			"Use this when you need to analyze a specific incident timeframe (e.g., 'what happened last Tuesday between 2pm-4pm'). " +
			"Useful for: incident investigation, comparing specific time windows, generating reports for past periods.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "PRTG sensor ID",
				},
				"start_time": map[string]interface{}{
					"type":        "string",
					"description": "Start time in RFC3339 format (e.g., '2025-10-30T00:00:00Z')",
				},
				"end_time": map[string]interface{}{
					"type":        "string",
					"description": "End time in RFC3339 format (e.g., '2025-10-31T23:59:59Z')",
				},
			},
			Required: []string{"sensor_id", "start_time", "end_time"},
		},
	}, h.handleGetSensorHistoryCustom)

	// Tool 3: prtg_get_channel_current_values
	s.AddTool(mcp.Tool{
		Name: "prtg_get_channel_current_values",
		Description: "**PRIMARY TOOL for checking sensor current state and discovering available channels.** " +
			"Returns ALL channels of a sensor with their current values, names, units, and last update time. " +
			"Each PRTG sensor has multiple channels (measurements). Examples: " +
			"SSL sensors → 'Days to Expiration', 'Response Time'; " +
			"Server sensors → 'CPU Load', 'Memory Usage', 'Disk Space'; " +
			"Network sensors → 'Traffic In', 'Traffic Out', 'Packet Loss'. " +
			"**ALWAYS use this tool first** when asked about a sensor's current state, values, or status. " +
			"Use prtg_get_sensor_timeseries only for historical trends.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "PRTG sensor ID",
				},
			},
			Required: []string{"sensor_id"},
		},
	}, h.handleGetChannelCurrentValues)
}

// handleGetSensorTimeSeries handles prtg_get_sensor_timeseries tool requests.
func (h *MetricsToolHandler) handleGetSensorTimeSeries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SensorID int    `json:"sensor_id"`
		TimeType string `json:"time_type"`
	}

	if err := parseArguments(request.Params.Arguments, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Validate time type
	timeType := prtg.TimeSeriesType(params.TimeType)
	validTypes := map[prtg.TimeSeriesType]bool{
		prtg.TimeSeriesLive:   true,
		prtg.TimeSeriesShort:  true,
		prtg.TimeSeriesMedium: true,
		prtg.TimeSeriesLong:   true,
	}

	if !validTypes[timeType] {
		return mcp.NewToolResultError("Invalid time_type. Must be: live, short, medium, or long"), nil
	}

	h.handler.logger.Info().
		Int("sensor_id", params.SensorID).
		Str("time_type", params.TimeType).
		Msg("Fetching sensor time series from PRTG API")

	// Fetch data from PRTG API
	data, err := h.prtgClient.GetTimeSeries(ctx, params.SensorID, timeType)
	if err != nil {
		h.handler.logger.Error().
			Err(err).
			Int("sensor_id", params.SensorID).
			Msg("Failed to fetch time series from PRTG API")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch time series: %v", err)), nil
	}

	// Format response for LLM
	formatted := formatTimeSeriesForLLM(data)

	return mcp.NewToolResultText(formatted), nil
}

// handleGetSensorHistoryCustom handles prtg_get_sensor_history_custom tool requests.
func (h *MetricsToolHandler) handleGetSensorHistoryCustom(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SensorID  int    `json:"sensor_id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := parseArguments(request.Params.Arguments, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Parse timestamps
	startTime, err := time.Parse(time.RFC3339, params.StartTime)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid start_time format (use RFC3339): %v", err)), nil
	}

	endTime, err := time.Parse(time.RFC3339, params.EndTime)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid end_time format (use RFC3339): %v", err)), nil
	}

	// Validate time range
	if endTime.Before(startTime) {
		return mcp.NewToolResultError("end_time must be after start_time"), nil
	}

	h.handler.logger.Info().
		Int("sensor_id", params.SensorID).
		Time("start", startTime).
		Time("end", endTime).
		Msg("Fetching custom time range from PRTG API")

	// Fetch data from PRTG API
	data, err := h.prtgClient.GetTimeSeriesCustom(ctx, params.SensorID, startTime, endTime)
	if err != nil {
		h.handler.logger.Error().
			Err(err).
			Int("sensor_id", params.SensorID).
			Msg("Failed to fetch custom time series from PRTG API")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch time series: %v", err)), nil
	}

	// Format response for LLM
	formatted := formatTimeSeriesForLLM(data)

	return mcp.NewToolResultText(formatted), nil
}

// handleGetChannelCurrentValues handles prtg_get_channel_current_values tool requests.
func (h *MetricsToolHandler) handleGetChannelCurrentValues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SensorID int `json:"sensor_id"`
	}

	if err := parseArguments(request.Params.Arguments, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	h.handler.logger.Info().
		Int("sensor_id", params.SensorID).
		Msg("Fetching channel current values from PRTG API")

	// Fetch channels from PRTG API
	channels, err := h.prtgClient.GetChannelsBySensor(ctx, params.SensorID)
	if err != nil {
		h.handler.logger.Error().
			Err(err).
			Int("sensor_id", params.SensorID).
			Msg("Failed to fetch channels from PRTG API")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch channels: %v", err)), nil
	}

	if len(channels) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No channels found for sensor %d", params.SensorID)), nil
	}

	// Format response for LLM
	formatted := formatChannelsForLLM(params.SensorID, channels)

	return mcp.NewToolResultText(formatted), nil
}

// formatTimeSeriesForLLM formats time series data in a readable format for LLMs.
func formatTimeSeriesForLLM(data *prtg.TimeSeriesData) string {
	if len(data.DataPoints) == 0 {
		return fmt.Sprintf("No data available for sensor %d", data.ObjectID)
	}

	var output string

	// Header
	if data.TimeType != "" {
		output += fmt.Sprintf("# Time Series Data - Sensor %d (%s)\n\n", data.ObjectID, data.TimeType)
	} else {
		output += fmt.Sprintf("# Time Series Data - Sensor %d\n", data.ObjectID)
		if data.StartTime != nil && data.EndTime != nil {
			output += fmt.Sprintf("Period: %s to %s\n\n",
				data.StartTime.Format("2006-01-02 15:04:05"),
				data.EndTime.Format("2006-01-02 15:04:05"))
		}
	}

	// Summary
	output += fmt.Sprintf("Total data points: %d\n", len(data.DataPoints))
	output += fmt.Sprintf("Channels: %s\n\n", formatChannelNames(data.Headers))

	// Data table (show first 10 and last 5 if more than 15 points)
	output += "## Measurements\n\n"
	output += formatDataTable(data)

	return output
}

// formatChannelNames formats channel names from headers (skip first which is timestamp).
func formatChannelNames(headers []string) string {
	if len(headers) <= 1 {
		return "none"
	}

	channels := ""
	for i := 1; i < len(headers); i++ {
		if i > 1 {
			channels += ", "
		}
		channels += headers[i]
	}

	return channels
}

// formatDataTable formats the time series data as a markdown table.
func formatDataTable(data *prtg.TimeSeriesData) string {
	if len(data.DataPoints) == 0 {
		return "No data\n"
	}

	// Determine how many points to show
	showFirst := 10
	showLast := 5
	totalPoints := len(data.DataPoints)
	truncated := totalPoints > (showFirst + showLast)

	// Build header row
	table := "| Timestamp |"
	for i := 1; i < len(data.Headers); i++ {
		table += fmt.Sprintf(" %s |", data.Headers[i])
	}
	table += "\n"

	// Build separator row
	table += "|-----------|"
	for i := 1; i < len(data.Headers); i++ {
		table += "----------|"
	}
	table += "\n"

	// Build data rows
	pointsToShow := totalPoints
	if truncated {
		pointsToShow = showFirst
	}

	for i := 0; i < pointsToShow && i < totalPoints; i++ {
		point := data.DataPoints[i]
		table += fmt.Sprintf("| %s |", point.Timestamp.Format("2006-01-02 15:04:05"))

		for j := 1; j < len(data.Headers); j++ {
			channelName := data.Headers[j]
			value := point.Values[channelName]
			table += fmt.Sprintf(" %v |", formatValue(value))
		}
		table += "\n"
	}

	// Add "..." row if truncated
	if truncated {
		table += "| ... | "
		for i := 1; i < len(data.Headers); i++ {
			table += "... | "
		}
		table += "\n"

		// Add last N points
		for i := totalPoints - showLast; i < totalPoints; i++ {
			point := data.DataPoints[i]
			table += fmt.Sprintf("| %s |", point.Timestamp.Format("2006-01-02 15:04:05"))

			for j := 1; j < len(data.Headers); j++ {
				channelName := data.Headers[j]
				value := point.Values[channelName]
				table += fmt.Sprintf(" %v |", formatValue(value))
			}
			table += "\n"
		}
	}

	return table
}

// formatValue formats a channel value for display.
func formatValue(value interface{}) string {
	if value == nil {
		return "N/A"
	}

	switch v := value.(type) {
	case float64:
		// Format with 2 decimal places
		return fmt.Sprintf("%.2f", v)
	case int, int64:
		return fmt.Sprintf("%d", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatChannelsForLLM formats channel data in a readable format for LLMs.
func formatChannelsForLLM(sensorID int, channels []prtg.Channel) string {
	output := fmt.Sprintf("# Current Channel Values - Sensor %d\n\n", sensorID)
	output += fmt.Sprintf("Total channels: %d\n\n", len(channels))

	output += "| Channel | Value | Unit | Timestamp |\n"
	output += "|---------|-------|------|----------|\n"

	for _, ch := range channels {
		value := "N/A"
		timestamp := "-"
		unit := ch.Basic.DisplayUnit

		if ch.LastMeasurement != nil {
			value = fmt.Sprintf("%.2f", ch.LastMeasurement.DisplayValue)
			timestamp = ch.LastMeasurement.Timestamp
		}

		output += fmt.Sprintf("| %s | %s | %s | %s |\n",
			ch.Name,
			value,
			unit,
			timestamp)
	}

	return output
}
