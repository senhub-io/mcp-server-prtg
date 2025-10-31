// Package handlers implements MCP (Model Context Protocol) tool handlers for PRTG monitoring data.
package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// formatDuration formats a duration in seconds to a human-readable string.
func formatDuration(seconds *float64) string {
	if seconds == nil || *seconds == 0 {
		return "-"
	}

	duration := time.Duration(*seconds) * time.Second
	hours := duration.Hours()

	if hours < 1 {
		minutes := duration.Minutes()
		return fmt.Sprintf("%.0fm", minutes)
	}

	if hours < 24 {
		return fmt.Sprintf("%.1fh", hours)
	}

	days := hours / 24
	return fmt.Sprintf("%.1fd", days)
}

// getStatusEmoji returns an emoji for a PRTG status code.
func getStatusEmoji(status int) string {
	switch status {
	case 3: // Up
		return "🟢"
	case 4: // Warning
		return "🟡"
	case 5: // Down
		return "🔴"
	case 7: // Paused
		return "⏸️"
	case 13: // Unknown
		return "❓"
	default:
		return "⚪"
	}
}

// getPriorityEmoji returns an emoji for a priority level (1-5).
func getPriorityEmoji(priority int) string {
	switch priority {
	case 5:
		return "🔴"
	case 4:
		return "🟠"
	case 3:
		return "🟡"
	case 2:
		return "🔵"
	case 1:
		return "⚪"
	default:
		return "⚪"
	}
}

// formatAlertsResponse formats alerts in a visual Markdown table format with full JSON data.
func formatAlertsResponse(alerts []types.Sensor) string {
	var sb strings.Builder

	// 1. Header with count
	sb.WriteString(fmt.Sprintf("## 🚨 Alert Summary\n\n"))
	sb.WriteString(fmt.Sprintf("Found **%d alert(s)** requiring attention\n\n", len(alerts)))

	if len(alerts) == 0 {
		sb.WriteString("✅ No alerts found. All systems operational!\n")
		return sb.String()
	}

	// 2. Status breakdown
	statusCount := make(map[int]int)
	for _, alert := range alerts {
		statusCount[alert.Status]++
	}

	sb.WriteString("**Breakdown by status:**\n")
	if count, ok := statusCount[5]; ok {
		sb.WriteString(fmt.Sprintf("- 🔴 **Critical (Down):** %d sensor(s)\n", count))
	}
	if count, ok := statusCount[4]; ok {
		sb.WriteString(fmt.Sprintf("- 🟡 **Warning:** %d sensor(s)\n", count))
	}
	if count, ok := statusCount[13]; ok {
		sb.WriteString(fmt.Sprintf("- ❓ **Unknown:** %d sensor(s)\n", count))
	}
	sb.WriteString("\n")

	// 3. Markdown table (show top 25)
	sb.WriteString("| Priority | Sensor | Device | Status | Downtime | Message |\n")
	sb.WriteString("|----------|--------|--------|--------|----------|----------|\n")

	displayCount := len(alerts)
	if displayCount > 25 {
		displayCount = 25
	}

	for i := 0; i < displayCount; i++ {
		alert := alerts[i]
		statusEmoji := getStatusEmoji(alert.Status)
		priorityEmoji := getPriorityEmoji(alert.Priority)
		downtime := formatDuration(alert.DowntimeSinceSecs)
		message := truncateString(alert.Message, 50)

		sb.WriteString(fmt.Sprintf("| %s %d | %s | %s | %s %s | %s | %s |\n",
			priorityEmoji,
			alert.Priority,
			truncateString(alert.Name, 25),
			truncateString(alert.DeviceName, 20),
			statusEmoji,
			alert.StatusText,
			downtime,
			message,
		))
	}

	if len(alerts) > 25 {
		sb.WriteString(fmt.Sprintf("| ... | *%d more alerts* | ... | ... | ... | ... |\n", len(alerts)-25))
	}

	// 4. Hint for artifact
	sb.WriteString("\n---\n\n")
	sb.WriteString("💾 **Complete dataset below** (downloadable for further analysis)\n\n")

	// 5. Full JSON data
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(alerts, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatSensorsResponse formats sensors in a visual Markdown table format with full JSON data.
func formatSensorsResponse(sensors []types.Sensor) string {
	var sb strings.Builder

	// 1. Header with count
	sb.WriteString(fmt.Sprintf("## 📊 Sensors Overview\n\n"))
	sb.WriteString(fmt.Sprintf("Found **%d sensor(s)**\n\n", len(sensors)))

	if len(sensors) == 0 {
		sb.WriteString("No sensors found matching the criteria.\n")
		return sb.String()
	}

	// 2. Status breakdown
	statusCount := make(map[int]int)
	for _, sensor := range sensors {
		statusCount[sensor.Status]++
	}

	sb.WriteString("**Breakdown by status:**\n")
	if count, ok := statusCount[3]; ok {
		sb.WriteString(fmt.Sprintf("- 🟢 **Up:** %d sensor(s)\n", count))
	}
	if count, ok := statusCount[4]; ok {
		sb.WriteString(fmt.Sprintf("- 🟡 **Warning:** %d sensor(s)\n", count))
	}
	if count, ok := statusCount[5]; ok {
		sb.WriteString(fmt.Sprintf("- 🔴 **Down:** %d sensor(s)\n", count))
	}
	if count, ok := statusCount[7]; ok {
		sb.WriteString(fmt.Sprintf("- ⏸️ **Paused:** %d sensor(s)\n", count))
	}
	sb.WriteString("\n")

	// 3. Markdown table (show top 20)
	sb.WriteString("| ID | Name | Status | Device | Type | Uptime |\n")
	sb.WriteString("|----|------|--------|--------|------|--------|\n")

	displayCount := len(sensors)
	if displayCount > 20 {
		displayCount = 20
	}

	for i := 0; i < displayCount; i++ {
		sensor := sensors[i]
		statusEmoji := getStatusEmoji(sensor.Status)
		uptime := formatDuration(sensor.UptimeSinceSecs)

		sb.WriteString(fmt.Sprintf("| %d | %s | %s %s | %s | %s | %s |\n",
			sensor.ID,
			truncateString(sensor.Name, 25),
			statusEmoji,
			sensor.StatusText,
			truncateString(sensor.DeviceName, 20),
			truncateString(sensor.SensorType, 15),
			uptime,
		))
	}

	if len(sensors) > 20 {
		sb.WriteString(fmt.Sprintf("| ... | *%d more sensors* | ... | ... | ... | ... |\n", len(sensors)-20))
	}

	// 4. Hint for artifact
	sb.WriteString("\n---\n\n")
	sb.WriteString("💾 **Complete dataset below** (downloadable for further analysis)\n\n")

	// 5. Full JSON data
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(sensors, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatDeviceOverviewResponse formats device overview in a visual format.
func formatDeviceOverviewResponse(overview *types.DeviceOverview) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 🖥️ Device Overview: %s\n\n", overview.Device.Name))

	// 2. Device info
	sb.WriteString("**Device Information:**\n")
	sb.WriteString(fmt.Sprintf("- **Device ID:** %d\n", overview.Device.ID))
	sb.WriteString(fmt.Sprintf("- **Host:** %s\n", overview.Device.Host))
	sb.WriteString(fmt.Sprintf("- **Total Sensors:** %d\n", overview.TotalSensors))
	if overview.Device.FullPath != "" {
		sb.WriteString(fmt.Sprintf("- **Path:** %s\n", overview.Device.FullPath))
	}
	sb.WriteString("\n")

	// 3. Status summary
	sb.WriteString("**Status Summary:**\n")
	sb.WriteString(fmt.Sprintf("- 🟢 **Up:** %d sensor(s)\n", overview.UpSensors))
	sb.WriteString(fmt.Sprintf("- 🟡 **Warning:** %d sensor(s)\n", overview.WarnSensors))
	sb.WriteString(fmt.Sprintf("- 🔴 **Down:** %d sensor(s)\n", overview.DownSensors))

	// Calculate other statuses
	otherSensors := overview.TotalSensors - overview.UpSensors - overview.WarnSensors - overview.DownSensors
	if otherSensors > 0 {
		sb.WriteString(fmt.Sprintf("- ⚪ **Other:** %d sensor(s)\n", otherSensors))
	}
	sb.WriteString("\n")

	// 4. Sensors table
	if len(overview.Sensors) > 0 {
		sb.WriteString("**Sensors:**\n\n")
		sb.WriteString("| Name | Status | Type | Last Check |\n")
		sb.WriteString("|------|--------|------|------------|\n")

		for _, sensor := range overview.Sensors {
			statusEmoji := getStatusEmoji(sensor.Status)
			lastCheck := "-"
			if sensor.LastCheckUTC != nil {
				lastCheck = sensor.LastCheckUTC.Format("2006-01-02 15:04")
			}

			sb.WriteString(fmt.Sprintf("| %s | %s %s | %s | %s |\n",
				truncateString(sensor.Name, 30),
				statusEmoji,
				sensor.StatusText,
				truncateString(sensor.SensorType, 20),
				lastCheck,
			))
		}
	}

	// 5. Full JSON data
	sb.WriteString("\n---\n\n")
	sb.WriteString("💾 **Complete data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(overview, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatTopSensorsResponse formats top sensors in a visual format.
func formatTopSensorsResponse(sensors []types.Sensor, metric string) string {
	var sb strings.Builder

	// 1. Header
	metricLabel := "sensors"
	switch metric {
	case "downtime":
		metricLabel = "sensors by downtime"
	case "priority":
		metricLabel = "sensors by priority"
	}

	sb.WriteString(fmt.Sprintf("## 📈 Top %s\n\n", metricLabel))
	sb.WriteString(fmt.Sprintf("Found **%d sensor(s)**\n\n", len(sensors)))

	if len(sensors) == 0 {
		sb.WriteString("No sensors found.\n")
		return sb.String()
	}

	// 2. Table
	sb.WriteString("| Rank | Sensor | Device | Status | Metric | Message |\n")
	sb.WriteString("|------|--------|--------|--------|--------|----------|\n")

	for i, sensor := range sensors {
		statusEmoji := getStatusEmoji(sensor.Status)
		metricValue := "-"

		switch metric {
		case "downtime":
			metricValue = formatDuration(sensor.DowntimeSinceSecs)
		case "priority":
			metricValue = fmt.Sprintf("%s %d", getPriorityEmoji(sensor.Priority), sensor.Priority)
		}

		sb.WriteString(fmt.Sprintf("| #%d | %s | %s | %s %s | %s | %s |\n",
			i+1,
			truncateString(sensor.Name, 25),
			truncateString(sensor.DeviceName, 20),
			statusEmoji,
			sensor.StatusText,
			metricValue,
			truncateString(sensor.Message, 30),
		))
	}

	// 3. Full JSON data
	sb.WriteString("\n---\n\n")
	sb.WriteString("💾 **Complete dataset below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(sensors, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
