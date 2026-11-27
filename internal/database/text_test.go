package database

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"backslash entity", `C:&#92; Label`, `C:\ Label`},
		{"ampersand", `R&amp;D`, `R&D`},
		{"angle brackets", `&lt;tag&gt;`, `<tag>`},
		{"quotes", `&quot;hi&quot; &#39;yo&#39;`, `"hi" 'yo'`},
		{"plain text", "no entities", "no entities"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cleanText(tt.input))
		})
	}
}

func TestGetSensorByIDDecodesHTMLEntities(t *testing.T) {
	db, mock := newMockDB(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}).AddRow(
		1, 1, `Disk Free: C:&#92; Label`, "diskfree", 10,
		`R&amp;D Server`, 60, 3, time.Now(), time.Now(), nil, 3,
		"OK", 100.0, 0.0, `/Root&#92;Group`, "tag1&amp;tag2",
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sensor, err := db.GetSensorByID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, `Disk Free: C:\ Label`, sensor.Name)
	assert.Equal(t, "R&D Server", sensor.DeviceName)
	assert.Equal(t, `/Root\Group`, sensor.FullPath)
	assert.Equal(t, "tag1&tag2", sensor.Tags)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAlertsExcludesPausedWithoutFilter(t *testing.T) {
	db, mock := newMockDB(t)
	ctx := context.Background()

	cols := []string{
		"id", "prtg_server_address_id", "name", "sensor_type", "prtg_device_id",
		"device_name", "scanning_interval_seconds", "status", "last_check_utc",
		"last_up_utc", "last_down_utc", "priority", "message",
		"uptime_since_seconds", "downtime_since_seconds", "full_path", "tags",
	}

	mock.ExpectQuery(`NOT IN \(7, 8, 9, 11, 12\)`).
		WithArgs(3).
		WillReturnRows(sqlmock.NewRows(cols))

	_, err := db.GetAlerts(ctx, 0, nil, "")
	require.NoError(t, err)

	mock.ExpectQuery(`WHERE s\.status != \$1 AND s\.status = \$2`).
		WithArgs(3, 5).
		WillReturnRows(sqlmock.NewRows(cols))

	status := 5
	_, err = db.GetAlerts(ctx, 0, &status, "")
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelLastValuesDecodesName(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true
	ctx := context.Background()

	mock.ExpectQuery(`SELECT s\.prtg_server_address_id FROM prtg_sensor`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(1))

	mock.ExpectQuery(`SELECT DISTINCT ON`).
		WithArgs(1, sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_channel_id", "value", "date_time", "name", "unit"}).
			AddRow(1, 42.5, time.Now(), `Traffic &amp; Load`, "%"))

	result, err := db.GetChannelLastValues(ctx, 1, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Traffic & Load", result[0].Name)
	assert.Equal(t, "%", result[0].Unit)
	require.NoError(t, mock.ExpectationsWereMet())
}
