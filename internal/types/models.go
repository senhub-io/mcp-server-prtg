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

// DeviceOverview represents a device with its sensors and aggregated statistics.
// Used by the prtg_device_overview MCP tool to provide a complete device status summary.
type DeviceOverview struct {
	Device       Device   `json:"device"`
	Sensors      []Sensor `json:"sensors"`
	TotalSensors int      `json:"total_sensors"`
	UpSensors    int      `json:"up_sensors"`
	DownSensors  int      `json:"down_sensors"`
	WarnSensors  int      `json:"warning_sensors"`
}

// HierarchyNode represents a node in the PRTG hierarchy tree.
// Used by the prtg_get_hierarchy MCP tool to navigate the PRTG structure.
type HierarchyNode struct {
	Group   Group             `json:"group"`
	Devices []HierarchyDevice `json:"devices"`
	Groups  []*HierarchyNode  `json:"groups,omitempty"`
}

// HierarchyDevice represents a device with its sensors in the hierarchy.
type HierarchyDevice struct {
	Device  Device   `json:"device"`
	Sensors []Sensor `json:"sensors,omitempty"`
}

// SearchResults represents the results of a universal search across PRTG objects.
// Used by the prtg_search MCP tool.
type SearchResults struct {
	Groups  []Group  `json:"groups"`
	Devices []Device `json:"devices"`
	Sensors []Sensor `json:"sensors"`
}

// SensorStatus represents PRTG sensor status values.
// Official PRTG status codes from documentation.
const (
	StatusUnknown            = 1
	StatusCollecting         = 2
	StatusUp                 = 3
	StatusWarning            = 4
	StatusDown               = 5
	StatusNoProbe            = 6
	StatusPausedByUser       = 7
	StatusPausedByDependency = 8
	StatusPausedBySchedule   = 9
	StatusUnusual            = 10
	StatusPausedByLicense    = 11
	StatusPausedUntil        = 12
	StatusDownAcknowledged   = 13
	StatusDownPartial        = 14
)

// GetStatusText returns the human-readable name for a PRTG status code (1-14).
// Returns "Unknown" for invalid status codes.
func GetStatusText(status int) string {
	switch status {
	case StatusUnknown:
		return "Unknown"
	case StatusCollecting:
		return "Collecting"
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
	case StatusPausedUntil:
		return "Paused Until"
	case StatusDownAcknowledged:
		return "Down (Acknowledged)"
	case StatusDownPartial:
		return "Down (Partial)"
	default:
		return "Unknown"
	}
}
