package types

import "time"

// Sensor represents a PRTG sensor with its metadata and current status.
type Sensor struct {
	ID                   int        `json:"id"`
	ServerID             int        `json:"server_id"`
	Name                 string     `json:"name"`
	SensorType           string     `json:"sensor_type"`
	DeviceID             int        `json:"device_id"`
	DeviceName           string     `json:"device_name,omitempty"`
	ScanningIntervalSecs int        `json:"scanning_interval_seconds"`
	Status               int        `json:"status"`
	StatusText           string     `json:"status_text"`
	LastCheckUTC         *time.Time `json:"last_check_utc,omitempty"`
	LastUpUTC            time.Time  `json:"last_up_utc"`
	LastDownUTC          *time.Time `json:"last_down_utc,omitempty"`
	Priority             int        `json:"priority"`
	Message              string     `json:"message,omitempty"`
	UptimeSinceSecs      *float64   `json:"uptime_since_seconds,omitempty"`
	DowntimeSinceSecs    *float64   `json:"downtime_since_seconds,omitempty"`
	FullPath             string     `json:"full_path,omitempty"`
	Tags                 string     `json:"tags,omitempty"`
}

// Device represents a PRTG device.
type Device struct {
	ID          int    `json:"id"`
	ServerID    int    `json:"server_id"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	GroupID     int    `json:"group_id"`
	GroupName   string `json:"group_name,omitempty"`
	FullPath    string `json:"full_path,omitempty"`
	SensorCount int    `json:"sensor_count"`
	TreeDepth   int    `json:"tree_depth"`
}

// Group represents a PRTG group/probe.
type Group struct {
	ID          int    `json:"id"`
	ServerID    int    `json:"server_id"`
	Name        string `json:"name"`
	IsProbeNode bool   `json:"is_probe_node"`
	ParentID    *int   `json:"parent_id,omitempty"`
	FullPath    string `json:"full_path,omitempty"`
	TreeDepth   int    `json:"tree_depth"`
}

// DeviceOverview represents a complete device view with its sensors.
type DeviceOverview struct {
	Device       Device   `json:"device"`
	Sensors      []Sensor `json:"sensors"`
	TotalSensors int      `json:"total_sensors"`
	UpSensors    int      `json:"up_sensors"`
	DownSensors  int      `json:"down_sensors"`
	WarnSensors  int      `json:"warning_sensors"`
}

// SensorStatus represents common PRTG sensor status values.
const (
	StatusUnknown            = 0
	StatusUp                 = 3
	StatusWarning            = 4
	StatusDown               = 5
	StatusNoProbe            = 6
	StatusPausedByUser       = 7
	StatusPausedByDependency = 8
	StatusPausedBySchedule   = 9
	StatusUnusual            = 10
	StatusPausedByLicense    = 11
)

// GetStatusText returns human-readable status text.
func GetStatusText(status int) string {
	switch status {
	case StatusUp:
		return "Up"
	case StatusWarning:
		return "Warning"
	case StatusDown:
		return "Down"
	case StatusNoProbe:
		return "No Probe"
	case StatusPausedByUser:
		return "Paused (User)"
	case StatusPausedByDependency:
		return "Paused (Dependency)"
	case StatusPausedBySchedule:
		return "Paused (Schedule)"
	case StatusUnusual:
		return "Unusual"
	case StatusPausedByLicense:
		return "Paused (License)"
	default:
		return "Unknown"
	}
}
