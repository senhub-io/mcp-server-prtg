// Package handlers implements MCP (Model Context Protocol) tool handlers for PRTG monitoring data.
// It provides 6 MCP tools: sensor queries, alerts, device overview, top sensors, and custom SQL.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// Config is an interface for accessing configuration settings.
type Config interface {
	AllowCustomQueries() bool
}

// DatabaseQuerier is an interface for database operations.
// This interface allows mocking in tests while maintaining type safety.
type DatabaseQuerier interface {
	GetSensors(ctx context.Context, deviceName, sensorName string, status *int, tags string, limit int) ([]types.Sensor, error)
	GetSensorByID(ctx context.Context, sensorID int) (*types.Sensor, error)
	GetAlerts(ctx context.Context, hours int, status *int, deviceName string) ([]types.Sensor, error)
	GetDeviceOverview(ctx context.Context, deviceName string) (*types.DeviceOverview, error)
	GetTopSensors(ctx context.Context, metric, sensorType string, limit, hours int) ([]types.Sensor, error)
	ExecuteCustomQuery(ctx context.Context, query string, limit int) ([]map[string]interface{}, error)
}

// ToolHandler handles MCP tool requests and dispatches them to the database layer.
// Each tool request includes context, authentication, and parameter validation.
type ToolHandler struct {
	db     DatabaseQuerier
	config Config
	logger *zerolog.Logger
}

// NewToolHandler creates a new MCP tool handler with the given database, config, and logger.
func NewToolHandler(db DatabaseQuerier, config Config, logger *zerolog.Logger) *ToolHandler {
	return &ToolHandler{
		db:     db,
		config: config,
		logger: logger,
	}
}

// RegisterTools registers all 6 MCP tools with the server.
// Tools: prtg_get_sensors, prtg_get_sensor_status, prtg_get_alerts,
// prtg_device_overview, prtg_top_sensors, prtg_query_sql.
//
//nolint:funlen // Tool registration function must define all MCP tools with their complete schemas inline.
func (h *ToolHandler) RegisterTools(s *server.MCPServer) {
	// Tool 1: prtg_get_sensors
	s.AddTool(mcp.Tool{
		Name: "prtg_get_sensors",
		Description: "Retrieve PRTG sensors with optional filters (device name, sensor name, status, tags). " +
			"Returns current sensor status and metadata.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"device_name": map[string]string{
					"type":        "string",
					"description": "Filter by device name (partial match, case-insensitive)",
				},
				"sensor_name": map[string]string{
					"type":        "string",
					"description": "Filter by sensor name (partial match, case-insensitive)",
				},
				"status": map[string]interface{}{
					"type": "integer",
					"description": "Filter by status (1=Unknown, 2=Collecting, 3=Up, 4=Warning, 5=Down, 6=NoProbe, " +
						"7=PausedByUser, 8=PausedByDependency, 9=PausedBySchedule, 10=Unusual, " +
						"11=PausedByLicense, 12=PausedUntil, 13=DownAcknowledged, 14=DownPartial)",
				},
				"tags": map[string]string{
					"type":        "string",
					"description": "Filter by tag name (partial match)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 50)",
					"default":     50,
				},
			},
		},
	}, h.handleGetSensors)

	// Tool 2: prtg_get_sensor_status
	s.AddTool(mcp.Tool{
		Name: "prtg_get_sensor_status",
		Description: "Get detailed current status of a specific sensor by ID. " +
			"Returns current values, uptime, downtime, and status information.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "The sensor ID to query",
				},
			},
			Required: []string{"sensor_id"},
		},
	}, h.handleGetSensorStatus)

	// Tool 3: prtg_get_alerts
	s.AddTool(mcp.Tool{
		Name:        "prtg_get_alerts",
		Description: "Retrieve sensors in alert state (not Up). Returns sensors with warnings, errors, or down status.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"hours": map[string]interface{}{
					"type":        "integer",
					"description": "Only include alerts from the last N hours (0 = all)",
					"default":     24,
				},
				"status": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by specific status (4=Warning, 5=Down)",
				},
				"device_name": map[string]string{
					"type":        "string",
					"description": "Filter by device name",
				},
			},
		},
	}, h.handleGetAlerts)

	// Tool 4: prtg_device_overview
	s.AddTool(mcp.Tool{
		Name:        "prtg_device_overview",
		Description: "Get a complete overview of a device including all its sensors and statistics (up/down/warning counts).",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"device_name": map[string]string{
					"type":        "string",
					"description": "Device name to query (partial match)",
				},
			},
			Required: []string{"device_name"},
		},
	}, h.handleDeviceOverview)

	// Tool 5: prtg_top_sensors
	s.AddTool(mcp.Tool{
		Name:        "prtg_top_sensors",
		Description: "Get top sensors ranked by various metrics (uptime, downtime, or alerts).",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"metric": map[string]interface{}{
					"type":        "string",
					"description": "Metric to rank by: 'uptime', 'downtime', or 'alerts'",
					"enum":        []string{"uptime", "downtime", "alerts"},
					"default":     "downtime",
				},
				"sensor_type": map[string]string{
					"type":        "string",
					"description": "Filter by sensor type (e.g., 'ping', 'http')",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return (default: 10)",
					"default":     10,
				},
				"hours": map[string]interface{}{
					"type":        "integer",
					"description": "Time window in hours (default: 24)",
					"default":     24,
				},
			},
		},
	}, h.handleTopSensors)

	// Tool 6: prtg_query_sql
	s.AddTool(mcp.Tool{
		Name: "prtg_query_sql",
		Description: "Execute a custom SQL query on the PRTG database (SELECT only). " +
			"Use for advanced queries not covered by other tools.\n\n" +
			"IMPORTANT - Table Schema:\n" +
			"- prtg_sensor: id, name, sensor_type, prtg_device_id, status, priority, message, last_check_utc, full_path\n" +
			"- prtg_device: id, name\n" +
			"- prtg_sensor_path: sensor_id, path\n" +
			"- prtg_tag: id, name\n" +
			"- prtg_sensor_tag: prtg_sensor_id, prtg_tag_id\n\n" +
			"Use these EXACT table names in your queries. " +
			"Status codes: 1=Unknown, 2=Collecting, 3=Up, 4=Warning, 5=Down, 6=NoProbe, " +
			"7=PausedByUser, 8=PausedByDependency, 9=PausedBySchedule, 10=Unusual, " +
			"11=PausedByLicense, 12=PausedUntil, 13=DownAcknowledged, 14=DownPartial",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]string{
					"type":        "string",
					"description": "SQL SELECT query to execute (use table names: prtg_sensor, prtg_device, etc.)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 100)",
					"default":     100,
				},
			},
			Required: []string{"query"},
		},
	}, h.handleCustomQuery)
}

// handleGetSensors handles the prtg_get_sensors tool.
func (h *ToolHandler) handleGetSensors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_sensors")

	var args struct {
		DeviceName string `json:"device_name"`
		SensorName string `json:"sensor_name"`
		Status     *int   `json:"status"`
		Tags       string `json:"tags"`
		Limit      int    `json:"limit"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit <= 0 {
		args.Limit = 1000 // Default to reasonable limit, user can override
	}

	h.logger.Debug().
		Str("device_name", args.DeviceName).
		Str("sensor_name", args.SensorName).
		Interface("status", args.Status).
		Str("tags", args.Tags).
		Int("limit", args.Limit).
		Msg("calling db.GetSensors")

	// Add timeout to parent context (preserves cancellation chain)
	// This allows client cancellation while providing reasonable timeout for DB operations
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sensors, err := h.db.GetSensors(dbCtx, args.DeviceName, args.SensorName, args.Status, args.Tags, args.Limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("db.GetSensors failed")
		return nil, fmt.Errorf("failed to get sensors: %w", err)
	}

	h.logger.Debug().Int("count", len(sensors)).Msg("db.GetSensors returned")

	// Use visual formatting for sensors
	formattedText := formatSensorsResponse(sensors)

	h.logger.Info().
		Int("sensors_count", len(sensors)).
		Int("response_size_bytes", len(formattedText)).
		Msg("returning result to MCP client")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedText,
			},
		},
	}, nil
}

// handleGetSensorStatus handles the prtg_get_sensor_status tool.
func (h *ToolHandler) handleGetSensorStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_sensor_status")

	var args struct {
		SensorID int `json:"sensor_id"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.SensorID <= 0 {
		return nil, fmt.Errorf("sensor_id must be greater than 0")
	}

	// Add timeout to parent context (preserves cancellation chain)
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sensor, err := h.db.GetSensorByID(dbCtx, args.SensorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor: %w", err)
	}

	return formatResult(sensor, 1)
}

// handleGetAlerts handles the prtg_get_alerts tool.
func (h *ToolHandler) handleGetAlerts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_alerts")

	var args struct {
		Hours      int    `json:"hours"`
		Status     *int   `json:"status"`
		DeviceName string `json:"device_name"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Hours == 0 {
		args.Hours = 24
	}

	// Add timeout to parent context (preserves cancellation chain)
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sensors, err := h.db.GetAlerts(dbCtx, args.Hours, args.Status, args.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}

	// Use visual formatting for alerts
	formattedText := formatAlertsResponse(sensors)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedText,
			},
		},
	}, nil
}

// handleDeviceOverview handles the prtg_device_overview tool.
func (h *ToolHandler) handleDeviceOverview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_device_overview")

	var args struct {
		DeviceName string `json:"device_name"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.DeviceName == "" {
		return nil, fmt.Errorf("device_name is required")
	}

	// Add timeout to parent context (preserves cancellation chain)
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	overview, err := h.db.GetDeviceOverview(dbCtx, args.DeviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get device overview: %w", err)
	}

	// Use visual formatting for device overview
	formattedText := formatDeviceOverviewResponse(overview)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedText,
			},
		},
	}, nil
}

// handleTopSensors handles the prtg_top_sensors tool.
func (h *ToolHandler) handleTopSensors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_top_sensors")

	var args struct {
		Metric     string `json:"metric"`
		SensorType string `json:"sensor_type"`
		Limit      int    `json:"limit"`
		Hours      int    `json:"hours"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Metric == "" {
		args.Metric = "downtime"
	}

	if args.Limit <= 0 {
		args.Limit = 10
	}

	if args.Hours <= 0 {
		args.Hours = 24
	}

	// Add timeout to parent context (preserves cancellation chain)
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	sensors, err := h.db.GetTopSensors(dbCtx, args.Metric, args.SensorType, args.Limit, args.Hours)
	if err != nil {
		return nil, fmt.Errorf("failed to get top sensors: %w", err)
	}

	// Use visual formatting for top sensors
	formattedText := formatTopSensorsResponse(sensors, args.Metric)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: formattedText,
			},
		},
	}, nil
}

// handleCustomQuery handles the prtg_query_sql tool.
func (h *ToolHandler) handleCustomQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_query_sql")

	// SECURITY: Check if custom queries are allowed (disabled by default for security)
	if !h.config.AllowCustomQueries() {
		h.logger.Warn().Msg("Custom SQL queries are disabled in configuration (allow_custom_queries: false)")

		return nil, fmt.Errorf(
			"custom SQL queries are disabled for security reasons - " +
				"set 'allow_custom_queries: true' in config.yaml to enable (not recommended in production)")
	}

	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := parseArguments(request.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if args.Limit <= 0 {
		args.Limit = 100
	}

	h.logger.Debug().
		Str("query", args.Query).
		Int("limit", args.Limit).
		Msg("calling db.ExecuteCustomQuery")

	// Add timeout to parent context (preserves cancellation chain)
	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	results, err := h.db.ExecuteCustomQuery(dbCtx, args.Query, args.Limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("db.ExecuteCustomQuery failed")
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	h.logger.Debug().Int("result_count", len(results)).Msg("db.ExecuteCustomQuery returned")

	return formatResult(results, len(results))
}

// parseArguments parses tool arguments from interface{} to target struct.
func parseArguments(args, target interface{}) error {
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// formatResult formats the response data as MCP tool result.
func formatResult(data interface{}, count int) (*mcp.CallToolResult, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Found %d result(s):\n\n%s", count, string(jsonData)),
			},
		},
	}, nil
}
