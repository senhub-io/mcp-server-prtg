package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

func TestGetSensorExportStateFeatureOff(t *testing.T) {
	db, mock := newMockDB(t)

	state, err := db.GetSensorExportState(context.Background(), 42)
	require.ErrorIs(t, err, ErrNoChannelData)
	assert.Nil(t, state)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSensorExportStateTracked(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{lastFetched: true})

	lastFetch := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(`SELECT s\.prtg_server_address_id FROM prtg_sensor`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(7))
	mock.ExpectQuery("prtg_sensor_last_fetched_day").
		WithArgs(42, 7).
		WillReturnRows(sqlmock.NewRows([]string{"last_fetch_date_utc", "no_monitoring_data"}).AddRow(lastFetch, false))

	state, err := db.GetSensorExportState(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, 42, state.SensorID)
	assert.Equal(t, 7, state.ServerID)
	assert.True(t, state.Tracked)
	require.NotNil(t, state.LastFetchUTC)
	assert.True(t, lastFetch.Equal(*state.LastFetchUTC))
	assert.False(t, state.NoMonitoringData)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSensorExportStateUntracked(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{lastFetched: true})

	mock.ExpectQuery(`SELECT s\.prtg_server_address_id FROM prtg_sensor`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(7))
	mock.ExpectQuery("prtg_sensor_last_fetched_day").
		WithArgs(42, 7).
		WillReturnError(sql.ErrNoRows)

	state, err := db.GetSensorExportState(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.False(t, state.Tracked)
	assert.Nil(t, state.LastFetchUTC)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSensorExportStateSensorNotFound(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{lastFetched: true})

	mock.ExpectQuery(`SELECT s\.prtg_server_address_id FROM prtg_sensor`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	state, err := db.GetSensorExportState(context.Background(), 999)
	require.ErrorIs(t, err, ErrSensorNotFound)
	assert.Nil(t, state)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelLastValuesNameMissing(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{channelValues: true})

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(`SELECT s\.prtg_server_address_id FROM prtg_sensor`).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(7))
	mock.ExpectQuery("DISTINCT ON").
		WithArgs(42, sqlmock.AnyArg(), 7).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_channel_id", "value", "date_time", "name", "unit"}).
			AddRow(5, 1.5, now, nil, nil).
			AddRow(6, 2.5, now, "Traffic", "%"))

	values, err := db.GetChannelLastValues(context.Background(), 42, time.Hour)
	require.NoError(t, err)
	require.Len(t, values, 2)

	byID := make(map[int]types.ChannelLastValue, len(values))
	for _, v := range values {
		byID[v.ID] = v
	}
	assert.Equal(t, "Channel 5", byID[5].Name)
	assert.True(t, byID[5].NameMissing)
	assert.Equal(t, "Traffic", byID[6].Name)
	assert.False(t, byID[6].NameMissing)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatisticsProfilesAndCoverage(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{collectionStatus: true, lastFetched: true})

	now := time.Now().UTC().Truncate(time.Second)
	next := now.Add(time.Hour)

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"total_sensors", "total_devices", "total_groups", "total_tags", "total_probes"}).
			AddRow(100, 10, 2, 3, 1),
	)
	mock.ExpectQuery(`SELECT s\.status`).WillReturnRows(
		sqlmock.NewRows([]string{"status", "count"}).AddRow(3, 100),
	)
	mock.ExpectQuery(`SELECT s\.sensor_type`).WillReturnRows(
		sqlmock.NewRows([]string{"sensor_type", "count"}).AddRow("ping", 100),
	)
	mock.ExpectQuery("data_collection_status").WillReturnRows(
		sqlmock.NewRows([]string{"profile_alias", "profile_type", "last_collection_end_date", "next_scheduled_collection", "has_errors", "has_warnings", "is_paused"}).
			AddRow("metrics", 0, now, next, false, true, true).
			AddRow("status", 1, now.Add(-time.Minute), nil, true, false, false),
	)
	mock.ExpectQuery("prtg_sensor_last_fetched_day").WithArgs(sqlmock.AnyArg()).WillReturnRows(
		sqlmock.NewRows([]string{"up_to_date", "stale", "never_fetched", "no_data", "not_tracked"}).
			AddRow(647, 3477, 52, 10, 11),
	)

	stats, err := db.GetStatistics(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats.LastCollectionUTC)
	assert.True(t, now.Equal(*stats.LastCollectionUTC))
	assert.True(t, stats.CollectionErrors)

	require.Len(t, stats.Profiles, 2)
	assert.Equal(t, "metrics", stats.Profiles[0].Alias)
	assert.Equal(t, "metrics", stats.Profiles[0].Kind)
	assert.True(t, stats.Profiles[0].Paused)
	assert.True(t, stats.Profiles[0].HasWarnings)
	require.NotNil(t, stats.Profiles[0].NextUTC)
	assert.Equal(t, "status", stats.Profiles[1].Kind)
	assert.True(t, stats.Profiles[1].HasErrors)

	require.NotNil(t, stats.ExportCoverage)
	assert.Equal(t, 647, stats.ExportCoverage.UpToDate)
	assert.Equal(t, 3477, stats.ExportCoverage.Stale)
	assert.Equal(t, 52, stats.ExportCoverage.NeverFetched)
	assert.Equal(t, 10, stats.ExportCoverage.NoData)
	assert.Equal(t, 11, stats.ExportCoverage.NotTracked)
	assert.Equal(t, 24, stats.ExportCoverage.FreshWithinHours)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatisticsOptionalQueriesFail(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{collectionStatus: true, lastFetched: true})

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"total_sensors", "total_devices", "total_groups", "total_tags", "total_probes"}).
			AddRow(100, 10, 2, 3, 1),
	)
	mock.ExpectQuery(`SELECT s\.status`).WillReturnRows(
		sqlmock.NewRows([]string{"status", "count"}).AddRow(3, 100),
	)
	mock.ExpectQuery(`SELECT s\.sensor_type`).WillReturnRows(
		sqlmock.NewRows([]string{"sensor_type", "count"}).AddRow("ping", 100),
	)
	mock.ExpectQuery("data_collection_status").WillReturnError(errors.New("boom"))
	mock.ExpectQuery("prtg_sensor_last_fetched_day").WillReturnError(errors.New("boom"))

	stats, err := db.GetStatistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 100, stats.TotalSensors)
	assert.Nil(t, stats.LastCollectionUTC)
	assert.False(t, stats.CollectionErrors)
	assert.Empty(t, stats.Profiles)
	assert.Nil(t, stats.ExportCoverage)
	assert.NoError(t, mock.ExpectationsWereMet())
}
