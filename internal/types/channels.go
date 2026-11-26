package types

import "time"

// ChannelInfo describes one channel of a sensor.
type ChannelInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Unit        string `json:"unit,omitempty"`
	NameMissing bool   `json:"name_missing,omitempty"`
}

// ChannelLastValue is the most recent exported value of a channel.
type ChannelLastValue struct {
	ChannelInfo
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// ChannelPoint holds the per-channel average of one time bucket, keyed by channel ID.
type ChannelPoint struct {
	Timestamp time.Time       `json:"timestamp"`
	Values    map[int]float64 `json:"values"`
}

// ChannelHistory is the bucketed history of a sensor's channels.
type ChannelHistory struct {
	SensorID int            `json:"sensor_id"`
	ServerID int            `json:"server_id"`
	From     time.Time      `json:"from"`
	To       time.Time      `json:"to"`
	Bucket   time.Duration  `json:"bucket"`
	Channels []ChannelInfo  `json:"channels"`
	Points   []ChannelPoint `json:"points"`
}

// SensorExportState describes how far Data Exporter has exported a sensor's channel history.
type SensorExportState struct {
	SensorID         int        `json:"sensor_id"`
	ServerID         int        `json:"server_id"`
	Tracked          bool       `json:"tracked"` // a prtg_sensor_last_fetched_day row exists
	LastFetchUTC     *time.Time `json:"last_fetch_utc,omitempty"`
	NoMonitoringData bool       `json:"no_monitoring_data,omitempty"`
}
