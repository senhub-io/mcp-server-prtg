package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// GetSensors retrieves sensors with optional filters
func (db *DB) GetSensors(ctx context.Context, deviceName, sensorName string, status *int, tags string, limit int) ([]types.Sensor, error) {
	// Simplified query without tags subquery for performance
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

	if status != nil {
		query += fmt.Sprintf(" AND s.status = $%d", argPos)

		args = append(args, *status)
		argPos++
	}

	// Tags filter temporarily disabled for performance
	// TODO: Re-enable with proper indexing
	_ = tags

	query += " ORDER BY s.name"

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

// GetSensorByID retrieves a single sensor by ID
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

// GetAlerts retrieves sensors in alert state (non-UP status)
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

	query += " ORDER BY s.priority DESC, s.status, s.name LIMIT 100"

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanSensors(rows)
}

// GetDeviceOverview retrieves a device with all its sensors
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

// GetTopSensors retrieves top sensors by various metrics
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

// ExecuteCustomQuery executes a custom SQL query (SELECT only)
// WARNING: This function accepts raw SQL and should be used with extreme caution.
// It is recommended to disable this in production environments.
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

// scanGenericResults scans generic SQL query results into maps
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

// scanSensors is a helper function to scan sensor rows
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
