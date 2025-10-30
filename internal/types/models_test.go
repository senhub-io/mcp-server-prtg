package types

import (
	"testing"
)

// TestStatusConstants validates that all PRTG status constants match official documentation.
// Official PRTG status codes: https://www.paessler.com/manuals/prtg/object_states
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant int
		expected int
	}{
		{"StatusUnknown", StatusUnknown, 1},
		{"StatusCollecting", StatusCollecting, 2},
		{"StatusUp", StatusUp, 3},
		{"StatusWarning", StatusWarning, 4},
		{"StatusDown", StatusDown, 5},
		{"StatusNoProbe", StatusNoProbe, 6},
		{"StatusPausedByUser", StatusPausedByUser, 7},
		{"StatusPausedByDependency", StatusPausedByDependency, 8},
		{"StatusPausedBySchedule", StatusPausedBySchedule, 9},
		{"StatusUnusual", StatusUnusual, 10},
		{"StatusPausedByLicense", StatusPausedByLicense, 11},
		{"StatusPausedUntil", StatusPausedUntil, 12},
		{"StatusDownAcknowledged", StatusDownAcknowledged, 13},
		{"StatusDownPartial", StatusDownPartial, 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestGetStatusText validates status text mapping for all PRTG status codes.
func TestGetStatusText(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{"Unknown status", StatusUnknown, "Unknown"},
		{"Collecting status", StatusCollecting, "Collecting"},
		{"Up status", StatusUp, "Up"},
		{"Warning status", StatusWarning, "Warning"},
		{"Down status", StatusDown, "Down"},
		{"No Probe status", StatusNoProbe, "No Probe"},
		{"Paused by User", StatusPausedByUser, "Paused (User)"},
		{"Paused by Dependency", StatusPausedByDependency, "Paused (Dependency)"},
		{"Paused by Schedule", StatusPausedBySchedule, "Paused (Schedule)"},
		{"Unusual status", StatusUnusual, "Unusual"},
		{"Paused by License", StatusPausedByLicense, "Paused (License)"},
		{"Paused Until", StatusPausedUntil, "Paused Until"},
		{"Down Acknowledged", StatusDownAcknowledged, "Down (Acknowledged)"},
		{"Down Partial", StatusDownPartial, "Down (Partial)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStatusText(tt.status)
			if got != tt.expected {
				t.Errorf("GetStatusText(%d) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

// TestGetStatusText_EdgeCases tests edge cases for GetStatusText function.
func TestGetStatusText_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected string
	}{
		{"Zero status", 0, "Unknown"},
		{"Negative status", -1, "Unknown"},
		{"Invalid status 15", 15, "Unknown"},
		{"Invalid status 100", 100, "Unknown"},
		{"Invalid status 999", 999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStatusText(tt.status)
			if got != tt.expected {
				t.Errorf("GetStatusText(%d) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

// TestGetStatusText_AllValidCodes ensures all valid PRTG codes (1-14) return non-empty strings.
func TestGetStatusText_AllValidCodes(t *testing.T) {
	for status := 1; status <= 14; status++ {
		t.Run("Status code", func(t *testing.T) {
			text := GetStatusText(status)
			if text == "" {
				t.Errorf("GetStatusText(%d) returned empty string", status)
			}
			// Status 1 (StatusUnknown) should return "Unknown", all others should not
			if status == StatusUnknown && text != "Unknown" {
				t.Errorf("GetStatusText(%d) = %q, want 'Unknown'", status, text)
			}
			if status != StatusUnknown && text == "Unknown" {
				t.Errorf("GetStatusText(%d) returned 'Unknown' for non-unknown status code", status)
			}
		})
	}
}

// TestSensorStructFields validates that Sensor struct has all required fields.
func TestSensorStructFields(t *testing.T) {
	sensor := Sensor{
		ID:                   1,
		ServerID:             1,
		Name:                 "Test Sensor",
		SensorType:           "ping",
		DeviceID:             100,
		DeviceName:           "Test Device",
		ScanningIntervalSecs: 60,
		Status:               StatusUp,
		StatusText:           "Up",
		Priority:             3,
		Message:              "OK",
		FullPath:             "/root/group/device/sensor",
		Tags:                 "production,monitoring",
	}

	// Validate fields are accessible and have expected types
	if sensor.ID != 1 {
		t.Errorf("Sensor.ID = %d, want 1", sensor.ID)
	}
	if sensor.Status != StatusUp {
		t.Errorf("Sensor.Status = %d, want %d", sensor.Status, StatusUp)
	}
	if sensor.Name != "Test Sensor" {
		t.Errorf("Sensor.Name = %q, want %q", sensor.Name, "Test Sensor")
	}
}

// TestDeviceStructFields validates Device struct integrity.
func TestDeviceStructFields(t *testing.T) {
	device := Device{
		ID:          1,
		ServerID:    1,
		Name:        "Test Device",
		Host:        "192.168.1.1",
		GroupID:     10,
		GroupName:   "Production",
		FullPath:    "/root/production/test-device",
		SensorCount: 5,
		TreeDepth:   3,
	}

	if device.ID != 1 {
		t.Errorf("Device.ID = %d, want 1", device.ID)
	}
	if device.SensorCount != 5 {
		t.Errorf("Device.SensorCount = %d, want 5", device.SensorCount)
	}
}

// TestGroupStructFields validates Group struct integrity.
func TestGroupStructFields(t *testing.T) {
	parentID := 1
	group := Group{
		ID:          10,
		ServerID:    1,
		Name:        "Production",
		IsProbeNode: false,
		ParentID:    &parentID,
		FullPath:    "/root/production",
		TreeDepth:   2,
	}

	if group.ID != 10 {
		t.Errorf("Group.ID = %d, want 10", group.ID)
	}
	if group.ParentID == nil || *group.ParentID != 1 {
		t.Errorf("Group.ParentID = %v, want 1", group.ParentID)
	}
}

// TestDeviceOverviewStructFields validates DeviceOverview struct integrity.
func TestDeviceOverviewStructFields(t *testing.T) {
	device := Device{ID: 1, Name: "Test Device"}
	sensors := []Sensor{
		{ID: 1, Status: StatusUp},
		{ID: 2, Status: StatusWarning},
		{ID: 3, Status: StatusDown},
	}

	overview := DeviceOverview{
		Device:       device,
		Sensors:      sensors,
		TotalSensors: 3,
		UpSensors:    1,
		DownSensors:  1,
		WarnSensors:  1,
	}

	if overview.TotalSensors != 3 {
		t.Errorf("DeviceOverview.TotalSensors = %d, want 3", overview.TotalSensors)
	}
	if overview.UpSensors != 1 {
		t.Errorf("DeviceOverview.UpSensors = %d, want 1", overview.UpSensors)
	}
	if overview.DownSensors != 1 {
		t.Errorf("DeviceOverview.DownSensors = %d, want 1", overview.DownSensors)
	}
	if overview.WarnSensors != 1 {
		t.Errorf("DeviceOverview.WarnSensors = %d, want 1", overview.WarnSensors)
	}
	if len(overview.Sensors) != 3 {
		t.Errorf("len(DeviceOverview.Sensors) = %d, want 3", len(overview.Sensors))
	}
}

// TestStatusConstants_NoGaps validates that there are no gaps in status code sequence.
func TestStatusConstants_NoGaps(t *testing.T) {
	statusMap := map[int]string{
		1:  "Unknown",
		2:  "Collecting",
		3:  "Up",
		4:  "Warning",
		5:  "Down",
		6:  "No Probe",
		7:  "Paused (User)",
		8:  "Paused (Dependency)",
		9:  "Paused (Schedule)",
		10: "Unusual",
		11: "Paused (License)",
		12: "Paused Until",
		13: "Down (Acknowledged)",
		14: "Down (Partial)",
	}

	// Ensure all codes 1-14 are present
	for code := 1; code <= 14; code++ {
		expectedText, exists := statusMap[code]
		if !exists {
			t.Errorf("Status code %d is missing from expected mapping", code)
			continue
		}

		actualText := GetStatusText(code)
		if actualText != expectedText {
			t.Errorf("GetStatusText(%d) = %q, want %q", code, actualText, expectedText)
		}
	}
}

// BenchmarkGetStatusText benchmarks the GetStatusText function.
func BenchmarkGetStatusText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetStatusText(StatusDown)
	}
}

// BenchmarkGetStatusText_AllCodes benchmarks GetStatusText for all valid codes.
func BenchmarkGetStatusText_AllCodes(b *testing.B) {
	codes := []int{
		StatusUnknown, StatusCollecting, StatusUp, StatusWarning,
		StatusDown, StatusNoProbe, StatusPausedByUser, StatusPausedByDependency,
		StatusPausedBySchedule, StatusUnusual, StatusPausedByLicense,
		StatusPausedUntil, StatusDownAcknowledged, StatusDownPartial,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, code := range codes {
			_ = GetStatusText(code)
		}
	}
}
