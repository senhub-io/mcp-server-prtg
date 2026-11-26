package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/senhub-io/mcp-server-prtg/internal/types"
)

var (
	// ErrSensorNotFound is returned when no live sensor has the requested ID.
	ErrSensorNotFound = errors.New("sensor not found")
	// ErrNoChannelData is returned when the database holds no channel history tables.
	ErrNoChannelData = errors.New("channel history not available in this database")
)

const channelQueryTimeout = 15 * time.Second
const maxHistoryBuckets = 2000

// ExportFreshWithin is the age after which an exported sensor is considered stale.
const ExportFreshWithin = 24 * time.Hour

func channelDisplayName(id int, name sql.NullString) (string, bool) {
	if name.Valid && name.String != "" {
		return name.String, false
	}
	return fmt.Sprintf("Channel %d", id), true
}

func (db *DB) resolveServerID(ctx context.Context, sensorID int) (int, error) {
	q := "SELECT s.prtg_server_address_id FROM prtg_sensor s WHERE s.id = $1" + db.notDeleted("s") + " ORDER BY s.prtg_server_address_id LIMIT 1"
	var serverID int
	err := db.QueryRow(ctx, q, sensorID).Scan(&serverID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("sensor %d: %w", sensorID, ErrSensorNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("resolve server for sensor %d: %w", sensorID, err)
	}
	return serverID, nil
}

// GetChannelLastValues returns the newest exported value of each channel of a sensor within maxAge.
func (db *DB) GetChannelLastValues(ctx context.Context, sensorID int, maxAge time.Duration) ([]types.ChannelLastValue, error) {
	if !db.schema.channelValues {
		return nil, ErrNoChannelData
	}
	if sensorID <= 0 {
		return nil, fmt.Errorf("invalid sensor id %d", sensorID)
	}
	if maxAge <= 0 {
		maxAge = 7 * 24 * time.Hour
	}

	ctx, cancel := context.WithTimeout(ctx, channelQueryTimeout)
	defer cancel()

	serverID, err := db.resolveServerID(ctx, sensorID)
	if err != nil {
		return nil, err
	}

	since := time.Now().UTC().Add(-maxAge)
	q := `SELECT DISTINCT ON (v.prtg_channel_id) v.prtg_channel_id, v.value, v.date_time, c.name, c.unit
FROM prtg_channel_value v
LEFT JOIN prtg_channel c ON c.id = v.prtg_channel_id AND c.prtg_server_address_id = v.prtg_server_address_id AND c.prtg_sensor_id = v.prtg_sensor_id
WHERE v.prtg_sensor_id = $1 AND v.prtg_server_address_id = $3 AND v.date_time >= $2
ORDER BY v.prtg_channel_id, v.date_time DESC`
	rows, err := db.Query(ctx, q, sensorID, since, serverID)
	if err != nil {
		return nil, fmt.Errorf("query last channel values for sensor %d: %w", sensorID, err)
	}
	defer rows.Close()

	result := make([]types.ChannelLastValue, 0)
	for rows.Next() {
		var (
			ch   types.ChannelLastValue
			name sql.NullString
			unit sql.NullString
		)
		if err := rows.Scan(&ch.ID, &ch.Value, &ch.Timestamp, &name, &unit); err != nil {
			return nil, fmt.Errorf("scan last channel value: %w", err)
		}
		cleanedName := ""
		if name.Valid {
			cleanedName = cleanText(name.String)
		}
		if cleanedName != "" {
			ch.Name = cleanedName
			ch.NameMissing = false
		} else {
			ch.Name, ch.NameMissing = channelDisplayName(ch.ID, sql.NullString{})
		}
		if unit.Valid {
			ch.Unit = cleanText(unit.String)
		}
		ch.Timestamp = ch.Timestamp.UTC()
		result = append(result, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate last channel values for sensor %d: %w", sensorID, err)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// GetChannelHistory returns per-bucket channel averages of a sensor over [from, to).
func (db *DB) GetChannelHistory(ctx context.Context, sensorID int, from, to time.Time, bucket time.Duration) (*types.ChannelHistory, error) {
	if !db.schema.channelValues {
		return nil, ErrNoChannelData
	}
	if sensorID <= 0 {
		return nil, fmt.Errorf("invalid sensor id %d", sensorID)
	}
	if !from.Before(to) {
		return nil, fmt.Errorf("invalid range: from %s must be before to %s", from, to)
	}
	if bucket < time.Minute {
		return nil, fmt.Errorf("bucket %s must be at least 1 minute", bucket)
	}
	if to.Sub(from)/bucket > maxHistoryBuckets {
		return nil, fmt.Errorf("range %s to %s with bucket %s exceeds %d buckets", from, to, bucket, maxHistoryBuckets)
	}

	ctx, cancel := context.WithTimeout(ctx, channelQueryTimeout)
	defer cancel()

	serverID, err := db.resolveServerID(ctx, sensorID)
	if err != nil {
		return nil, err
	}

	names := make(map[int]types.ChannelInfo)
	rows, err := db.Query(ctx, "SELECT id, name, unit FROM prtg_channel WHERE prtg_sensor_id = $1 AND prtg_server_address_id = $2", sensorID, serverID)
	if err != nil {
		return nil, fmt.Errorf("query channels for sensor %d: %w", sensorID, err)
	}
	for rows.Next() {
		var (
			info types.ChannelInfo
			unit sql.NullString
		)
		if err := rows.Scan(&info.ID, &info.Name, &unit); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan channel info: %w", err)
		}
		info.Name = cleanText(info.Name)
		if unit.Valid {
			info.Unit = cleanText(unit.String)
		}
		names[info.ID] = info
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate channels for sensor %d: %w", sensorID, err)
	}
	rows.Close()

	q := `SELECT v.prtg_channel_id, to_timestamp(floor(extract(epoch FROM v.date_time) / $4) * $4) AS bucket, avg(v.value)
FROM prtg_channel_value v
WHERE v.prtg_sensor_id = $1 AND v.prtg_server_address_id = $5 AND v.date_time >= $2 AND v.date_time < $3
GROUP BY 1, 2 ORDER BY 2, 1`
	rows, err = db.Query(ctx, q, sensorID, from.UTC(), to.UTC(), bucket.Seconds(), serverID)
	if err != nil {
		return nil, fmt.Errorf("query channel history for sensor %d: %w", sensorID, err)
	}
	defer rows.Close()

	history := &types.ChannelHistory{
		SensorID: sensorID,
		ServerID: serverID,
		From:     from.UTC(),
		To:       to.UTC(),
		Bucket:   bucket,
		Channels: make([]types.ChannelInfo, 0),
		Points:   make([]types.ChannelPoint, 0),
	}
	pointIndex := make(map[time.Time]int)
	seen := make(map[int]bool)
	for rows.Next() {
		var (
			channelID int
			ts        time.Time
			avg       float64
		)
		if err := rows.Scan(&channelID, &ts, &avg); err != nil {
			return nil, fmt.Errorf("scan history point: %w", err)
		}
		ts = ts.UTC()
		idx, ok := pointIndex[ts]
		if !ok {
			idx = len(history.Points)
			pointIndex[ts] = idx
			history.Points = append(history.Points, types.ChannelPoint{Timestamp: ts, Values: make(map[int]float64)})
		}
		history.Points[idx].Values[channelID] = avg
		seen[channelID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channel history for sensor %d: %w", sensorID, err)
	}

	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if info, ok := names[id]; ok {
			history.Channels = append(history.Channels, info)
		} else {
			name, missing := channelDisplayName(id, sql.NullString{})
			history.Channels = append(history.Channels, types.ChannelInfo{ID: id, Name: name, NameMissing: missing})
		}
	}
	return history, nil
}

// GetSensorExportState reports how fresh Data Exporter's export is for one sensor.
func (db *DB) GetSensorExportState(ctx context.Context, sensorID int) (*types.SensorExportState, error) {
	if !db.schema.lastFetched {
		return nil, ErrNoChannelData
	}
	if sensorID <= 0 {
		return nil, fmt.Errorf("invalid sensor id %d", sensorID)
	}

	ctx, cancel := context.WithTimeout(ctx, channelQueryTimeout)
	defer cancel()

	serverID, err := db.resolveServerID(ctx, sensorID)
	if err != nil {
		return nil, err
	}

	state := &types.SensorExportState{SensorID: sensorID, ServerID: serverID}
	var lastFetch sql.NullTime
	err = db.QueryRow(ctx, `
		SELECT last_fetch_date_utc, no_monitoring_data
		FROM prtg_sensor_last_fetched_day
		WHERE prtg_sensor_id = $1 AND prtg_server_address_id = $2
	`, sensorID, serverID).Scan(&lastFetch, &state.NoMonitoringData)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query export state for sensor %d: %w", sensorID, err)
	}
	state.Tracked = true
	if lastFetch.Valid {
		t := lastFetch.Time.UTC()
		state.LastFetchUTC = &t
	}
	return state, nil
}
