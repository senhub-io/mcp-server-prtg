package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	mcptool "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/senhub-io/mcp-server-prtg/internal/prtg"
)

// APIv2ToolHandler handles all MCP tools that use PRTG API v2.
// These tools work without database and can be used in stdio mode.
type APIv2ToolHandler struct {
	prtgClient  PRTGClient
	mainHandler *ToolHandler
}

// NewAPIv2ToolHandler creates a new API v2 tool handler.
func NewAPIv2ToolHandler(prtgClient PRTGClient, mainHandler *ToolHandler) *APIv2ToolHandler {
	return &APIv2ToolHandler{
		prtgClient:  prtgClient,
		mainHandler: mainHandler,
	}
}

// RegisterAPIv2Tools registers all API v2-based tools with the MCP server.
func (h *APIv2ToolHandler) RegisterAPIv2Tools(server *mcpserver.MCPServer) {
	// Register all 11 database-free tools (prtg_query_sql removed by design)
	server.AddTool(h.toolGetSensors(), h.handleGetSensors)
	server.AddTool(h.toolGetSensorStatus(), h.handleGetSensorStatus)
	server.AddTool(h.toolGetAlerts(), h.handleGetAlerts)
	server.AddTool(h.toolDeviceOverview(), h.handleDeviceOverview)
	server.AddTool(h.toolGetHierarchy(), h.handleGetHierarchy)
	server.AddTool(h.toolSearch(), h.handleSearch)
	server.AddTool(h.toolGetGroups(), h.handleGetGroups)
	server.AddTool(h.toolGetTags(), h.handleGetTags)
	server.AddTool(h.toolGetBusinessProcesses(), h.handleGetBusinessProcesses)
	server.AddTool(h.toolTopSensors(), h.handleTopSensors)
	server.AddTool(h.toolGetStatistics(), h.handleGetStatistics)
}

// toolGetSensors returns the prtg_get_sensors tool definition.
func (h *APIv2ToolHandler) toolGetSensors() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_sensors",
		Description: "Retrieve PRTG sensors with optional filters (device, sensor name, type, group, status, tags). Returns current sensor status and metadata. Supports ordering by various fields.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_name": map[string]string{
					"type":        "string",
					"description": "Filter by sensor name (partial match, case-insensitive)",
				},
				"device_name": map[string]string{
					"type":        "string",
					"description": "Filter by device name (partial match, case-insensitive)",
				},
				"sensor_type": map[string]string{
					"type":        "string",
					"description": "Filter by sensor type (e.g., 'ping', 'http', 'snmp')",
				},
				"group_name": map[string]string{
					"type":        "string",
					"description": "Filter by group name (partial match, case-insensitive)",
				},
				"status": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by status (1=Unknown, 2=Collecting, 3=Up, 4=Warning, 5=Down, 6=NoProbe, 7=PausedByUser, 8=PausedByDependency, 9=PausedBySchedule, 10=Unusual, 11=PausedByLicense, 12=PausedUntil, 13=DownAcknowledged, 14=DownPartial)",
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
				"order_by": map[string]interface{}{
					"type":        "string",
					"description": "Order results by field: 'name' (default), 'status', 'priority', 'device', 'type', 'last_check'",
					"enum":        []string{"name", "status", "priority", "device", "type", "last_check"},
					"default":     "name",
				},
			},
		},
	}
}

// handleGetSensors handles the prtg_get_sensors tool call.
func (h *APIv2ToolHandler) handleGetSensors(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_sensors")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	// Collect filter expressions to avoid map key collision
	// Multiple filter expressions are passed as separate query parameters
	var filterExpressions []string

	if sensorName, ok := args["sensor_name"].(string); ok && sensorName != "" {
		filterExpressions = append(filterExpressions, "name contains "+prtg.FilterText(sensorName))
	}

	if deviceName, ok := args["device_name"].(string); ok && deviceName != "" {
		// Filter by parent device name
		filterExpressions = append(filterExpressions, "parent.name contains "+prtg.FilterText(deviceName))
	}

	if groupName, ok := args["group_name"].(string); ok && groupName != "" {
		// Filter by group - PRTG sensors have a group field
		filterExpressions = append(filterExpressions, "group contains "+prtg.FilterText(groupName))
	}

	if tags, ok := args["tags"].(string); ok && tags != "" {
		filterExpressions = append(filterExpressions, "tags contains "+prtg.FilterText(tags))
	}

	// Pass filter expressions as a slice - buildQueryParams will create multiple filter parameters
	if len(filterExpressions) > 0 {
		filters["filter"] = filterExpressions
	}

	// Handle special filter parameters
	if status, ok := args["status"].(float64); ok {
		filters["filter_status"] = int(status)
	}

	if sensorType, ok := args["sensor_type"].(string); ok && sensorType != "" {
		filters["filter_type"] = sensorType
	}

	// Handle limit parameter
	if limit, ok := args["limit"].(float64); ok {
		filters["limit"] = int(limit)
	} else {
		filters["limit"] = 50
	}

	// Handle order_by parameter
	if orderBy, ok := args["order_by"].(string); ok && orderBy != "" {
		// Map user-friendly field names to PRTG API field names
		sortField := orderBy
		switch orderBy {
		case "last_check":
			sortField = "lastcheck"
		case "device":
			sortField = "parent.name"
		}
		filters["sort"] = sortField
	}

	// Call API
	sensors, err := h.prtgClient.GetSensors(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors: %w", err)
	}

	// Format results
	result := formatSensorsResult(sensors)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetSensorStatus returns the prtg_get_sensor_status tool definition.
func (h *APIv2ToolHandler) toolGetSensorStatus() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_sensor_status",
		Description: "Get detailed current status of a specific sensor by ID. Returns current values, uptime, downtime, and status information.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "The sensor ID to query",
				},
			},
			Required: []string{"sensor_id"},
		},
	}
}

// handleGetSensorStatus handles the prtg_get_sensor_status tool call.
func (h *APIv2ToolHandler) handleGetSensorStatus(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_sensor_status")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	// Parse sensor ID
	sensorIDFloat, ok := args["sensor_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("sensor_id is required and must be an integer")
	}
	sensorID := int(sensorIDFloat)

	// Get sensor
	sensor, err := h.prtgClient.GetSensorByID(ctx, sensorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensor: %w", err)
	}

	// Format result
	result := formatSensorStatusResult(sensor)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetAlerts returns the prtg_get_alerts tool definition.
func (h *APIv2ToolHandler) toolGetAlerts() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_alerts",
		Description: "Retrieve sensors in alert state (not Up). Returns sensors with warnings, errors, or down status.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"device_name": map[string]string{
					"type":        "string",
					"description": "Filter by device name",
				},
				"status": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by specific status (4=Warning, 5=Down)",
				},
				"hours": map[string]interface{}{
					"type":        "integer",
					"description": "Only include alerts from the last N hours (0 = all)",
					"default":     24,
				},
			},
		},
	}
}

// handleGetAlerts handles the prtg_get_alerts tool call.
func (h *APIv2ToolHandler) handleGetAlerts(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_alerts")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	// Filter for non-Up status
	if status, ok := args["status"].(float64); ok {
		// Use filter_status which will be converted to status=warning format by buildQueryParams
		filters["filter_status"] = int(status)
	} else {
		// Default: get all non-Up sensors (Warning, Down, Unknown, Unusual, etc.)
		// PRTG API v2 filter syntax: status != up (lowercase enumeration token)
		filters["filter"] = "status != up"
	}

	if deviceName, ok := args["device_name"].(string); ok && deviceName != "" {
		filters["filter"] = "parent.name contains " + prtg.FilterText(deviceName)
	}

	filters["limit"] = 100

	// Get sensors
	sensors, err := h.prtgClient.GetSensors(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}

	// Format result
	result := formatAlertsResult(sensors)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolDeviceOverview returns the prtg_device_overview tool definition.
func (h *APIv2ToolHandler) toolDeviceOverview() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_device_overview",
		Description: "Get a complete overview of a device including all its sensors and statistics (up/down/warning counts).",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"device_name": map[string]interface{}{
					"type":        "string",
					"description": "Device name to query (partial match)",
				},
			},
			Required: []string{"device_name"},
		},
	}
}

// handleDeviceOverview handles the prtg_device_overview tool call.
func (h *APIv2ToolHandler) handleDeviceOverview(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_device_overview")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	deviceName, ok := args["device_name"].(string)
	if !ok || deviceName == "" {
		return nil, fmt.Errorf("device_name is required")
	}

	// Search for devices by name
	filters := map[string]interface{}{
		"filter": "name contains " + prtg.FilterText(deviceName),
		"limit":  10,
	}

	devices, err := h.prtgClient.GetObjects(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}

	// Filter for device type
	var deviceObjects []prtg.PRTGObject
	for _, obj := range devices {
		if obj.Type == "device" {
			deviceObjects = append(deviceObjects, obj)
		}
	}

	if len(deviceObjects) == 0 {
		return &mcptool.CallToolResult{
			Content: []mcptool.Content{
				mcptool.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No device found matching '%s'", deviceName),
				},
			},
		}, nil
	}

	// Get first device details
	deviceID, _ := strconv.Atoi(deviceObjects[0].ID)
	device, err := h.prtgClient.GetDeviceByID(ctx, deviceID, "sensor_status_summary")
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Get sensors for this device
	sensorFilters := map[string]interface{}{
		"filter_objid": deviceID,
		"limit":        100,
	}
	sensors, err := h.prtgClient.GetSensors(ctx, sensorFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get device sensors: %w", err)
	}

	// Format result
	result := formatDeviceOverviewResult(device, sensors)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetHierarchy returns the prtg_get_hierarchy tool definition.
func (h *APIv2ToolHandler) toolGetHierarchy() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_hierarchy",
		Description: "Navigate the PRTG hierarchy tree structure. Returns groups, devices, and optionally sensors in a tree format. Useful for understanding the organization and structure of your PRTG installation.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"group_name": map[string]string{
					"type":        "string",
					"description": "Starting group name (leave empty for root groups)",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum depth to traverse (0 = unlimited, default: 2)",
					"default":     2,
				},
				"include_sensors": map[string]interface{}{
					"type":        "boolean",
					"description": "Include sensors in the output (default: false)",
					"default":     false,
				},
			},
		},
	}
}

// handleGetHierarchy handles the prtg_get_hierarchy tool call.
func (h *APIv2ToolHandler) handleGetHierarchy(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_hierarchy")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	if groupName, ok := args["group_name"].(string); ok && groupName != "" {
		filters["filter"] = "name contains " + prtg.FilterText(groupName)
	}

	filters["limit"] = 100

	// Get objects
	objects, err := h.prtgClient.GetObjects(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy: %w", err)
	}

	// Format result
	result := formatHierarchyResult(objects)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolSearch returns the prtg_search tool definition.
func (h *APIv2ToolHandler) toolSearch() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_search",
		Description: "Universal search across groups, devices, and sensors. Searches by name, host, or sensor type. Returns all matching results organized by type.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"search_term": map[string]interface{}{
					"type":        "string",
					"description": "Search term to find (case-insensitive, partial match)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results per category (default: 50)",
					"default":     50,
				},
			},
			Required: []string{"search_term"},
		},
	}
}

// handleSearch handles the prtg_search tool call.
func (h *APIv2ToolHandler) handleSearch(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_search")

	// Parse arguments
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	searchTerm, ok := args["search_term"].(string)
	if !ok || searchTerm == "" {
		return nil, fmt.Errorf("search_term is required")
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	filters := map[string]interface{}{
		"filter": "name contains " + prtg.FilterText(searchTerm),
		"limit":  limit,
	}

	// Search all objects
	objects, err := h.prtgClient.GetObjects(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Format result
	result := formatSearchResult(objects, searchTerm)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetGroups returns the prtg_get_groups tool definition.
func (h *APIv2ToolHandler) toolGetGroups() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_groups",
		Description: "List PRTG groups/probes with optional filtering. Groups organize devices in a hierarchical structure. Returns group information including paths and probe status.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"group_name": map[string]string{
					"type":        "string",
					"description": "Filter by group name (partial match, case-insensitive)",
				},
				"parent_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by parent group ID (shows direct children)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 100)",
					"default":     100,
				},
			},
		},
	}
}

// handleGetGroups handles the prtg_get_groups tool call.
func (h *APIv2ToolHandler) handleGetGroups(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_groups")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	if groupName, ok := args["group_name"].(string); ok && groupName != "" {
		filters["filter"] = "name contains " + prtg.FilterText(groupName)
	}

	if parentID, ok := args["parent_id"].(float64); ok {
		filters["filter_parentId"] = int(parentID)
	}

	if limit, ok := args["limit"].(float64); ok {
		filters["limit"] = int(limit)
	} else {
		filters["limit"] = 100
	}

	// Get groups
	groups, err := h.prtgClient.GetGroups(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	// Format result
	result := formatGroupsResult(groups)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetTags returns the prtg_get_tags tool definition.
func (h *APIv2ToolHandler) toolGetTags() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_tags",
		Description: "List PRTG tags with usage statistics. Tags are labels applied to sensors for organization and filtering. Returns tag names and sensor counts.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"tag_name": map[string]string{
					"type":        "string",
					"description": "Filter by tag name (partial match, case-insensitive)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 100)",
					"default":     100,
				},
			},
		},
	}
}

// handleGetTags handles the prtg_get_tags tool call.
func (h *APIv2ToolHandler) handleGetTags(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_tags")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})
	filters["limit"] = 1000

	// Get all sensors to extract tags
	sensors, err := h.prtgClient.GetSensors(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors for tags: %w", err)
	}

	// Extract and count tags
	tagCounts := make(map[string]int)
	for _, sensor := range sensors {
		for _, tag := range sensor.Tags {
			tagCounts[tag]++
		}
	}

	// Filter by tag name if provided
	tagName, _ := args["tag_name"].(string)
	if tagName != "" {
		filteredTags := make(map[string]int)
		for tag, count := range tagCounts {
			if strings.Contains(strings.ToLower(tag), strings.ToLower(tagName)) {
				filteredTags[tag] = count
			}
		}
		tagCounts = filteredTags
	}

	// Format result
	result := formatTagsResult(tagCounts)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetBusinessProcesses returns the prtg_get_business_processes tool definition.
func (h *APIv2ToolHandler) toolGetBusinessProcesses() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_business_processes",
		Description: "List PRTG Business Process sensors with optional filtering. Business Process sensors aggregate the status of multiple source sensors to monitor complete business workflows. Returns sensor information including current status, priority, and associated metadata.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"process_name": map[string]string{
					"type":        "string",
					"description": "Filter by process name (partial match, case-insensitive)",
				},
				"status": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by status (1=Unknown, 2=Collecting, 3=Up, 4=Warning, 5=Down, 6=NoProbe, 7=PausedByUser, 8=PausedByDependency, 9=PausedBySchedule, 10=Unusual, 11=PausedByLicense, 12=PausedUntil, 13=DownAcknowledged, 14=DownPartial)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 100)",
					"default":     100,
				},
			},
		},
	}
}

// handleGetBusinessProcesses handles the prtg_get_business_processes tool call.
func (h *APIv2ToolHandler) handleGetBusinessProcesses(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_business_processes")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	// Filter for business process sensor kind
	// Note: Business Process sensors use kind "businessprocess", not type
	filters["filter"] = "kind contains \"businessprocess\""

	if processName, ok := args["process_name"].(string); ok && processName != "" {
		filters["filter"] = "name contains " + prtg.FilterText(processName)
	}

	if status, ok := args["status"].(float64); ok {
		filters["filter_status"] = int(status)
	}

	if limit, ok := args["limit"].(float64); ok {
		filters["limit"] = int(limit)
	} else {
		filters["limit"] = 100
	}

	// Get sensors
	sensors, err := h.prtgClient.GetSensors(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get business processes: %w", err)
	}

	// Format result
	result := formatBusinessProcessesResult(sensors)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolTopSensors returns the prtg_top_sensors tool definition.
func (h *APIv2ToolHandler) toolTopSensors() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_top_sensors",
		Description: "Get top sensors ranked by various metrics (uptime, downtime, or alerts).",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"metric": map[string]interface{}{
					"type":        "string",
					"description": "Metric to rank by: 'uptime', 'downtime', or 'alerts'",
					"enum":        []string{"uptime", "downtime", "alerts"},
					"default":     "downtime",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return (default: 10)",
					"default":     10,
				},
				"sensor_type": map[string]string{
					"type":        "string",
					"description": "Filter by sensor type (e.g., 'ping', 'http')",
				},
				"hours": map[string]interface{}{
					"type":        "integer",
					"description": "Time window in hours (default: 24)",
					"default":     24,
				},
			},
		},
	}
}

// handleTopSensors handles the prtg_top_sensors tool call.
func (h *APIv2ToolHandler) handleTopSensors(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_top_sensors")

	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	filters := make(map[string]interface{})

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	filters["limit"] = limit * 3 // Get more to sort locally

	if sensorType, ok := args["sensor_type"].(string); ok && sensorType != "" {
		filters["filter_type"] = sensorType
	}

	// Get sensors
	sensors, err := h.prtgClient.GetSensors(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors: %w", err)
	}

	metric := "downtime"
	if m, ok := args["metric"].(string); ok {
		metric = m
	}

	// Format result
	result := formatTopSensorsResult(sensors, metric, limit)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// toolGetStatistics returns the prtg_get_statistics tool definition.
func (h *APIv2ToolHandler) toolGetStatistics() mcptool.Tool {
	return mcptool.Tool{
		Name:        "prtg_get_statistics",
		Description: "Get aggregated PRTG server statistics including total counts, status breakdown, and sensor type distribution. Provides a comprehensive overview of your PRTG installation's health and composition.",
		InputSchema: mcptool.ToolInputSchema{
			Type: "object",
		},
	}
}

// handleGetStatistics handles the prtg_get_statistics tool call.
func (h *APIv2ToolHandler) handleGetStatistics(ctx context.Context, request mcptool.CallToolRequest) (*mcptool.CallToolResult, error) {
	h.mainHandler.logger.Info().Interface("arguments", request.Params.Arguments).Msg("handling prtg_get_statistics")

	// Get status summary
	summary, err := h.prtgClient.GetStatusSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status summary: %w", err)
	}

	// Get object counts
	counts, err := h.prtgClient.GetObjectCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object counts: %w", err)
	}

	// Format result
	result := formatStatisticsResult(summary, counts)

	return &mcptool.CallToolResult{
		Content: []mcptool.Content{
			mcptool.TextContent{
				Type: "text",
				Text: result,
			},
		},
	}, nil
}

// Formatting helper functions

func formatSensorsResult(sensors []prtg.Sensor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PRTG Sensors (%d found)\n\n", len(sensors)))

	for i, sensor := range sensors {
		if i >= 50 { // Limit output
			sb.WriteString(fmt.Sprintf("\n... and %d more sensors (use limit parameter to see more)\n", len(sensors)-50))
			break
		}

		sb.WriteString(fmt.Sprintf("## %s (ID: %s)\n", sensor.Name, sensor.ID))
		sb.WriteString(fmt.Sprintf("- **Type**: %s\n", sensor.Type))
		sb.WriteString(fmt.Sprintf("- **Status**: %s (%d)\n", sensor.Status.Name, sensor.Status.Code))
		if sensor.Priority > 0 {
			sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", sensor.Priority))
		}
		if sensor.Message != "" {
			sb.WriteString(fmt.Sprintf("- **Message**: %s\n", sensor.Message))
		}
		if sensor.Parent != nil {
			sb.WriteString(fmt.Sprintf("- **Parent**: %s (%s)\n", sensor.Parent.Name, sensor.Parent.Type))
		}
		if len(sensor.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(sensor.Tags, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatSensorStatusResult(sensor *prtg.Sensor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Sensor: %s (ID: %s)\n\n", sensor.Name, sensor.ID))
	sb.WriteString(fmt.Sprintf("## Status\n"))
	sb.WriteString(fmt.Sprintf("- **Current Status**: %s (%d)\n", sensor.Status.Name, sensor.Status.Code))
	sb.WriteString(fmt.Sprintf("- **Since**: %s\n", sensor.Status.Since))
	if sensor.Message != "" {
		sb.WriteString(fmt.Sprintf("- **Message**: %s\n", sensor.Message))
	}

	sb.WriteString(fmt.Sprintf("\n## Details\n"))
	sb.WriteString(fmt.Sprintf("- **Type**: %s\n", sensor.Type))
	if sensor.Priority > 0 {
		sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", sensor.Priority))
	}
	if sensor.Parent != nil {
		sb.WriteString(fmt.Sprintf("- **Parent Device**: %s (ID: %s)\n", sensor.Parent.Name, sensor.Parent.ID))
	}
	if len(sensor.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("- **Tags**: %s\n", strings.Join(sensor.Tags, ", ")))
	}

	return sb.String()
}

func formatAlertsResult(sensors []prtg.Sensor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Active Alerts (%d found)\n\n", len(sensors)))

	if len(sensors) == 0 {
		sb.WriteString("No active alerts found. All sensors are Up.\n")
		return sb.String()
	}

	// Group by status
	byStatus := make(map[string][]prtg.Sensor)
	for _, sensor := range sensors {
		byStatus[sensor.Status.Name] = append(byStatus[sensor.Status.Name], sensor)
	}

	// Show Down first, then Warning
	for _, statusName := range []string{"Down", "Warning", "Unusual", "Unknown"} {
		sensors := byStatus[statusName]
		if len(sensors) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s (%d)\n\n", statusName, len(sensors)))
		for _, sensor := range sensors {
			sb.WriteString(fmt.Sprintf("- **%s** (ID: %s)\n", sensor.Name, sensor.ID))
			if sensor.Parent != nil {
				sb.WriteString(fmt.Sprintf("  - Device: %s\n", sensor.Parent.Name))
			}
			if sensor.Message != "" {
				sb.WriteString(fmt.Sprintf("  - Message: %s\n", sensor.Message))
			}
			sb.WriteString(fmt.Sprintf("  - Since: %s\n", sensor.Status.Since))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatDeviceOverviewResult(device *prtg.Device, sensors []prtg.Sensor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Device: %s (ID: %s)\n\n", device.Name, device.ID))

	sb.WriteString(fmt.Sprintf("## Device Status\n"))
	sb.WriteString(fmt.Sprintf("- **Status**: %s (%d)\n", device.Status.Name, device.Status.Code))
	if device.Host != "" {
		sb.WriteString(fmt.Sprintf("- **Host**: %s\n", device.Host))
	}
	if device.Parent != nil {
		sb.WriteString(fmt.Sprintf("- **Group**: %s\n", device.Parent.Name))
	}

	if device.Summary != nil {
		sb.WriteString(fmt.Sprintf("\n## Sensor Summary\n"))
		sb.WriteString(fmt.Sprintf("- **Total Sensors**: %d\n", device.Summary.Total))
		if device.Summary.Up > 0 {
			sb.WriteString(fmt.Sprintf("- ✅ **Up**: %d\n", device.Summary.Up))
		}
		if device.Summary.Warning > 0 {
			sb.WriteString(fmt.Sprintf("- ⚠️ **Warning**: %d\n", device.Summary.Warning))
		}
		if device.Summary.Down > 0 {
			sb.WriteString(fmt.Sprintf("- ❌ **Down**: %d\n", device.Summary.Down))
		}
		if device.Summary.Paused > 0 {
			sb.WriteString(fmt.Sprintf("- ⏸️ **Paused**: %d\n", device.Summary.Paused))
		}
	}

	sb.WriteString(fmt.Sprintf("\n## Sensors (%d)\n\n", len(sensors)))
	for i, sensor := range sensors {
		if i >= 20 { // Limit output
			sb.WriteString(fmt.Sprintf("\n... and %d more sensors\n", len(sensors)-20))
			break
		}

		statusIcon := "✅"
		if sensor.Status.Code == 5 {
			statusIcon = "❌"
		} else if sensor.Status.Code == 4 {
			statusIcon = "⚠️"
		}

		sb.WriteString(fmt.Sprintf("- %s **%s** (ID: %s) - %s\n", statusIcon, sensor.Name, sensor.ID, sensor.Type))
	}

	return sb.String()
}

func formatHierarchyResult(objects []prtg.PRTGObject) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PRTG Hierarchy (%d objects)\n\n", len(objects)))

	// Group by type
	byType := make(map[string][]prtg.PRTGObject)
	for _, obj := range objects {
		byType[obj.Type] = append(byType[obj.Type], obj)
	}

	// Show in order: probes, groups, devices, sensors
	for _, objType := range []string{"probe", "group", "device", "sensor"} {
		objs := byType[objType]
		if len(objs) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %ss (%d)\n\n", strings.Title(objType), len(objs)))
		for _, obj := range objs {
			sb.WriteString(fmt.Sprintf("- **%s** (ID: %s)\n", obj.Name, obj.ID))
			if obj.Parent != nil {
				sb.WriteString(fmt.Sprintf("  - Parent: %s (%s)\n", obj.Parent.Name, obj.Parent.Type))
			}
			if obj.Status != nil {
				sb.WriteString(fmt.Sprintf("  - Status: %s\n", obj.Status.Name))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatSearchResult(objects []prtg.PRTGObject, searchTerm string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Search Results for '%s' (%d found)\n\n", searchTerm, len(objects)))

	if len(objects) == 0 {
		sb.WriteString("No results found.\n")
		return sb.String()
	}

	// Group by type
	byType := make(map[string][]prtg.PRTGObject)
	for _, obj := range objects {
		byType[obj.Type] = append(byType[obj.Type], obj)
	}

	// Show results by type
	for _, objType := range []string{"probe", "group", "device", "sensor"} {
		objs := byType[objType]
		if len(objs) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %ss (%d)\n\n", strings.Title(objType), len(objs)))
		for i, obj := range objs {
			if i >= 10 { // Limit per type
				sb.WriteString(fmt.Sprintf("... and %d more %ss\n\n", len(objs)-10, objType))
				break
			}

			sb.WriteString(fmt.Sprintf("- **%s** (ID: %s)\n", obj.Name, obj.ID))
			if obj.Parent != nil {
				sb.WriteString(fmt.Sprintf("  - Parent: %s\n", obj.Parent.Name))
			}
			if obj.Status != nil {
				sb.WriteString(fmt.Sprintf("  - Status: %s\n", obj.Status.Name))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatGroupsResult(groups []prtg.Group) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PRTG Groups (%d found)\n\n", len(groups)))

	for _, group := range groups {
		sb.WriteString(fmt.Sprintf("## %s (ID: %s)\n", group.Name, group.ID))
		if group.Parent != nil {
			sb.WriteString(fmt.Sprintf("- **Parent**: %s (ID: %s)\n", group.Parent.Name, group.Parent.ID))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatTagsResult(tagCounts map[string]int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PRTG Tags (%d found)\n\n", len(tagCounts)))

	if len(tagCounts) == 0 {
		sb.WriteString("No tags found.\n")
		return sb.String()
	}

	// Convert to slice for sorting
	type tagCount struct {
		name  string
		count int
	}
	var tags []tagCount
	for name, count := range tagCounts {
		tags = append(tags, tagCount{name, count})
	}

	// Sort by count descending
	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			if tags[j].count > tags[i].count {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}

	sb.WriteString("| Tag | Sensor Count |\n")
	sb.WriteString("|-----|-------------|\n")
	for _, tag := range tags {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", tag.name, tag.count))
	}

	return sb.String()
}

func formatBusinessProcessesResult(sensors []prtg.Sensor) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Business Process Sensors (%d found)\n\n", len(sensors)))

	for _, sensor := range sensors {
		statusIcon := "✅"
		if sensor.Status.Code == 5 {
			statusIcon = "❌"
		} else if sensor.Status.Code == 4 {
			statusIcon = "⚠️"
		}

		sb.WriteString(fmt.Sprintf("## %s %s (ID: %s)\n", statusIcon, sensor.Name, sensor.ID))
		sb.WriteString(fmt.Sprintf("- **Status**: %s (%d)\n", sensor.Status.Name, sensor.Status.Code))
		if sensor.Priority > 0 {
			sb.WriteString(fmt.Sprintf("- **Priority**: %d\n", sensor.Priority))
		}
		if sensor.Parent != nil {
			sb.WriteString(fmt.Sprintf("- **Parent**: %s\n", sensor.Parent.Name))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatTopSensorsResult(sensors []prtg.Sensor, metric string, limit int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Top %d Sensors by %s\n\n", limit, metric))

	// Limit to requested number
	if len(sensors) > limit {
		sensors = sensors[:limit]
	}

	for i, sensor := range sensors {
		sb.WriteString(fmt.Sprintf("%d. **%s** (ID: %s)\n", i+1, sensor.Name, sensor.ID))
		sb.WriteString(fmt.Sprintf("   - Type: %s\n", sensor.Type))
		sb.WriteString(fmt.Sprintf("   - Status: %s\n", sensor.Status.Name))
		if sensor.Parent != nil {
			sb.WriteString(fmt.Sprintf("   - Device: %s\n", sensor.Parent.Name))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatStatisticsResult(summary *prtg.StatusSummary, counts *prtg.SensorCount) string {
	var sb strings.Builder
	sb.WriteString("# PRTG Server Statistics\n\n")

	sb.WriteString("## Object Counts\n")
	sb.WriteString(fmt.Sprintf("- **Total Objects**: %d\n", counts.Total))
	if counts.Sensors > 0 {
		sb.WriteString(fmt.Sprintf("- **Sensors**: %d\n", counts.Sensors))
	}
	if counts.Devices > 0 {
		sb.WriteString(fmt.Sprintf("- **Devices**: %d\n", counts.Devices))
	}
	if counts.Groups > 0 {
		sb.WriteString(fmt.Sprintf("- **Groups**: %d\n", counts.Groups))
	}
	if counts.Probes > 0 {
		sb.WriteString(fmt.Sprintf("- **Probes**: %d\n", counts.Probes))
	}

	sb.WriteString("\n## Sensor Status Summary\n")
	sb.WriteString(fmt.Sprintf("- **Total**: %d\n", summary.Total))
	if summary.Up > 0 {
		percentage := float64(summary.Up) / float64(summary.Total) * 100
		sb.WriteString(fmt.Sprintf("- ✅ **Up**: %d (%.1f%%)\n", summary.Up, percentage))
	}
	if summary.Warning > 0 {
		percentage := float64(summary.Warning) / float64(summary.Total) * 100
		sb.WriteString(fmt.Sprintf("- ⚠️ **Warning**: %d (%.1f%%)\n", summary.Warning, percentage))
	}
	if summary.Down > 0 {
		percentage := float64(summary.Down) / float64(summary.Total) * 100
		sb.WriteString(fmt.Sprintf("- ❌ **Down**: %d (%.1f%%)\n", summary.Down, percentage))
	}
	if summary.Paused > 0 {
		sb.WriteString(fmt.Sprintf("- ⏸️ **Paused**: %d\n", summary.Paused))
	}
	if summary.Unusual > 0 {
		sb.WriteString(fmt.Sprintf("- ⚡ **Unusual**: %d\n", summary.Unusual))
	}
	if summary.Unknown > 0 {
		sb.WriteString(fmt.Sprintf("- ❓ **Unknown**: %d\n", summary.Unknown))
	}

	// Calculate health score
	if summary.Total > 0 {
		healthScore := float64(summary.Up) / float64(summary.Total) * 100
		sb.WriteString(fmt.Sprintf("\n## Health Score: %.1f%%\n", healthScore))
		if healthScore >= 95 {
			sb.WriteString("Status: 🟢 Excellent\n")
		} else if healthScore >= 80 {
			sb.WriteString("Status: 🟡 Good\n")
		} else if healthScore >= 60 {
			sb.WriteString("Status: 🟠 Warning\n")
		} else {
			sb.WriteString("Status: 🔴 Critical\n")
		}
	}

	return sb.String()
}

// MarshalJSON implements custom JSON marshaling for PRTGClient interface.
func (c *APIv2ToolHandler) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
	}{
		Type: "APIv2ToolHandler",
	})
}
