package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// GetSensors retrieves sensors matching the given filters.
// Results are ordered by sensor name. The limit parameter controls the maximum number of results.
func (db *DB) GetSensors(ctx context.Context, deviceName, sensorName string, status *int, tags string, limit int) ([]types.Sensor, error) {
	return db.GetSensorsExtended(ctx, deviceName, sensorName, "", "", status, tags, "name", limit)
}

// GetSensorsExtended retrieves sensors matching the given filters with additional options.
// Supports filtering by sensor_type, group_name, and custom ordering.
func (db *DB) GetSensorsExtended(ctx context.Context, deviceName, sensorName, sensorType, groupName string, status *int, tags, orderBy string, limit int) ([]types.Sensor, error) {
	// Query with group join for group_name filter
	query := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			'' AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		INNER JOIN prtg_group g ON d.prtg_group_id = g.id
			AND d.prtg_server_address_id = g.prtg_server_address_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Add filters
	if deviceName != "" {
		query += fmt.Sprintf(" AND d.name ILIKE $%d", argPos)
		args = append(args, "%"+deviceName+"%")
		argPos++
	}

	if sensorName != "" {
		query += fmt.Sprintf(" AND s.name ILIKE $%d", argPos)
		args = append(args, "%"+sensorName+"%")
		argPos++
	}

	if sensorType != "" {
		query += fmt.Sprintf(" AND s.sensor_type ILIKE $%d", argPos)
		args = append(args, "%"+sensorType+"%")
		argPos++
	}

	if groupName != "" {
		query += fmt.Sprintf(" AND g.name ILIKE $%d", argPos)
		args = append(args, "%"+groupName+"%")
		argPos++
	}

	if status != nil {
		query += fmt.Sprintf(" AND s.status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	// Tags filter temporarily disabled for performance
	// TODO: Re-enable with proper indexing
	_ = tags

	// Add ordering
	orderClause := " ORDER BY s.name" // Default
	switch orderBy {
	case "status":
		orderClause = " ORDER BY s.status, s.name"
	case "priority":
		orderClause = " ORDER BY s.priority DESC, s.name"
	case "device":
		orderClause = " ORDER BY d.name, s.name"
	case "type":
		orderClause = " ORDER BY s.sensor_type, s.name"
	case "last_check":
		orderClause = " ORDER BY s.last_check_utc DESC NULLS LAST, s.name"
	}
	query += orderClause

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
	}

	db.logger.Debug().
		Str("query", query).
		Interface("args", args).
		Msg("executing GetSensors query")

	startTime := time.Now()
	rows, err := db.Query(ctx, query, args...)
	queryDuration := time.Since(startTime)

	if err != nil {
		db.logger.Error().
			Err(err).
			Dur("duration_ms", queryDuration).
			Str("query", query).
			Msg("query failed")

		return nil, fmt.Errorf("query failed: %w", err)
	}

	defer rows.Close()

	db.logger.Info().Dur("query_duration_ms", queryDuration).Msg("query executed, scanning rows")

	scanStart := time.Now()
	sensors, err := scanSensors(rows)
	scanDuration := time.Since(scanStart)

	if err != nil {
		db.logger.Error().Err(err).Dur("scan_duration_ms", scanDuration).Msg("scanSensors failed")
		return nil, err
	}

	db.logger.Info().
		Int("sensors_count", len(sensors)).
		Dur("query_ms", queryDuration).
		Dur("scan_ms", scanDuration).
		Dur("total_ms", time.Since(startTime)).
		Msg("GetSensors completed")

	return sensors, nil
}

// GetSensorByID retrieves a single sensor by ID.
// Returns sql.ErrNoRows if the sensor is not found.
func (db *DB) GetSensorByID(ctx context.Context, sensorID int) (*types.Sensor, error) {
	query := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			COALESCE(
				(SELECT string_agg(t.name, ',')
				 FROM prtg_sensor_tag st
				 JOIN prtg_tag t ON st.prtg_tag_id = t.id
				 WHERE st.prtg_sensor_id = s.id
				 AND st.prtg_server_address_id = s.prtg_server_address_id),
				''
			) AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		WHERE s.id = $1
	`

	var sensor types.Sensor

	var lastCheckUTC, lastDownUTC sql.NullTime

	var uptimeSecs, downtimeSecs sql.NullFloat64

	var message, tags sql.NullString

	err := db.QueryRow(ctx, query, sensorID).Scan(
		&sensor.ID,
		&sensor.ServerID,
		&sensor.Name,
		&sensor.SensorType,
		&sensor.DeviceID,
		&sensor.DeviceName,
		&sensor.ScanningIntervalSecs,
		&sensor.Status,
		&lastCheckUTC,
		&sensor.LastUpUTC,
		&lastDownUTC,
		&sensor.Priority,
		&message,
		&uptimeSecs,
		&downtimeSecs,
		&sensor.FullPath,
		&tags,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("sensor not found")
		}

		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Handle nullable fields
	if lastCheckUTC.Valid {
		sensor.LastCheckUTC = &lastCheckUTC.Time
	}

	if lastDownUTC.Valid {
		sensor.LastDownUTC = &lastDownUTC.Time
	}

	if uptimeSecs.Valid {
		sensor.UptimeSinceSecs = &uptimeSecs.Float64
	}

	if downtimeSecs.Valid {
		sensor.DowntimeSinceSecs = &downtimeSecs.Float64
	}

	if message.Valid {
		sensor.Message = message.String
	}

	if tags.Valid {
		sensor.Tags = tags.String
	}

	sensor.StatusText = types.GetStatusText(sensor.Status)

	return &sensor, nil
}

// GetAlerts retrieves sensors in alert state (non-UP status).
// Results are sorted by priority and severity (Down first, then Warning, etc.), limited to 100 results.
func (db *DB) GetAlerts(ctx context.Context, hours int, statusFilter *int, deviceName string) ([]types.Sensor, error) {
	query := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			COALESCE(
				(SELECT string_agg(t.name, ',')
				 FROM prtg_sensor_tag st
				 JOIN prtg_tag t ON st.prtg_tag_id = t.id
				 WHERE st.prtg_sensor_id = s.id
				 AND st.prtg_server_address_id = s.prtg_server_address_id),
				''
			) AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		WHERE s.status != $1
	`

	args := []interface{}{types.StatusUp}
	argPos := 2

	if hours > 0 {
		query += fmt.Sprintf(" AND s.last_check_utc >= NOW() - ($%d || ' hours')::interval", argPos)

		args = append(args, hours)
		argPos++
	}

	if statusFilter != nil {
		query += fmt.Sprintf(" AND s.status = $%d", argPos)

		args = append(args, *statusFilter)
		argPos++
	}

	if deviceName != "" {
		query += fmt.Sprintf(" AND d.name ILIKE $%d", argPos)

		args = append(args, "%"+deviceName+"%")
	}

	// Order by severity: Down statuses first, then Warning, then others
	// Severity order: Down(5), DownPartial(14), DownAcknowledged(13), Warning(4), Unusual(10),
	//                 NoProbe(6), Unknown(1), Collecting(2), then Paused statuses
	query += ` ORDER BY
		s.priority DESC,
		CASE s.status
			WHEN 5 THEN 1   -- Down (most critical)
			WHEN 14 THEN 2  -- Down Partial
			WHEN 13 THEN 3  -- Down Acknowledged
			WHEN 4 THEN 4   -- Warning
			WHEN 10 THEN 5  -- Unusual
			WHEN 6 THEN 6   -- No Probe
			WHEN 1 THEN 7   -- Unknown
			WHEN 2 THEN 8   -- Collecting
			ELSE 9          -- Paused statuses (7,8,9,11,12)
		END,
		s.name
		LIMIT 100`

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanSensors(rows)
}

// GetDeviceOverview retrieves a device with all its sensors and aggregated statistics.
// Returns sql.ErrNoRows if no device matches the given name.
func (db *DB) GetDeviceOverview(ctx context.Context, deviceName string) (*types.DeviceOverview, error) {
	// Get device info
	deviceQuery := `
		SELECT
			d.id,
			d.prtg_server_address_id,
			d.name,
			d.host,
			d.prtg_group_id,
			g.name AS group_name,
			dp.path AS full_path,
			d.tree_depth,
			COALESCE(
				(SELECT COUNT(*) FROM prtg_sensor s
				 WHERE s.prtg_device_id = d.id
				 AND s.prtg_server_address_id = d.prtg_server_address_id),
				0
			) AS sensor_count
		FROM prtg_device d
		INNER JOIN prtg_group g ON d.prtg_group_id = g.id
			AND d.prtg_server_address_id = g.prtg_server_address_id
		INNER JOIN prtg_device_path dp ON d.id = dp.device_id
			AND d.prtg_server_address_id = dp.prtg_server_address_id
		WHERE d.name ILIKE $1
		LIMIT 1
	`

	var device types.Device
	err := db.QueryRow(ctx, deviceQuery, "%"+deviceName+"%").Scan(
		&device.ID,
		&device.ServerID,
		&device.Name,
		&device.Host,
		&device.GroupID,
		&device.GroupName,
		&device.FullPath,
		&device.TreeDepth,
		&device.SensorCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("device not found")
		}

		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Get all sensors for this device
	sensorsQuery := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			$2 AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			COALESCE(
				(SELECT string_agg(t.name, ',')
				 FROM prtg_sensor_tag st
				 JOIN prtg_tag t ON st.prtg_tag_id = t.id
				 WHERE st.prtg_sensor_id = s.id
				 AND st.prtg_server_address_id = s.prtg_server_address_id),
				''
			) AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		WHERE s.prtg_device_id = $1
		AND s.prtg_server_address_id = $3
		ORDER BY s.status, s.name
	`

	rows, err := db.Query(ctx, sensorsQuery, device.ID, device.Name, device.ServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sensors: %w", err)
	}
	defer rows.Close()

	sensors, err := scanSensors(rows)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	upCount := 0
	downCount := 0
	warnCount := 0

	for i := range sensors {
		switch sensors[i].Status {
		case types.StatusUp:
			upCount++
		case types.StatusDown:
			downCount++
		case types.StatusWarning:
			warnCount++
		}
	}

	return &types.DeviceOverview{
		Device:       device,
		Sensors:      sensors,
		TotalSensors: len(sensors),
		UpSensors:    upCount,
		DownSensors:  downCount,
		WarnSensors:  warnCount,
	}, nil
}

// GetTopSensors retrieves top sensors ranked by the given metric.
// Valid metrics: "uptime", "downtime", "alerts". Results are limited by the limit parameter.
func (db *DB) GetTopSensors(ctx context.Context, metric, sensorType string, limit, _ int) ([]types.Sensor, error) {
	query := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			COALESCE(
				(SELECT string_agg(t.name, ',')
				 FROM prtg_sensor_tag st
				 JOIN prtg_tag t ON st.prtg_tag_id = t.id
				 WHERE st.prtg_sensor_id = s.id
				 AND st.prtg_server_address_id = s.prtg_server_address_id),
				''
			) AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if sensorType != "" {
		query += fmt.Sprintf(" AND s.sensor_type ILIKE $%d", argPos)

		args = append(args, "%"+sensorType+"%")
		argPos++
	}

	// Add ordering based on metric
	switch metric {
	case "downtime":
		query += " ORDER BY s.downtime_since_seconds DESC NULLS LAST"
	case "alerts":
		// Order by non-UP status, then by priority
		query += fmt.Sprintf(" AND s.status != $%d ORDER BY s.priority DESC, s.status", argPos)

		args = append(args, types.StatusUp)
		argPos++
	default: // "uptime" or default
		query += " ORDER BY s.uptime_since_seconds DESC NULLS LAST"
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)

		args = append(args, limit)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanSensors(rows)
}

// ExecuteCustomQuery executes a custom SQL SELECT query with security validation.
// Only SELECT queries are allowed - INSERT/UPDATE/DELETE/DROP are rejected.
// This function should be disabled in production (set allow_custom_queries: false in config).
func (db *DB) ExecuteCustomQuery(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	// Security: Validate query is SELECT only
	queryUpper := strings.ToUpper(strings.TrimSpace(query))
	if !strings.HasPrefix(queryUpper, "SELECT") {
		return nil, fmt.Errorf("only SELECT queries are allowed")
	}

	// Check for dangerous keywords (including comments to prevent bypass)
	dangerous := []string{"DROP", "DELETE", "UPDATE", "INSERT", "ALTER", "CREATE", "TRUNCATE", "EXEC", "EXECUTE", "/*", "--", ";"}
	for _, keyword := range dangerous {
		if strings.Contains(queryUpper, keyword) || strings.Contains(query, keyword) {
			return nil, fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	// Enforce maximum limit
	maxLimit := 1000

	if limit <= 0 {
		limit = 100
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	// Add limit if not present using parameterized query
	if !strings.Contains(queryUpper, "LIMIT") {
		query += " LIMIT $1"

		rows, err := db.conn.QueryContext(ctx, query, limit)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		return scanGenericResults(rows)
	}

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanGenericResults(rows)
}

// scanGenericResults scans generic SQL query results into maps.
func scanGenericResults(rows *sql.Rows) ([]map[string]interface{}, error) {
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	results := []map[string]interface{}{}

	for rows.Next() {
		// Create a slice of interface{}'s to represent each column
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Create map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}

		results = append(results, row)
	}

	return results, rows.Err()
}

// GetGroups retrieves all PRTG groups matching the given filters.
func (db *DB) GetGroups(ctx context.Context, groupName string, parentID *int, limit int) ([]types.Group, error) {
	query := `
		SELECT
			g.id,
			g.prtg_server_address_id,
			g.name,
			g.is_probe_node,
			g.self_group_id,
			gp.path AS full_path,
			g.tree_depth
		FROM prtg_group g
		INNER JOIN prtg_group_path gp ON g.id = gp.group_id
			AND g.prtg_server_address_id = gp.prtg_server_address_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if groupName != "" {
		query += fmt.Sprintf(" AND g.name ILIKE $%d", argPos)
		args = append(args, "%"+groupName+"%")
		argPos++
	}

	if parentID != nil {
		query += fmt.Sprintf(" AND g.self_group_id = $%d", argPos)
		args = append(args, *parentID)
		argPos++
	}

	query += " ORDER BY g.tree_depth, g.name"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	groups := []types.Group{}
	for rows.Next() {
		var group types.Group
		var parentID sql.NullInt32

		err := rows.Scan(
			&group.ID,
			&group.ServerID,
			&group.Name,
			&group.IsProbeNode,
			&parentID,
			&group.FullPath,
			&group.TreeDepth,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		if parentID.Valid {
			parentIDInt := int(parentID.Int32)
			group.ParentID = &parentIDInt
		}

		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// GetDevicesByGroupID retrieves all devices in a given group.
func (db *DB) GetDevicesByGroupID(ctx context.Context, groupID int) ([]types.Device, error) {
	query := `
		SELECT
			d.id,
			d.prtg_server_address_id,
			d.name,
			d.host,
			d.prtg_group_id,
			g.name AS group_name,
			dp.path AS full_path,
			COALESCE(
				(SELECT COUNT(*) FROM prtg_sensor s
				 WHERE s.prtg_device_id = d.id
				 AND s.prtg_server_address_id = d.prtg_server_address_id),
				0
			) AS sensor_count,
			d.tree_depth
		FROM prtg_device d
		INNER JOIN prtg_group g ON d.prtg_group_id = g.id
			AND d.prtg_server_address_id = g.prtg_server_address_id
		INNER JOIN prtg_device_path dp ON d.id = dp.device_id
			AND d.prtg_server_address_id = dp.prtg_server_address_id
		WHERE d.prtg_group_id = $1
		ORDER BY d.name
	`

	rows, err := db.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	devices := []types.Device{}
	for rows.Next() {
		var device types.Device

		err := rows.Scan(
			&device.ID,
			&device.ServerID,
			&device.Name,
			&device.Host,
			&device.GroupID,
			&device.GroupName,
			&device.FullPath,
			&device.SensorCount,
			&device.TreeDepth,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		devices = append(devices, device)
	}

	return devices, rows.Err()
}

// GetHierarchy retrieves the PRTG hierarchy starting from a group.
// If groupName is empty, returns root groups. Includes devices and optionally sensors.
func (db *DB) GetHierarchy(ctx context.Context, groupName string, includeSensors bool, maxDepth int) (*types.HierarchyNode, error) {
	// Get the starting group(s)
	var groups []types.Group
	var err error

	if groupName != "" {
		groups, err = db.GetGroups(ctx, groupName, nil, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get groups: %w", err)
		}
		if len(groups) == 0 {
			return nil, fmt.Errorf("group not found: %s", groupName)
		}
	} else {
		// Get root groups (parentID is NULL)
		query := `
			SELECT
				g.id,
				g.prtg_server_address_id,
				g.name,
				g.is_probe_node,
				g.self_group_id,
				gp.path AS full_path,
				g.tree_depth
			FROM prtg_group g
			INNER JOIN prtg_group_path gp ON g.id = gp.group_id
				AND g.prtg_server_address_id = gp.prtg_server_address_id
			WHERE g.self_group_id IS NULL
			ORDER BY g.name
			LIMIT 10
		`

		rows, err := db.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var group types.Group
			var parentID sql.NullInt32

			err := rows.Scan(
				&group.ID,
				&group.ServerID,
				&group.Name,
				&group.IsProbeNode,
				&parentID,
				&group.FullPath,
				&group.TreeDepth,
			)
			if err != nil {
				return nil, fmt.Errorf("scan failed: %w", err)
			}

			if parentID.Valid {
				parentIDInt := int(parentID.Int32)
				group.ParentID = &parentIDInt
			}

			groups = append(groups, group)
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no groups found")
	}

	// Build hierarchy starting from first group
	return db.buildHierarchyNode(ctx, &groups[0], includeSensors, maxDepth, 0)
}

// buildHierarchyNode recursively builds a hierarchy node.
func (db *DB) buildHierarchyNode(ctx context.Context, group *types.Group, includeSensors bool, maxDepth, currentDepth int) (*types.HierarchyNode, error) {
	node := &types.HierarchyNode{
		Group:   *group,
		Devices: []types.HierarchyDevice{},
		Groups:  []*types.HierarchyNode{},
	}

	// Stop if we've reached max depth
	if maxDepth > 0 && currentDepth >= maxDepth {
		return node, nil
	}

	// Get devices in this group
	devices, err := db.GetDevicesByGroupID(ctx, group.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	// Build device nodes
	for _, device := range devices {
		deviceNode := types.HierarchyDevice{
			Device:  device,
			Sensors: []types.Sensor{},
		}

		// Get sensors if requested
		if includeSensors {
			sensorsQuery := `
				SELECT
					s.id,
					s.prtg_server_address_id,
					s.name,
					s.sensor_type,
					s.prtg_device_id,
					$2 AS device_name,
					s.scanning_interval_seconds,
					s.status,
					s.last_check_utc,
					s.last_up_utc,
					s.last_down_utc,
					s.priority,
					s.message,
					s.uptime_since_seconds,
					s.downtime_since_seconds,
					sp.path AS full_path,
					'' AS tags
				FROM prtg_sensor s
				INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
					AND s.prtg_server_address_id = sp.prtg_server_address_id
				WHERE s.prtg_device_id = $1
				AND s.prtg_server_address_id = $3
				ORDER BY s.name
				LIMIT 50
			`

			rows, err := db.Query(ctx, sensorsQuery, device.ID, device.Name, device.ServerID)
			if err != nil {
				return nil, fmt.Errorf("failed to get sensors: %w", err)
			}

			sensors, err := scanSensors(rows)
			rows.Close()

			if err != nil {
				return nil, fmt.Errorf("failed to scan sensors: %w", err)
			}

			deviceNode.Sensors = sensors
		}

		node.Devices = append(node.Devices, deviceNode)
	}

	// Get child groups
	childGroups, err := db.GetGroups(ctx, "", &group.ID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get child groups: %w", err)
	}

	// Recursively build child nodes
	for i := range childGroups {
		childNode, err := db.buildHierarchyNode(ctx, &childGroups[i], includeSensors, maxDepth, currentDepth+1)
		if err != nil {
			return nil, err
		}
		node.Groups = append(node.Groups, childNode)
	}

	return node, nil
}

// Search performs a universal search across groups, devices, and sensors.
// Returns matching results organized by type.
func (db *DB) Search(ctx context.Context, searchTerm string, limit int) (*types.SearchResults, error) {
	if limit <= 0 {
		limit = 50
	}

	results := &types.SearchResults{
		Groups:  []types.Group{},
		Devices: []types.Device{},
		Sensors: []types.Sensor{},
	}

	// Search in groups
	groupQuery := `
		SELECT
			g.id,
			g.prtg_server_address_id,
			g.name,
			g.is_probe_node,
			g.self_group_id,
			gp.path AS full_path,
			g.tree_depth
		FROM prtg_group g
		INNER JOIN prtg_group_path gp ON g.id = gp.group_id
			AND g.prtg_server_address_id = gp.prtg_server_address_id
		WHERE g.name ILIKE $1
		ORDER BY g.name
		LIMIT $2
	`

	groupRows, err := db.Query(ctx, groupQuery, "%"+searchTerm+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("group search failed: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var group types.Group
		var parentID sql.NullInt32

		err := groupRows.Scan(
			&group.ID,
			&group.ServerID,
			&group.Name,
			&group.IsProbeNode,
			&parentID,
			&group.FullPath,
			&group.TreeDepth,
		)
		if err != nil {
			return nil, fmt.Errorf("group scan failed: %w", err)
		}

		if parentID.Valid {
			parentIDInt := int(parentID.Int32)
			group.ParentID = &parentIDInt
		}

		results.Groups = append(results.Groups, group)
	}

	if err := groupRows.Err(); err != nil {
		return nil, err
	}

	// Search in devices
	deviceQuery := `
		SELECT
			d.id,
			d.prtg_server_address_id,
			d.name,
			d.host,
			d.prtg_group_id,
			g.name AS group_name,
			dp.path AS full_path,
			COALESCE(
				(SELECT COUNT(*) FROM prtg_sensor s
				 WHERE s.prtg_device_id = d.id
				 AND s.prtg_server_address_id = d.prtg_server_address_id),
				0
			) AS sensor_count,
			d.tree_depth
		FROM prtg_device d
		INNER JOIN prtg_group g ON d.prtg_group_id = g.id
			AND d.prtg_server_address_id = g.prtg_server_address_id
		INNER JOIN prtg_device_path dp ON d.id = dp.device_id
			AND d.prtg_server_address_id = dp.prtg_server_address_id
		WHERE d.name ILIKE $1 OR d.host ILIKE $1
		ORDER BY d.name
		LIMIT $2
	`

	deviceRows, err := db.Query(ctx, deviceQuery, "%"+searchTerm+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("device search failed: %w", err)
	}
	defer deviceRows.Close()

	for deviceRows.Next() {
		var device types.Device

		err := deviceRows.Scan(
			&device.ID,
			&device.ServerID,
			&device.Name,
			&device.Host,
			&device.GroupID,
			&device.GroupName,
			&device.FullPath,
			&device.SensorCount,
			&device.TreeDepth,
		)
		if err != nil {
			return nil, fmt.Errorf("device scan failed: %w", err)
		}

		results.Devices = append(results.Devices, device)
	}

	if err := deviceRows.Err(); err != nil {
		return nil, err
	}

	// Search in sensors
	sensorQuery := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name AS device_name,
			s.scanning_interval_seconds,
			s.status,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			sp.path AS full_path,
			'' AS tags
		FROM prtg_sensor s
		INNER JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
			AND s.prtg_server_address_id = sp.prtg_server_address_id
		WHERE s.name ILIKE $1 OR s.sensor_type ILIKE $1
		ORDER BY s.name
		LIMIT $2
	`

	sensorRows, err := db.Query(ctx, sensorQuery, "%"+searchTerm+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("sensor search failed: %w", err)
	}
	defer sensorRows.Close()

	sensors, err := scanSensors(sensorRows)
	if err != nil {
		return nil, err
	}

	results.Sensors = sensors

	return results, nil
}

// GetTags retrieves all PRTG tags matching the given filters.
func (db *DB) GetTags(ctx context.Context, tagName string, limit int) ([]types.Tag, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			t.id,
			t.prtg_server_address_id,
			t.name,
			COUNT(DISTINCT st.prtg_sensor_id) as sensor_count
		FROM prtg_tag t
		LEFT JOIN prtg_sensor_tag st ON t.id = st.prtg_tag_id
			AND t.prtg_server_address_id = st.prtg_server_address_id
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if tagName != "" {
		query += fmt.Sprintf(" AND t.name ILIKE $%d", argPos)
		args = append(args, "%"+tagName+"%")
		argPos++
	}

	query += ` GROUP BY t.id, t.prtg_server_address_id, t.name
		ORDER BY t.name`

	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	tags := []types.Tag{}
	for rows.Next() {
		var tag types.Tag

		err := rows.Scan(
			&tag.ID,
			&tag.ServerID,
			&tag.Name,
			&tag.SensorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// GetBusinessProcesses retrieves Business Process sensors from PRTG.
// Business Process sensors are special sensors that aggregate status from multiple source sensors.
func (db *DB) GetBusinessProcesses(ctx context.Context, processName string, status *int, limit int) ([]types.Sensor, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT
			s.id,
			s.prtg_server_address_id,
			s.name,
			s.sensor_type,
			s.prtg_device_id,
			d.name as device_name,
			s.scanning_interval_seconds,
			s.status,
			s.status_text,
			s.last_check_utc,
			s.last_up_utc,
			s.last_down_utc,
			s.priority,
			s.message,
			s.uptime_since_seconds,
			s.downtime_since_seconds,
			s.full_path,
			COALESCE(
				(SELECT STRING_AGG(t.name, ', ' ORDER BY t.name)
				 FROM prtg_sensor_tag st
				 JOIN prtg_tag t ON st.prtg_tag_id = t.id
				 WHERE st.prtg_sensor_id = s.id
				   AND st.prtg_server_address_id = s.prtg_server_address_id),
				''
			) as tags
		FROM prtg_sensor s
		LEFT JOIN prtg_device d ON s.prtg_device_id = d.id
			AND s.prtg_server_address_id = d.prtg_server_address_id
		WHERE s.sensor_type ILIKE '%business%process%'
	`

	args := []interface{}{}
	argPos := 1

	if processName != "" {
		query += fmt.Sprintf(" AND s.name ILIKE $%d", argPos)
		args = append(args, "%"+processName+"%")
		argPos++
	}

	if status != nil {
		query += fmt.Sprintf(" AND s.status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	query += ` ORDER BY s.priority DESC, s.name`

	query += fmt.Sprintf(" LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanSensors(rows)
}

// GetStatistics retrieves aggregated PRTG server statistics.
func (db *DB) GetStatistics(ctx context.Context) (*types.Statistics, error) {
	stats := &types.Statistics{
		SensorsByStatus: make(map[string]int),
		TopSensorTypes:  []types.SensorTypeCount{},
	}

	// Get total counts
	countQuery := `
		SELECT
			(SELECT COUNT(*) FROM prtg_sensor) as total_sensors,
			(SELECT COUNT(*) FROM prtg_device) as total_devices,
			(SELECT COUNT(*) FROM prtg_group) as total_groups,
			(SELECT COUNT(*) FROM prtg_tag) as total_tags,
			(SELECT COUNT(*) FROM prtg_group WHERE is_probe_node = true) as total_probes
	`

	err := db.QueryRow(ctx, countQuery).Scan(
		&stats.TotalSensors,
		&stats.TotalDevices,
		&stats.TotalGroups,
		&stats.TotalTags,
		&stats.TotalProbes,
	)
	if err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	// Calculate average sensors per device
	if stats.TotalDevices > 0 {
		stats.AvgSensorsPerDevice = float64(stats.TotalSensors) / float64(stats.TotalDevices)
	}

	// Get status breakdown
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM prtg_sensor
		GROUP BY status
		ORDER BY status
	`

	statusRows, err := db.Query(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("status query failed: %w", err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var status, count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("status scan failed: %w", err)
		}
		statusText := types.GetStatusText(status)
		stats.SensorsByStatus[statusText] = count
	}

	// Get top sensor types
	typeQuery := `
		SELECT sensor_type, COUNT(*) as count
		FROM prtg_sensor
		WHERE sensor_type IS NOT NULL AND sensor_type != ''
		GROUP BY sensor_type
		ORDER BY count DESC
		LIMIT 15
	`

	typeRows, err := db.Query(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("sensor type query failed: %w", err)
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var sensorType string
		var count int
		if err := typeRows.Scan(&sensorType, &count); err != nil {
			return nil, fmt.Errorf("sensor type scan failed: %w", err)
		}
		stats.TopSensorTypes = append(stats.TopSensorTypes, types.SensorTypeCount{
			Type:  sensorType,
			Count: count,
		})
	}

	return stats, nil
}

// scanSensors is a helper function to scan sensor rows.
func scanSensors(rows *sql.Rows) ([]types.Sensor, error) {
	sensors := []types.Sensor{}

	for rows.Next() {
		var sensor types.Sensor

		var lastCheckUTC, lastDownUTC sql.NullTime

		var uptimeSecs, downtimeSecs sql.NullFloat64

		var message, tags sql.NullString

		err := rows.Scan(
			&sensor.ID,
			&sensor.ServerID,
			&sensor.Name,
			&sensor.SensorType,
			&sensor.DeviceID,
			&sensor.DeviceName,
			&sensor.ScanningIntervalSecs,
			&sensor.Status,
			&lastCheckUTC,
			&sensor.LastUpUTC,
			&lastDownUTC,
			&sensor.Priority,
			&message,
			&uptimeSecs,
			&downtimeSecs,
			&sensor.FullPath,
			&tags,
		)

		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// Handle nullable fields
		if lastCheckUTC.Valid {
			sensor.LastCheckUTC = &lastCheckUTC.Time
		}

		if lastDownUTC.Valid {
			sensor.LastDownUTC = &lastDownUTC.Time
		}

		if uptimeSecs.Valid {
			sensor.UptimeSinceSecs = &uptimeSecs.Float64
		}

		if downtimeSecs.Valid {
			sensor.DowntimeSinceSecs = &downtimeSecs.Float64
		}

		if message.Valid {
			sensor.Message = message.String
		}

		if tags.Valid {
			sensor.Tags = tags.String
		}

		sensor.StatusText = types.GetStatusText(sensor.Status)

		sensors = append(sensors, sensor)
	}

	return sensors, rows.Err()
}
