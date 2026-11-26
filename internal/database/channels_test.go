package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/senhub-io/mcp-server-prtg/internal/types"
)

func TestGetChannelLastValues_FeatureOff(t *testing.T) {
	db, mock := newMockDB(t)

	_, err := db.GetChannelLastValues(context.Background(), 1, time.Hour)
	assert.ErrorIs(t, err, ErrNoChannelData)

	_, err = db.GetChannelHistory(context.Background(), 1, time.Now(), time.Now().Add(time.Hour), time.Hour)
	assert.ErrorIs(t, err, ErrNoChannelData)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelHistory_SensorNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true

	mock.ExpectQuery("SELECT s.prtg_server_address_id FROM prtg_sensor").
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}))

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := db.GetChannelHistory(context.Background(), 42, from, from.Add(time.Hour), time.Hour)
	assert.ErrorIs(t, err, ErrSensorNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelLastValues(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true

	ts := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"prtg_channel_id", "value", "date_time", "name", "unit"}).
		AddRow(2, 5.5, ts, "beta", "ms").
		AddRow(1, 1.0, ts, nil, nil).
		AddRow(3, 9.0, ts, "alpha", nil)
	mock.ExpectQuery("SELECT s.prtg_server_address_id FROM prtg_sensor").
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(4))
	mock.ExpectQuery("SELECT DISTINCT ON").
		WithArgs(7, sqlmock.AnyArg(), 4).
		WillReturnRows(rows)

	result, err := db.GetChannelLastValues(context.Background(), 7, time.Hour)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, types.ChannelLastValue{ChannelInfo: types.ChannelInfo{ID: 1, Name: "Channel 1", NameMissing: true}, Value: 1.0, Timestamp: ts}, result[0])
	assert.Equal(t, 3, result[1].ID)
	assert.Equal(t, "alpha", result[1].Name)
	assert.Equal(t, 2, result[2].ID)
	assert.Equal(t, "beta", result[2].Name)
	assert.Equal(t, "ms", result[2].Unit)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelHistory(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Hour)
	bucket := time.Hour
	b0 := from
	b1 := from.Add(time.Hour)

	mock.ExpectQuery("SELECT s.prtg_server_address_id FROM prtg_sensor").
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_server_address_id"}).AddRow(3))
	mock.ExpectQuery("SELECT id, name, unit FROM prtg_channel").
		WithArgs(7, 3).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "unit"}).
			AddRow(1, "Traffic", "GB").
			AddRow(2, "Errors", nil))
	mock.ExpectQuery("FROM prtg_channel_value").
		WithArgs(7, sqlmock.AnyArg(), sqlmock.AnyArg(), bucket.Seconds(), 3).
		WillReturnRows(sqlmock.NewRows([]string{"prtg_channel_id", "bucket", "avg"}).
			AddRow(1, b0, 10.0).
			AddRow(2, b0, 20.0).
			AddRow(1, b1, 11.0).
			AddRow(2, b1, 21.0))

	history, err := db.GetChannelHistory(context.Background(), 7, from, to, bucket)
	require.NoError(t, err)
	require.NotNil(t, history)
	assert.Equal(t, 7, history.SensorID)
	assert.Equal(t, 3, history.ServerID)
	assert.Equal(t, from, history.From)
	assert.Equal(t, to, history.To)
	assert.Equal(t, bucket, history.Bucket)

	require.Len(t, history.Channels, 2)
	assert.Equal(t, types.ChannelInfo{ID: 1, Name: "Traffic", Unit: "GB"}, history.Channels[0])
	assert.Equal(t, types.ChannelInfo{ID: 2, Name: "Errors"}, history.Channels[1])

	require.Len(t, history.Points, 2)
	assert.Equal(t, b0, history.Points[0].Timestamp)
	assert.Equal(t, map[int]float64{1: 10.0, 2: 20.0}, history.Points[0].Values)
	assert.Equal(t, b1, history.Points[1].Timestamp)
	assert.Equal(t, map[int]float64{1: 11.0, 2: 21.0}, history.Points[1].Values)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelHistory_BucketLimit(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := db.GetChannelHistory(context.Background(), 7, from, from.Add(2001*time.Hour), time.Hour)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrSensorNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChannelHistory_InvalidRange(t *testing.T) {
	db, mock := newMockDB(t)
	db.schema.channelValues = true

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := db.GetChannelHistory(context.Background(), 7, from, from, time.Hour)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrSensorNotFound))

	_, err = db.GetChannelHistory(context.Background(), 7, from, from.Add(time.Hour), 30*time.Second)
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}
