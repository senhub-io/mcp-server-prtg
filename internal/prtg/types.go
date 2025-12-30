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
	ObjectID   int                   `json:"object_id"`
	TimeType   TimeSeriesType        `json:"time_type,omitempty"`
	StartTime  *time.Time            `json:"start_time,omitempty"`
	EndTime    *time.Time            `json:"end_time,omitempty"`
	Headers    []string              `json:"headers"`
	DataPoints []TimeSeriesDataPoint `json:"data_points"`
}

// TimeSeriesDataPoint represents a single data point in time series.
type TimeSeriesDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"` // Channel name -> value
}

// Channel represents a PRTG channel (sensor measurement) from API v2.
type Channel struct {
	ID              string              `json:"id"` // e.g., "1002.0"
	Name            string              `json:"name"`
	Kind            string              `json:"kind"`
	Type            string              `json:"type"`
	Basic           ChannelBasic        `json:"basic"`
	LastMeasurement *ChannelMeasurement `json:"last_measurement,omitempty"`
	Parent          *ChannelParent      `json:"parent,omitempty"`
}

// ChannelBasic contains basic channel information.
type ChannelBasic struct {
	DisplayUnit string `json:"displayunit"`
	UnitType    string `json:"unittype"`
	Name        string `json:"name"`
}

// ChannelMeasurement represents the last measurement of a channel.
type ChannelMeasurement struct {
	Timestamp    string  `json:"timestamp"`
	Value        float64 `json:"value"`
	DisplayValue float64 `json:"display_value"`
	Minimum      float64 `json:"minimum"`
	Maximum      float64 `json:"maximum"`
	Average      float64 `json:"average"`
}

// ChannelParent represents the parent sensor of a channel.
type ChannelParent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
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

// Sensor represents a PRTG sensor from API v2.
type Sensor struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Status   SensorStatus   `json:"status"`
	Priority int            `json:"priority,omitempty"`
	Message  string         `json:"message,omitempty"`
	Parent   *ObjectParent  `json:"parent,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Position int            `json:"position,omitempty"`
	Settings *SensorSetting `json:"settings,omitempty"`
}

// SensorStatus represents the status of a sensor.
type SensorStatus struct {
	Code  int    `json:"code"`  // 1=Unknown, 2=Collecting, 3=Up, 4=Warning, 5=Down, etc.
	Name  string `json:"name"`  // "Up", "Down", "Warning", etc.
	Since string `json:"since"` // ISO 8601 timestamp
}

// SensorSetting contains sensor settings.
type SensorSetting struct {
	Interval     int    `json:"interval,omitempty"`
	IntervalText string `json:"intervaltext,omitempty"`
}

// Device represents a PRTG device from API v2.
type Device struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Host     string          `json:"host,omitempty"`
	Status   SensorStatus    `json:"status"`
	Parent   *ObjectParent   `json:"parent,omitempty"`
	Position int             `json:"position,omitempty"`
	Summary  *StatusSummary  `json:"sensor_status_summary,omitempty"`
}

// Group represents a PRTG group from API v2.
type Group struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Parent   *ObjectParent `json:"parent,omitempty"`
	Position int           `json:"position,omitempty"`
}

// ObjectParent represents the parent of an object.
type ObjectParent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// StatusSummary represents a summary of sensor statuses.
type StatusSummary struct {
	Up                int `json:"up,omitempty"`
	Down              int `json:"down,omitempty"`
	Warning           int `json:"warning,omitempty"`
	Paused            int `json:"paused,omitempty"`
	Unusual           int `json:"unusual,omitempty"`
	Unknown           int `json:"unknown,omitempty"`
	DownAcknowledged  int `json:"down_acknowledged,omitempty"`
	DownPartial       int `json:"down_partial,omitempty"`
	Total             int `json:"total,omitempty"`
}

// PRTGObject represents a generic PRTG object (sensor, device, group, probe).
type PRTGObject struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Type     string        `json:"type"` // "sensor", "device", "group", "probe"
	Status   *SensorStatus `json:"status,omitempty"`
	Parent   *ObjectParent `json:"parent,omitempty"`
	Position int           `json:"position,omitempty"`
}

// Tag represents a PRTG tag.
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count,omitempty"` // Number of sensors with this tag
}

// SensorCount represents object counts by type.
type SensorCount struct {
	Total   int `json:"total"`
	Sensors int `json:"sensors,omitempty"`
	Devices int `json:"devices,omitempty"`
	Groups  int `json:"groups,omitempty"`
	Probes  int `json:"probes,omitempty"`
}
