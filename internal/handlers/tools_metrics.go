// Package handlers implements MCP tools for querying PRTG historical metrics via API v2.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/prtg"
	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// PRTGClient interface for PRTG API operations.
type PRTGClient interface {
	// Metrics operations
	GetTimeSeries(ctx context.Context, objectID int, timeType prtg.TimeSeriesType) (*prtg.TimeSeriesData, error)
	GetChannelsBySensor(ctx context.Context, sensorID int) ([]prtg.Channel, error)

	// API v2 operations
	GetSensors(ctx context.Context, filters map[string]interface{}) ([]prtg.Sensor, error)
	GetSensorByID(ctx context.Context, sensorID int) (*prtg.Sensor, error)
	GetDeviceByID(ctx context.Context, deviceID int, includes ...string) (*prtg.Device, error)
	GetGroups(ctx context.Context, filters map[string]interface{}) ([]prtg.Group, error)
	GetObjects(ctx context.Context, filters map[string]interface{}) ([]prtg.PRTGObject, error)
	GetObjectCounts(ctx context.Context) (*prtg.SensorCount, error)
	GetStatusSummary(ctx context.Context) (*prtg.StatusSummary, error)
}

// ChannelStore provides access to channel history exported by PRTG Data Exporter.
type ChannelStore interface {
	GetChannelLastValues(ctx context.Context, sensorID int, maxAge time.Duration) ([]types.ChannelLastValue, error)
	GetChannelHistory(ctx context.Context, sensorID int, from, to time.Time, bucket time.Duration) (*types.ChannelHistory, error)
	GetSensorExportState(ctx context.Context, sensorID int) (*types.SensorExportState, error)
}

const (
	sourceAuto     = "auto"
	sourceDatabase = "database"
	sourcePRTG     = "prtg"
)

const sourceParamDescription = "auto: database first, PRTG API only when the database has nothing; " +
	"database: never call PRTG; prtg: real-time values from the PRTG API (adds load on the PRTG server)"

var errNoStoreData = errors.New("no channel data available in store")

// MetricsToolHandler handles MCP tool requests for PRTG metrics/historical data.
type MetricsToolHandler struct {
	prtgClient PRTGClient
	store      ChannelStore
	handler    *ToolHandler // Reference to main handler for database queries
}

// NewMetricsToolHandler creates a new metrics tool handler backed only by the PRTG API.
func NewMetricsToolHandler(prtgClient PRTGClient, mainHandler *ToolHandler) *MetricsToolHandler {
	return &MetricsToolHandler{
		prtgClient: prtgClient,
		handler:    mainHandler,
	}
}

// NewMetricsToolHandlerWithStore creates a new metrics tool handler with an optional channel store.
// Either prtgClient or store may be nil, never both.
func NewMetricsToolHandlerWithStore(prtgClient PRTGClient, store ChannelStore, mainHandler *ToolHandler) *MetricsToolHandler {
	return &MetricsToolHandler{
		prtgClient: prtgClient,
		store:      store,
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
			"compare metrics between time periods, detect patterns in historical data. " +
			"Data is read from the PRTG Data Exporter database first; values may lag PRTG by the Data Exporter schedule.",
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
				"source": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"auto", "database", "prtg"},
					"description": sourceParamDescription,
				},
			},
			Required: []string{"sensor_id", "time_type"},
		},
	}, h.handleGetSensorTimeSeries)

	// Tool 2: prtg_get_channel_current_values
	s.AddTool(mcp.Tool{
		Name: "prtg_get_channel_current_values",
		Description: "**PRIMARY TOOL for checking sensor current state and discovering available channels.** " +
			"Returns ALL channels of a sensor with their current values, names, units, and last update time. " +
			"Each PRTG sensor has multiple channels (measurements). Examples: " +
			"SSL sensors → 'Days to Expiration', 'Response Time'; " +
			"Server sensors → 'CPU Load', 'Memory Usage', 'Disk Space'; " +
			"Network sensors → 'Traffic In', 'Traffic Out', 'Packet Loss'. " +
			"**ALWAYS use this tool first** when asked about a sensor's current state, values, or status. " +
			"Use prtg_get_sensor_timeseries only for historical trends. " +
			"Data is read from the PRTG Data Exporter database first; values may lag PRTG by the Data Exporter schedule.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"sensor_id": map[string]interface{}{
					"type":        "integer",
					"description": "PRTG sensor ID",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"auto", "database", "prtg"},
					"description": sourceParamDescription,
				},
			},
			Required: []string{"sensor_id"},
		},
	}, h.handleGetChannelCurrentValues)
}

// parseSource validates the optional source parameter, defaulting to auto.
func parseSource(s string) (string, error) {
	if s == "" {
		return sourceAuto, nil
	}
	switch s {
	case sourceAuto, sourceDatabase, sourcePRTG:
		return s, nil
	}
	return "", errors.New("invalid source: must be auto, database or prtg")
}

// windowForTimeType maps a PRTG time type to a database window and bucket.
func windowForTimeType(t prtg.TimeSeriesType) (window, bucket time.Duration) {
	switch t {
	case prtg.TimeSeriesLive:
		return 2 * time.Hour, 5 * time.Minute
	case prtg.TimeSeriesShort:
		return 24 * time.Hour, 15 * time.Minute
	case prtg.TimeSeriesMedium:
		return 7 * 24 * time.Hour, time.Hour
	case prtg.TimeSeriesLong:
		return 30 * 24 * time.Hour, 6 * time.Hour
	}
	return 24 * time.Hour, 15 * time.Minute
}

// exportStaleness reports whether the Data Exporter export for a sensor is stale, with a human-readable reason.
func exportStaleness(state *types.SensorExportState, now time.Time) (stale bool, note string) {
	if state == nil {
		return false, ""
	}
	switch {
	case !state.Tracked:
		return true, "Data Exporter does not export this sensor's channel history."
	case state.NoMonitoringData:
		return true, "Data Exporter reports no monitoring data for this sensor."
	case state.LastFetchUTC == nil:
		return true, "Data Exporter has not exported this sensor's channel history yet."
	case now.Sub(*state.LastFetchUTC) > database.ExportFreshWithin:
		return true, fmt.Sprintf("Data Exporter stopped exporting this sensor on %s; later values are missing.",
			state.LastFetchUTC.UTC().Format("2006-01-02 15:04 UTC"))
	}
	return false, ""
}

// stalenessSuffix is appended to every staleness note.
func (h *MetricsToolHandler) stalenessSuffix() string {
	if h.prtgClient != nil {
		return " Add the sensor to the Data Exporter metrics profile, or use source=prtg for real-time values."
	}
	return " Add the sensor to the Data Exporter metrics profile."
}

// checkExportFreshness queries the export state before reading the store.
// notFound mirrors the store's own ErrSensorNotFound handling.
func (h *MetricsToolHandler) checkExportFreshness(ctx context.Context, sensorID int) (stale bool, note string, notFound bool) {
	if h.store == nil {
		return false, "", false
	}
	state, err := h.store.GetSensorExportState(ctx, sensorID)
	if err != nil {
		if errors.Is(err, database.ErrNoChannelData) {
			return false, "", false
		}
		if errors.Is(err, database.ErrSensorNotFound) {
			return false, "", true
		}
		h.handler.logger.Warn().
			Err(err).
			Int("sensor_id", sensorID).
			Msg("Failed to get sensor export state; continuing without freshness information")
		return false, "", false
	}
	stale, reason := exportStaleness(state, time.Now())
	if !stale {
		return false, "", false
	}
	return true, reason + h.stalenessSuffix(), false
}

// resolveChannelNames replaces fallback channel names using the PRTG API when available.
func (h *MetricsToolHandler) resolveChannelNames(ctx context.Context, sensorID int, channels []types.ChannelInfo) (resolved, stillMissing bool) {
	hasMissing := false
	for _, ch := range channels {
		if ch.NameMissing {
			hasMissing = true
			break
		}
	}
	if !hasMissing {
		return false, false
	}
	if h.prtgClient == nil {
		return false, true
	}
	prtgChannels, err := h.prtgClient.GetChannelsBySensor(ctx, sensorID)
	if err != nil {
		h.handler.logger.Warn().
			Err(err).
			Int("sensor_id", sensorID).
			Msg("Failed to fetch channel names from PRTG API; keeping fallback names")
		return false, true
	}
	byChannelID := make(map[string]prtg.Channel, len(prtgChannels))
	for _, c := range prtgChannels {
		if idx := strings.LastIndex(c.ID, "."); idx >= 0 {
			byChannelID[c.ID[idx+1:]] = c
		}
	}
	for i := range channels {
		if !channels[i].NameMissing {
			continue
		}
		c, ok := byChannelID[strconv.Itoa(channels[i].ID)]
		if !ok {
			continue
		}
		channels[i].Name = c.Name
		channels[i].NameMissing = false
		if channels[i].Unit == "" {
			channels[i].Unit = c.Basic.DisplayUnit
		}
		resolved = true
	}
	for _, ch := range channels {
		if ch.NameMissing {
			stillMissing = true
			break
		}
	}
	return resolved, stillMissing
}

// resolveLastValueNames is resolveChannelNames for ChannelLastValue slices.
func (h *MetricsToolHandler) resolveLastValueNames(ctx context.Context, sensorID int, values []types.ChannelLastValue) (resolved, stillMissing bool) {
	infos := make([]types.ChannelInfo, len(values))
	for i, v := range values {
		infos[i] = v.ChannelInfo
	}
	resolved, stillMissing = h.resolveChannelNames(ctx, sensorID, infos)
	for i := range values {
		values[i].ChannelInfo = infos[i]
	}
	return resolved, stillMissing
}

const missingNamesNote = "Note: channel names are not exported for this sensor; channels are identified by their PRTG channel ID."

// prtgUnreachableNote builds the note shown when the PRTG API fails but exported data is served instead.
func prtgUnreachableNote(err error) string {
	msg := err.Error()
	if len(msg) > 120 {
		msg = msg[:120]
	}
	return fmt.Sprintf("Note: the PRTG API is not reachable (%s); showing exported data.", msg)
}

// handleGetSensorTimeSeries handles prtg_get_sensor_timeseries tool requests.
func (h *MetricsToolHandler) handleGetSensorTimeSeries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SensorID int    `json:"sensor_id"`
		TimeType string `json:"time_type"`
		Source   string `json:"source"`
	}

	if err := parseArguments(request.Params.Arguments, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Validate sensor_id
	if params.SensorID <= 0 {
		return ErrInvalidSensorID(), nil
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

	source, err := parseSource(params.Source)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch source {
	case sourcePRTG:
		if h.prtgClient == nil {
			return mcp.NewToolResultError("source=prtg requested but no PRTG API client is configured"), nil
		}
		res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
		if prtgErr != nil {
			return mcp.NewToolResultError(prtgErr.Error()), nil
		}
		return res, nil
	case sourceDatabase:
		if h.store == nil {
			return mcp.NewToolResultError("source=database requested but no channel database is configured"), nil
		}
		_, note, notFound := h.checkExportFreshness(ctx, params.SensorID)
		if notFound {
			return mcp.NewToolResultText(fmt.Sprintf("No exported channel history for sensor %d", params.SensorID)), nil
		}
		result, err := h.timeSeriesFromStore(ctx, params.SensorID, timeType, note)
		if err != nil {
			if errors.Is(err, errNoStoreData) {
				return mcp.NewToolResultText(noHistoryText(params.SensorID, note)), nil
			}
			h.handler.logger.Error().
				Err(err).
				Int("sensor_id", params.SensorID).
				Msg("Failed to fetch time series from channel database")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch time series from database: %v", err)), nil
		}
		return result, nil
	default: // auto
		if timeType == prtg.TimeSeriesLive && h.prtgClient != nil {
			res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
			if prtgErr == nil {
				return res, nil
			}
			if h.store == nil {
				return mcp.NewToolResultError(prtgErr.Error()), nil
			}
			h.handler.logger.Warn().
				Err(prtgErr).
				Int("sensor_id", params.SensorID).
				Msg("PRTG API unreachable; serving live window from the channel database")
			note := prtgUnreachableNote(prtgErr)
			result, err := h.timeSeriesFromStore(ctx, params.SensorID, timeType, note)
			if err == nil {
				return result, nil
			}
			if errors.Is(err, errNoStoreData) {
				return mcp.NewToolResultText(noHistoryText(params.SensorID, note)), nil
			}
			h.handler.logger.Error().
				Err(err).
				Int("sensor_id", params.SensorID).
				Msg("Failed to fetch time series from channel database")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch time series from database: %v", err)), nil
		}
		if h.store != nil {
			stale, note, notFound := h.checkExportFreshness(ctx, params.SensorID)
			if notFound {
				if h.prtgClient != nil {
					res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
					if prtgErr == nil {
						return res, nil
					}
					return mcp.NewToolResultError(prtgErr.Error()), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("No exported channel history for sensor %d", params.SensorID)), nil
			}
			if stale && h.prtgClient != nil {
				h.handler.logger.Info().
					Int("sensor_id", params.SensorID).
					Str("reason", "export_stale").
					Msg("Data Exporter export is stale; serving time series from PRTG API")
				res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
				if prtgErr == nil {
					return res, nil
				}
				h.handler.logger.Warn().
					Err(prtgErr).
					Int("sensor_id", params.SensorID).
					Msg("PRTG API unreachable; serving stale export from the channel database")
				note += "\n" + prtgUnreachableNote(prtgErr)
			}
			result, err := h.timeSeriesFromStore(ctx, params.SensorID, timeType, note)
			if err == nil {
				return result, nil
			}
			if !errors.Is(err, errNoStoreData) {
				h.handler.logger.Error().
					Err(err).
					Int("sensor_id", params.SensorID).
					Msg("Failed to fetch time series from channel database")
				return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch time series from database: %v", err)), nil
			}
			if h.prtgClient != nil && !stale {
				res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
				if prtgErr == nil {
					return res, nil
				}
				return mcp.NewToolResultError(prtgErr.Error()), nil
			}
			return mcp.NewToolResultText(noHistoryText(params.SensorID, note)), nil
		}
		if h.prtgClient != nil {
			res, prtgErr := h.timeSeriesFromPRTG(ctx, params.SensorID, timeType)
			if prtgErr != nil {
				return mcp.NewToolResultError(prtgErr.Error()), nil
			}
			return res, nil
		}
		return mcp.NewToolResultError("No metrics source configured: neither channel database nor PRTG API client is available"), nil
	}
}

// noHistoryText builds the no-data message, carrying the staleness note when present.
func noHistoryText(sensorID int, note string) string {
	text := fmt.Sprintf("No exported channel history for sensor %d", sensorID)
	if note != "" {
		text += "\n" + note
	}
	return text
}

// timeSeriesFromPRTG fetches time series from the PRTG API v2.
// A non-nil error means the PRTG call itself failed; the result is nil in that case.
func (h *MetricsToolHandler) timeSeriesFromPRTG(ctx context.Context, sensorID int, timeType prtg.TimeSeriesType) (*mcp.CallToolResult, error) {
	data, err := h.prtgClient.GetTimeSeries(ctx, sensorID, timeType)
	if err != nil {
		h.handler.logger.Error().
			Err(err).
			Int("sensor_id", sensorID).
			Msg("Failed to fetch time series from PRTG API")
		return nil, fmt.Errorf("Failed to fetch time series: %w", err)
	}

	h.handler.logger.Info().
		Int("sensor_id", sensorID).
		Str("source", sourcePRTG).
		Str("time_type", string(timeType)).
		Msg("Served sensor time series")

	formatted := formatTimeSeriesForLLM(data)
	return mcp.NewToolResultText("Source: PRTG API v2 (real time)\n" + formatted), nil
}

// timeSeriesFromStore fetches time series from the channel database.
func (h *MetricsToolHandler) timeSeriesFromStore(ctx context.Context, sensorID int, timeType prtg.TimeSeriesType, note string) (*mcp.CallToolResult, error) {
	window, bucket := windowForTimeType(timeType)
	to := time.Now()
	from := to.Add(-window)

	hist, err := h.store.GetChannelHistory(ctx, sensorID, from, to, bucket)
	if err != nil {
		if errors.Is(err, database.ErrNoChannelData) || errors.Is(err, database.ErrSensorNotFound) {
			return nil, fmt.Errorf("%w: %w", errNoStoreData, err)
		}
		return nil, err
	}
	if hist == nil || len(hist.Points) == 0 {
		return nil, errNoStoreData
	}

	resolved, stillMissing := h.resolveChannelNames(ctx, sensorID, hist.Channels)

	h.handler.logger.Info().
		Int("sensor_id", sensorID).
		Str("source", sourceDatabase).
		Str("time_type", string(timeType)).
		Msg("Served sensor time series")

	data := historyToTimeSeries(sensorID, timeType, hist)
	formatted := formatTimeSeriesForLLM(data)

	latest := hist.To
	if len(hist.Points) > 0 {
		latest = hist.Points[len(hist.Points)-1].Timestamp
	}
	var prefix strings.Builder
	prefix.WriteString(fmt.Sprintf("Source: PRTG Data Exporter database (averages per %s, latest point %s)",
		hist.Bucket, latest.UTC().Format(time.RFC3339)))
	if resolved {
		prefix.WriteString(" (channel names from the PRTG API)")
	}
	prefix.WriteString("\n")
	if note != "" {
		prefix.WriteString(note + "\n")
	}
	if stillMissing {
		prefix.WriteString(missingNamesNote + "\n")
	}

	return mcp.NewToolResultText(prefix.String() + formatted), nil
}

// historyToTimeSeries converts a ChannelHistory into prtg.TimeSeriesData.
func historyToTimeSeries(sensorID int, timeType prtg.TimeSeriesType, hist *types.ChannelHistory) *prtg.TimeSeriesData {
	headers := []string{"Timestamp"}
	labels := make(map[int]string, len(hist.Channels))
	units := make(map[int]string, len(hist.Channels))
	used := make(map[string]bool, len(hist.Channels))
	for _, ch := range hist.Channels {
		label := ch.Name
		if ch.Unit != "" {
			label = fmt.Sprintf("%s (%s)", ch.Name, ch.Unit)
		}
		if used[label] {
			label = fmt.Sprintf("%s #%d", label, ch.ID)
		}
		used[label] = true
		labels[ch.ID] = label
		units[ch.ID] = ch.Unit
		headers = append(headers, label)
	}

	points := make([]prtg.TimeSeriesDataPoint, 0, len(hist.Points))
	for _, p := range hist.Points {
		values := make(map[string]interface{}, len(p.Values))
		for id, v := range p.Values {
			if label, ok := labels[id]; ok {
				values[label] = displayValue(v, units[id])
			}
		}
		points = append(points, prtg.TimeSeriesDataPoint{
			Timestamp: p.Timestamp,
			Values:    values,
		})
	}

	from, to := hist.From, hist.To
	return &prtg.TimeSeriesData{
		ObjectID:   sensorID,
		TimeType:   timeType,
		StartTime:  &from,
		EndTime:    &to,
		Headers:    headers,
		DataPoints: points,
	}
}

// handleGetChannelCurrentValues handles prtg_get_channel_current_values tool requests.
func (h *MetricsToolHandler) handleGetChannelCurrentValues(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		SensorID int    `json:"sensor_id"`
		Source   string `json:"source"`
	}

	if err := parseArguments(request.Params.Arguments, &params); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid parameters: %v", err)), nil
	}

	// Validate sensor_id
	if params.SensorID <= 0 {
		return ErrInvalidSensorID(), nil
	}

	source, err := parseSource(params.Source)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	switch source {
	case sourcePRTG:
		if h.prtgClient == nil {
			return mcp.NewToolResultError("source=prtg requested but no PRTG API client is configured"), nil
		}
		res, prtgErr := h.currentValuesFromPRTG(ctx, params.SensorID)
		if prtgErr != nil {
			return mcp.NewToolResultError(prtgErr.Error()), nil
		}
		return res, nil
	case sourceDatabase:
		if h.store == nil {
			return mcp.NewToolResultError("source=database requested but no channel database is configured"), nil
		}
		_, note, notFound := h.checkExportFreshness(ctx, params.SensorID)
		if notFound {
			return mcp.NewToolResultText(noValuesText(params.SensorID, "")), nil
		}
		result, err := h.currentValuesFromStore(ctx, params.SensorID, note)
		if err != nil {
			if errors.Is(err, errNoStoreData) {
				return mcp.NewToolResultText(noValuesText(params.SensorID, note)), nil
			}
			h.handler.logger.Error().
				Err(err).
				Int("sensor_id", params.SensorID).
				Msg("Failed to fetch channel values from channel database")
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch channel values from database: %v", err)), nil
		}
		return result, nil
	default: // auto
		if h.store != nil {
			stale, note, notFound := h.checkExportFreshness(ctx, params.SensorID)
			if notFound {
				if h.prtgClient != nil {
					res, prtgErr := h.currentValuesFromPRTG(ctx, params.SensorID)
					if prtgErr == nil {
						return res, nil
					}
					return mcp.NewToolResultError(prtgErr.Error()), nil
				}
				return mcp.NewToolResultText(noValuesText(params.SensorID, "")), nil
			}
			if stale && h.prtgClient != nil {
				h.handler.logger.Info().
					Int("sensor_id", params.SensorID).
					Str("reason", "export_stale").
					Msg("Data Exporter export is stale; serving current values from PRTG API")
				res, prtgErr := h.currentValuesFromPRTG(ctx, params.SensorID)
				if prtgErr == nil {
					return res, nil
				}
				h.handler.logger.Warn().
					Err(prtgErr).
					Int("sensor_id", params.SensorID).
					Msg("PRTG API unreachable; serving stale export from the channel database")
				note += "\n" + prtgUnreachableNote(prtgErr)
			}
			result, err := h.currentValuesFromStore(ctx, params.SensorID, note)
			if err == nil {
				return result, nil
			}
			if !errors.Is(err, errNoStoreData) {
				h.handler.logger.Error().
					Err(err).
					Int("sensor_id", params.SensorID).
					Msg("Failed to fetch channel values from channel database")
				return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch channel values from database: %v", err)), nil
			}
			if h.prtgClient != nil && !stale {
				res, prtgErr := h.currentValuesFromPRTG(ctx, params.SensorID)
				if prtgErr == nil {
					return res, nil
				}
				return mcp.NewToolResultError(prtgErr.Error()), nil
			}
			return mcp.NewToolResultText(noValuesText(params.SensorID, note)), nil
		}
		if h.prtgClient != nil {
			res, prtgErr := h.currentValuesFromPRTG(ctx, params.SensorID)
			if prtgErr != nil {
				return mcp.NewToolResultError(prtgErr.Error()), nil
			}
			return res, nil
		}
		return mcp.NewToolResultError("No metrics source configured: neither channel database nor PRTG API client is available"), nil
	}
}

// noValuesText builds the no-data message, carrying the staleness note when present.
func noValuesText(sensorID int, note string) string {
	text := fmt.Sprintf("No exported channel values for sensor %d in the last 7 days", sensorID)
	if note != "" {
		text += "\n" + note
	}
	return text
}

// currentValuesFromPRTG fetches current channel values from the PRTG API v2.
// A non-nil error means the PRTG call itself failed; the result is nil in that case.
func (h *MetricsToolHandler) currentValuesFromPRTG(ctx context.Context, sensorID int) (*mcp.CallToolResult, error) {
	channels, err := h.prtgClient.GetChannelsBySensor(ctx, sensorID)
	if err != nil {
		h.handler.logger.Error().
			Err(err).
			Int("sensor_id", sensorID).
			Msg("Failed to fetch channels from PRTG API")
		return nil, fmt.Errorf("Failed to fetch channels: %w", err)
	}

	h.handler.logger.Info().
		Int("sensor_id", sensorID).
		Str("source", sourcePRTG).
		Msg("Served channel current values")

	var formatted string
	if len(channels) == 0 {
		formatted = fmt.Sprintf("No channels found for sensor %d", sensorID)
	} else {
		formatted = formatChannelsForLLM(sensorID, channels)
	}
	return mcp.NewToolResultText("Source: PRTG API v2 (real time)\n" + formatted), nil
}

// currentValuesFromStore fetches the latest exported channel values from the database.
func (h *MetricsToolHandler) currentValuesFromStore(ctx context.Context, sensorID int, note string) (*mcp.CallToolResult, error) {
	values, err := h.store.GetChannelLastValues(ctx, sensorID, 7*24*time.Hour)
	if err != nil {
		if errors.Is(err, database.ErrNoChannelData) || errors.Is(err, database.ErrSensorNotFound) {
			return nil, fmt.Errorf("%w: %w", errNoStoreData, err)
		}
		return nil, err
	}
	if len(values) == 0 {
		return nil, errNoStoreData
	}

	resolved, stillMissing := h.resolveLastValueNames(ctx, sensorID, values)

	h.handler.logger.Info().
		Int("sensor_id", sensorID).
		Str("source", sourceDatabase).
		Msg("Served channel current values")

	newest := values[0].Timestamp
	for _, v := range values[1:] {
		if v.Timestamp.After(newest) {
			newest = v.Timestamp
		}
	}

	var b strings.Builder
	b.WriteString("Source: PRTG Data Exporter database")
	if resolved {
		b.WriteString(" (channel names from the PRTG API)")
	}
	b.WriteString("\n")
	if note != "" {
		b.WriteString(note + "\n")
	}
	if stillMissing {
		b.WriteString(missingNamesNote + "\n")
	}
	if age := time.Since(newest); age > 2*time.Hour {
		b.WriteString(fmt.Sprintf("Note: the newest exported value is %s old; use source=prtg for real-time values.\n", age.Round(time.Minute)))
	}
	b.WriteString("\n| Channel | Value | Unit | Timestamp (UTC) |\n")
	b.WriteString("|---------|-------|------|-------------------|\n")
	for _, v := range values {
		b.WriteString(fmt.Sprintf("| %s | %.2f | %s | %s |\n",
			v.Name, displayValue(v.Value, v.Unit), v.Unit, v.Timestamp.UTC().Format(time.RFC3339)))
	}

	return mcp.NewToolResultText(b.String()), nil
}

// extractChannelValues extracts numeric values from a specific channel
func extractChannelValues(data *prtg.TimeSeriesData, channelName string) []float64 {
	var values []float64
	for _, point := range data.DataPoints {
		if val, ok := point.Values[channelName]; ok {
			// Try to convert to float64
			switch v := val.(type) {
			case float64:
				values = append(values, v)
			case int:
				values = append(values, float64(v))
			case int64:
				values = append(values, float64(v))
			}
		}
	}
	return values
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

	// Add sparkline and statistics for the first numeric channel
	if len(data.Headers) > 1 {
		// Try to extract values from the first channel (skip timestamp at index 0)
		firstChannel := data.Headers[1]
		values := extractChannelValues(data, firstChannel)

		if len(values) > 0 {
			output += "## Quick Trend\n\n"
			output += fmt.Sprintf("**Sparkline (%s):** %s %s\n",
				firstChannel,
				Sparkline(values, 40),
				TrendIndicator(values))
			output += fmt.Sprintf("**Stats:** %s\n\n", CompactStats(values))

			// Detect anomalies
			anomalies := DetectAnomalies(values)
			if len(anomalies) > 0 {
				output += fmt.Sprintf("WARNING: **%d anomaly/anomalies detected**\n", len(anomalies))
				maxDisplay := 3
				if len(anomalies) < maxDisplay {
					maxDisplay = len(anomalies)
				}
				for i := 0; i < maxDisplay; i++ {
					a := anomalies[i]
					output += fmt.Sprintf("- Index %d: %.1f (expected: ~%.1f)\n",
						a.Index, a.Value, a.Expected)
				}
				output += "\n"
			}
		}
	}

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
