package prtg

import "time"

// TimeSeriesType represents the predefined time periods for PRTG time series data.
type TimeSeriesType string

const (
	// TimeSeriesLive represents live data (last few minutes).
	TimeSeriesLive TimeSeriesType = "live"
	// TimeSeriesShort represents short-term data (typically last 24 hours).
	TimeSeriesShort TimeSeriesType = "short"
	// TimeSeriesMedium represents medium-term data (typically last 7 days).
	TimeSeriesMedium TimeSeriesType = "medium"
	// TimeSeriesLong represents long-term data (typically last 30+ days).
	TimeSeriesLong TimeSeriesType = "long"
)

// TimeSeriesResponse represents the response from the PRTG time series API.
// The API returns data as a table with headers (column names) and rows (data points).
type TimeSeriesResponse struct {
	Headers []string        `json:"headers"` // Column names: ["timestamp", "channel1", "channel2", ...]
	Data    [][]interface{} `json:"data"`    // Data rows: [[timestamp, value1, value2, ...], ...]
}

// TimeSeriesData represents parsed time series data with typed values.
type TimeSeriesData struct {
	ObjectID  int                      `json:"object_id"`
	TimeType  TimeSeriesType           `json:"time_type,omitempty"`
	StartTime *time.Time               `json:"start_time,omitempty"`
	EndTime   *time.Time               `json:"end_time,omitempty"`
	Headers   []string                 `json:"headers"`
	DataPoints []TimeSeriesDataPoint   `json:"data_points"`
}

// TimeSeriesDataPoint represents a single data point in time series.
type TimeSeriesDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"` // Channel name -> value
}

// Channel represents a PRTG channel (sensor measurement).
type Channel struct {
	ID          int    `json:"id"`
	ObjectID    int    `json:"objid"`    // Sensor ID this channel belongs to
	Name        string `json:"name"`
	LastValue   string `json:"lastvalue"`
	LastMessage string `json:"lastmessage"`
	Unit        string `json:"unit"`
}

// ChannelData represents current data for a channel.
type ChannelData struct {
	ChannelID   int       `json:"channel_id"`
	SensorID    int       `json:"sensor_id"`
	Name        string    `json:"name"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
}

// ChannelsResponse represents the response from the /channels endpoint.
type ChannelsResponse struct {
	Channels []Channel `json:"channels"`
}
