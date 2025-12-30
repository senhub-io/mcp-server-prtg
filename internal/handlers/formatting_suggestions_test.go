package handlers

import (
	"strings"
	"testing"

	"github.com/matthieu/mcp-server-prtg/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatAlertsResponse_WithSuggestions(t *testing.T) {
	t.Run("No alerts - should suggest checking statistics", func(t *testing.T) {
		alerts := []types.Sensor{}
		result := formatAlertsResponse(alerts)

		assert.Contains(t, result, "No alerts found")
		assert.Contains(t, result, "All systems operational")
	})

	t.Run("Alerts with Down sensors - should suggest investigating critical", func(t *testing.T) {
		alerts := []types.Sensor{
			{
				ID:       123,
				Name:     "Test Sensor",
				Status:   5, // Down
				Priority: 5,
			},
		}
		result := formatAlertsResponse(alerts)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_sensor_status sensor_id=123")
		assert.Contains(t, result, "prtg_get_sensor_timeseries sensor_id=123")
		assert.Contains(t, result, "prtg_get_channel_current_values sensor_id=123")
	})

	t.Run("Only warning alerts - should suggest checking trends", func(t *testing.T) {
		alerts := []types.Sensor{
			{
				ID:       456,
				Name:     "Warning Sensor",
				Status:   4, // Warning
				Priority: 3,
			},
		}
		result := formatAlertsResponse(alerts)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_top_sensors")
		assert.Contains(t, result, "prtg_get_statistics")
	})
}

func TestFormatSensorsResponse_WithSuggestions(t *testing.T) {
	t.Run("Empty sensors list", func(t *testing.T) {
		sensors := []types.Sensor{}
		result := formatSensorsResponse(sensors)

		assert.Contains(t, result, "No sensors found")
	})

	t.Run("Sensors with Down status - should suggest viewing alerts", func(t *testing.T) {
		sensors := []types.Sensor{
			{
				ID:     789,
				Name:   "Down Sensor",
				Status: 5, // Down
			},
		}
		result := formatSensorsResponse(sensors)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_alerts status=5")
		assert.Contains(t, result, "prtg_get_sensor_status sensor_id=789")
	})

	t.Run("Sensors with Warning status - should suggest reviewing warnings", func(t *testing.T) {
		sensors := []types.Sensor{
			{
				ID:     101,
				Name:   "Warning Sensor",
				Status: 4, // Warning
			},
		}
		result := formatSensorsResponse(sensors)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_alerts status=4")
	})

	t.Run("Many sensors - should suggest narrowing results", func(t *testing.T) {
		sensors := make([]types.Sensor, 25)
		for i := 0; i < 25; i++ {
			sensors[i] = types.Sensor{
				ID:     i + 1,
				Name:   "Sensor",
				Status: 3, // Up
			}
		}
		result := formatSensorsResponse(sensors)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "Narrow results with filters")
	})
}

func TestFormatDeviceOverviewResponse_WithSuggestions(t *testing.T) {
	t.Run("Device with Down sensors - should suggest viewing alerts", func(t *testing.T) {
		overview := &types.DeviceOverview{
			Device: types.Device{
				ID:        1,
				Name:      "TestDevice",
				Host:      "test.local",
				GroupName: "TestGroup",
			},
			TotalSensors: 10,
			UpSensors:    5,
			WarnSensors:  2,
			DownSensors:  3,
			Sensors: []types.Sensor{
				{
					ID:     100,
					Name:   "Sensor1",
					Status: 5, // Down
				},
			},
		}
		result := formatDeviceOverviewResponse(overview)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_alerts device_name=\"TestDevice\" status=5")
		assert.Contains(t, result, "prtg_get_sensor_status sensor_id=100")
		assert.Contains(t, result, "prtg_get_sensors group_name=\"TestGroup\"")
	})

	t.Run("Device with only Warning sensors", func(t *testing.T) {
		overview := &types.DeviceOverview{
			Device: types.Device{
				ID:   2,
				Name: "WarnDevice",
			},
			TotalSensors: 5,
			UpSensors:    3,
			WarnSensors:  2,
			DownSensors:  0,
			Sensors: []types.Sensor{
				{
					ID:     200,
					Name:   "WarnSensor",
					Status: 4, // Warning
				},
			},
		}
		result := formatDeviceOverviewResponse(overview)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_alerts device_name=\"WarnDevice\" status=4")
	})
}

func TestFormatSearchResponse_WithSuggestions(t *testing.T) {
	t.Run("Search with multiple result types", func(t *testing.T) {
		results := &types.SearchResults{
			Groups: []types.Group{
				{ID: 1, Name: "TestGroup"},
			},
			Devices: []types.Device{
				{ID: 2, Name: "TestDevice"},
			},
			Sensors: []types.Sensor{
				{ID: 3, Name: "TestSensor"},
			},
		}
		searchTerm := "test"
		result := formatSearchResponse(results, searchTerm)

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "prtg_get_sensor_status sensor_id=3")
		assert.Contains(t, result, "prtg_device_overview device_name=\"TestDevice\"")
		assert.Contains(t, result, "prtg_get_hierarchy group_name=\"TestGroup\"")
	})

	t.Run("Search with many results - should suggest refining", func(t *testing.T) {
		sensors := make([]types.Sensor, 65)
		for i := 0; i < 65; i++ {
			sensors[i] = types.Sensor{
				ID:   i + 1,
				Name: "Sensor",
			}
		}

		results := &types.SearchResults{
			Sensors: sensors,
		}
		result := formatSearchResponse(results, "test")

		assert.Contains(t, result, "Next suggested actions")
		assert.Contains(t, result, "Refine search with more specific terms")
	})
}

func TestSuggestionsFormat(t *testing.T) {
	t.Run("All suggestions use backticks for commands", func(t *testing.T) {
		alerts := []types.Sensor{
			{
				ID:       999,
				Name:     "Test",
				Status:   5,
				Priority: 5,
			},
		}
		result := formatAlertsResponse(alerts)

		// Count backticks - should have pairs for each command
		backtickCount := strings.Count(result, "`")
		assert.Greater(t, backtickCount, 0, "Should contain command suggestions with backticks")
		assert.Equal(t, 0, backtickCount%2, "Backticks should come in pairs")
	})

	t.Run("Suggestions section has proper formatting", func(t *testing.T) {
		sensors := []types.Sensor{
			{
				ID:     111,
				Name:   "Test",
				Status: 5,
			},
		}
		result := formatSensorsResponse(sensors)

		assert.Contains(t, result, "**Next suggested actions:**")
		// Check that suggestions are formatted as list items
		lines := strings.Split(result, "\n")
		foundSuggestionSection := false
		hasSuggestions := false
		for _, line := range lines {
			if strings.Contains(line, "Next suggested actions") {
				foundSuggestionSection = true
			}
			if foundSuggestionSection && strings.HasPrefix(line, "- ") {
				hasSuggestions = true
				break
			}
		}
		assert.True(t, foundSuggestionSection, "Should have suggestions section")
		assert.True(t, hasSuggestions, "Should have list items for suggestions")
	})
}
