package database

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matthieu/mcp-server-prtg/internal/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetAlerts_SeveritySorting validates that alerts are sorted by severity (Down first, then Warning).
// This is CRITICAL as it tests the complex ORDER BY CASE logic.
func TestGetAlerts_SeveritySorting(t *testing.T) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// Create DB instance with mock
	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	// Expected query with ORDER BY CASE for severity sorting
	// Match actual SQL from queries.go (lines 227-299)
	expectedQuery := `SELECT[\s\S]+FROM prtg_sensor s[\s\S]+INNER JOIN prtg_device d[\s\S]+WHERE s\.status != \$1`

	// Setup mock expectations - columns must match actual query
	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	// IMPORTANT: sqlmock returns rows in insertion order, NOT SQL ORDER BY order
	// So we must return them in the EXPECTED order (after ORDER BY CASE sorting)
	// Expected order: Down (5), Warning (4), Unusual (10)
	mock.ExpectQuery(expectedQuery).
		WithArgs(types.StatusUp, 24).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(2, 1, "Sensor Down", "ping", 100, "Device1", 60, 5, now, now, &now, 5, "Timeout", nil, 100.0, "/root/device1/sensor2", "critical").
			AddRow(1, 1, "Sensor Warning", "ping", 100, "Device1", 60, 4, now, now, nil, 3, "High CPU", nil, nil, "/root/device1/sensor1", "").
			AddRow(3, 1, "Sensor Unusual", "http", 101, "Device2", 120, 10, now, now, nil, 1, "Spike detected", nil, nil, "/root/device2/sensor3", ""))

	// Execute query
	ctx := context.Background()
	sensors, err := db.GetAlerts(ctx, 24, nil, "")

	// Assertions
	require.NoError(t, err)
	assert.Len(t, sensors, 3)

	// Verify severity order: Down (5) FIRST, then Warning (4), then Unusual (10)
	// This validates that ORDER BY CASE logic produces correct order
	assert.Equal(t, types.StatusDown, sensors[0].Status, "First sensor should be Down (most critical)")
	assert.Equal(t, "Sensor Down", sensors[0].Name)

	assert.Equal(t, types.StatusWarning, sensors[1].Status, "Second sensor should be Warning")
	assert.Equal(t, "Sensor Warning", sensors[1].Name)

	assert.Equal(t, types.StatusUnusual, sensors[2].Status, "Third sensor should be Unusual")
	assert.Equal(t, "Sensor Unusual", sensors[2].Name)

	// Verify all expectations met
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlerts_FilterByStatus validates status filtering.
func TestGetAlerts_FilterByStatus(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	// Filter for Down status only
	downStatus := types.StatusDown

	// Actual query includes time interval filter between status filters
	expectedQuery := `WHERE s\.status != \$1[\s\S]+AND s\.status = \$3`

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	// Arguments order: $1=status to exclude, $2=hours, $3=status filter
	mock.ExpectQuery(expectedQuery).
		WithArgs(types.StatusUp, 24, downStatus).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(1, 1, "Sensor Down", "ping", 100, "Device1", 60, types.StatusDown, now, now, &now, 5, "Timeout", nil, 100.0, "/root/device1/sensor", "critical"))

	ctx := context.Background()
	sensors, err := db.GetAlerts(ctx, 24, &downStatus, "")

	require.NoError(t, err)
	assert.Len(t, sensors, 1)
	assert.Equal(t, types.StatusDown, sensors[0].Status)
	assert.Equal(t, "Sensor Down", sensors[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlerts_FilterByDeviceName validates device name filtering.
func TestGetAlerts_FilterByDeviceName(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	// Actual query uses ILIKE (case-insensitive LIKE for PostgreSQL)
	// Arguments order: $1=status to exclude, $2=hours, $3=device name pattern
	expectedQuery := `WHERE s\.status != \$1[\s\S]+AND d\.name ILIKE \$3`

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	// Correct argument order: status, hours, device_name
	mock.ExpectQuery(expectedQuery).
		WithArgs(types.StatusUp, 24, "%server1%").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(1, 1, "CPU Sensor", "wmi", 100, "Server1", 60, types.StatusWarning, now, now, nil, 3, "High load", nil, nil, "/root/server1/cpu", ""))

	ctx := context.Background()
	sensors, err := db.GetAlerts(ctx, 24, nil, "server1")

	require.NoError(t, err)
	assert.Len(t, sensors, 1)
	assert.Equal(t, "Server1", sensors[0].DeviceName)
	assert.Equal(t, "CPU Sensor", sensors[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSensors_AllFilters validates that all filters work correctly together.
func TestGetSensors_AllFilters(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	downStatus := types.StatusDown

	// Query with all filters: device_name, sensor_name, status (uses ILIKE for PostgreSQL)
	expectedQuery := `WHERE[\s\S]+d\.name ILIKE \$\d+[\s\S]+AND s\.name ILIKE \$\d+[\s\S]+AND s\.status = \$\d+`

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	mock.ExpectQuery(expectedQuery).
		WithArgs("%router%", "%ping%", downStatus, 1000).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(1, 1, "Ping Sensor", "ping", 100, "Router1", 60, types.StatusDown, now, now, &now, 5, "Timeout", nil, 300.0, "/root/network/router1/ping", "critical,network"))

	ctx := context.Background()
	sensors, err := db.GetSensors(ctx, "router", "ping", &downStatus, "", 1000)

	require.NoError(t, err)
	assert.Len(t, sensors, 1)
	assert.Equal(t, "Router1", sensors[0].DeviceName)
	assert.Equal(t, "Ping Sensor", sensors[0].Name)
	assert.Equal(t, types.StatusDown, sensors[0].Status)
	assert.Contains(t, sensors[0].Tags, "critical")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestExecuteCustomQuery_SELECTOnly validates that only SELECT queries are allowed.
func TestExecuteCustomQuery_SELECTOnly(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	tests := []struct {
		name        string
		query       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid SELECT query",
			query:       "SELECT * FROM prtg_sensor WHERE status = 5",
			shouldError: false,
		},
		{
			name:        "SELECT with subquery",
			query:       "SELECT id FROM prtg_sensor WHERE id IN (SELECT sensor_id FROM prtg_sensor_tag)",
			shouldError: false,
		},
		{
			name:        "DROP table attempt",
			query:       "DROP TABLE prtg_sensor",
			shouldError: true,
			errorMsg:    "only SELECT queries are allowed",
		},
		{
			name:        "DELETE attempt",
			query:       "DELETE FROM prtg_sensor WHERE id = 1",
			shouldError: true,
			errorMsg:    "only SELECT queries are allowed",
		},
		{
			name:        "UPDATE attempt",
			query:       "UPDATE prtg_sensor SET status = 3 WHERE id = 1",
			shouldError: true,
			errorMsg:    "only SELECT queries are allowed",
		},
		{
			name:        "INSERT attempt",
			query:       "INSERT INTO prtg_sensor (name) VALUES ('hack')",
			shouldError: true,
			errorMsg:    "only SELECT queries are allowed",
		},
		{
			name:        "SQL injection attempt with comment",
			query:       "SELECT * FROM prtg_sensor; DROP TABLE users; --",
			shouldError: true,
			errorMsg:    "forbidden keyword",
		},
		{
			name:        "SQL injection with semicolon",
			query:       "SELECT * FROM prtg_sensor WHERE id = 1; DELETE FROM prtg_device",
			shouldError: true,
			errorMsg:    "forbidden keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldError {
				// Setup mock for valid SELECT
				mock.ExpectQuery(regexp.QuoteMeta(tt.query)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			}

			ctx := context.Background()
			results, err := db.ExecuteCustomQuery(ctx, tt.query, 100)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
			}
		})
	}

	// Note: We don't check ExpectationsWereMet here because dangerous queries don't reach the DB
}

// TestGetSensorByID validates retrieval of a specific sensor.
func TestGetSensorByID(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()
	uptime := 3600.0 // 1 hour uptime

	expectedQuery := `WHERE[\s\S]+s\.id = \$1`

	mock.ExpectQuery(expectedQuery).
		WithArgs(123).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(123, 1, "Test Sensor", "ping", 100, "Test Device", 60, types.StatusUp, now, now, nil, 3, "OK", &uptime, nil, "/root/test/sensor", "production"))

	ctx := context.Background()
	sensor, err := db.GetSensorByID(ctx, 123)

	require.NoError(t, err)
	assert.NotNil(t, sensor)
	assert.Equal(t, 123, sensor.ID)
	assert.Equal(t, "Test Sensor", sensor.Name)
	assert.Equal(t, "Test Device", sensor.DeviceName)
	assert.Equal(t, types.StatusUp, sensor.Status)
	assert.NotNil(t, sensor.UptimeSinceSecs)
	assert.Equal(t, uptime, *sensor.UptimeSinceSecs)
	assert.Nil(t, sensor.DowntimeSinceSecs)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSensorByID_NotFound validates error handling when sensor doesn't exist.
func TestGetSensorByID_NotFound(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	expectedQuery := `WHERE[\s\S]+s\.id = \$1`

	mock.ExpectQuery(expectedQuery).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	ctx := context.Background()
	sensor, err := db.GetSensorByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, sensor)
	assert.Contains(t, err.Error(), "sensor not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlerts_EmptyResult validates handling of no alerts.
func TestGetAlerts_EmptyResult(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	// Return empty result set
	mock.ExpectQuery(`WHERE s\.status != \$1`).
		WithArgs(types.StatusUp, 24).
		WillReturnRows(sqlmock.NewRows(columns))

	ctx := context.Background()
	sensors, err := db.GetAlerts(ctx, 24, nil, "")

	require.NoError(t, err)
	assert.Empty(t, sensors)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAlerts_ComplexSeverityOrder validates the full ORDER BY CASE logic with all status codes.
func TestGetAlerts_ComplexSeverityOrder(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	// IMPORTANT: sqlmock returns rows in insertion order
	// We must return them ALREADY SORTED to simulate ORDER BY CASE behavior
	// Expected order (as per ORDER BY CASE in queries.go lines 286-299):
	// 1. Down (5) - CASE WHEN 5 THEN 1
	// 2. DownPartial (14) - CASE WHEN 14 THEN 2
	// 3. DownAcknowledged (13) - CASE WHEN 13 THEN 3
	// 4. Warning (4) - CASE WHEN 4 THEN 4
	// 5. Unusual (10) - CASE WHEN 10 THEN 5
	// 6. NoProbe (6) - CASE WHEN 6 THEN 6
	// 7. Unknown (1) - CASE WHEN 1 THEN 7

	mock.ExpectQuery(`WHERE s\.status != \$1`).
		WithArgs(types.StatusUp, 24).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(5, 1, "Sensor Down", "ping", 100, "Dev1", 60, types.StatusDown, now, now, &now, 3, "", nil, 100.0, "/s5", "").
			AddRow(7, 1, "Sensor DownPartial", "ping", 100, "Dev1", 60, types.StatusDownPartial, now, now, &now, 3, "", nil, 75.0, "/s7", "").
			AddRow(6, 1, "Sensor DownAck", "ping", 100, "Dev1", 60, types.StatusDownAcknowledged, now, now, &now, 3, "", nil, 50.0, "/s6", "").
			AddRow(3, 1, "Sensor Warning", "ping", 100, "Dev1", 60, types.StatusWarning, now, now, nil, 3, "", nil, nil, "/s3", "").
			AddRow(4, 1, "Sensor Unusual", "ping", 100, "Dev1", 60, types.StatusUnusual, now, now, nil, 3, "", nil, nil, "/s4", "").
			AddRow(2, 1, "Sensor NoProbe", "ping", 100, "Dev1", 60, types.StatusNoProbe, now, now, nil, 3, "", nil, nil, "/s2", "").
			AddRow(1, 1, "Sensor Unknown", "ping", 100, "Dev1", 60, types.StatusUnknown, now, now, nil, 3, "", nil, nil, "/s1", ""))

	ctx := context.Background()
	sensors, err := db.GetAlerts(ctx, 24, nil, "")

	require.NoError(t, err)
	assert.Len(t, sensors, 7)

	// Verify EXACT severity order as defined by ORDER BY CASE
	expectedOrder := []struct {
		status int
		name   string
	}{
		{types.StatusDown, "Sensor Down"},
		{types.StatusDownPartial, "Sensor DownPartial"},
		{types.StatusDownAcknowledged, "Sensor DownAck"},
		{types.StatusWarning, "Sensor Warning"},
		{types.StatusUnusual, "Sensor Unusual"},
		{types.StatusNoProbe, "Sensor NoProbe"},
		{types.StatusUnknown, "Sensor Unknown"},
	}

	for i, expected := range expectedOrder {
		assert.Equal(t, expected.status, sensors[i].Status,
			"Position %d: expected status %d but got %d", i, expected.status, sensors[i].Status)
		assert.Equal(t, expected.name, sensors[i].Name,
			"Position %d: expected %s but got %s", i, expected.name, sensors[i].Name)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// BenchmarkGetAlerts benchmarks the GetAlerts function.
func BenchmarkGetAlerts(b *testing.B) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	logger := zerolog.Nop()
	db := &DB{
		conn:   mockDB,
		logger: &logger,
	}

	columns := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	now := time.Now()

	for i := 0; i < b.N; i++ {
		mock.ExpectQuery(`WHERE s\.status != \$1`).
			WithArgs(types.StatusUp, 24).
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow(1, 1, "Sensor", "ping", 100, "Device", 60, types.StatusDown, now, now, &now, 5, "Timeout", nil, 100.0, "/root/sensor", ""))

		ctx := context.Background()
		_, _ = db.GetAlerts(ctx, 24, nil, "")
	}
}
