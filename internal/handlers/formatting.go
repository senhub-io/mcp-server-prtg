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

// formatHierarchyResponse formats hierarchy in a visual tree format with full JSON data.
func formatHierarchyResponse(node *types.HierarchyNode) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 🌳 PRTG Hierarchy: %s\n\n", node.Group.Name))

	// 2. Group info
	sb.WriteString("**Group Information:**\n")
	sb.WriteString(fmt.Sprintf("- **Group ID:** %d\n", node.Group.ID))
	sb.WriteString(fmt.Sprintf("- **Tree Depth:** %d\n", node.Group.TreeDepth))
	if node.Group.IsProbeNode {
		sb.WriteString("- **Type:** 📡 Probe Node\n")
	} else {
		sb.WriteString("- **Type:** 📁 Group\n")
	}
	if node.Group.FullPath != "" {
		sb.WriteString(fmt.Sprintf("- **Path:** %s\n", node.Group.FullPath))
	}
	sb.WriteString("\n")

	// 3. Tree structure
	sb.WriteString("**Tree Structure:**\n\n")
	formatHierarchyNode(&sb, node, "", true)
	sb.WriteString("\n")

	// 4. Statistics summary
	deviceCount, sensorCount := countHierarchyStats(node)
	childGroupCount := len(node.Groups)

	sb.WriteString("**Summary:**\n")
	sb.WriteString(fmt.Sprintf("- **Child Groups:** %d\n", childGroupCount))
	sb.WriteString(fmt.Sprintf("- **Total Devices:** %d\n", deviceCount))
	sb.WriteString(fmt.Sprintf("- **Total Sensors:** %d\n", sensorCount))
	sb.WriteString("\n")

	// 5. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete hierarchy data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(node, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatHierarchyNode recursively formats a hierarchy node as a tree structure.
func formatHierarchyNode(sb *strings.Builder, node *types.HierarchyNode, prefix string, isLast bool) {
	// Determine the branch characters
	branch := "├── "
	if isLast {
		branch = "└── "
	}

	// Group name
	groupType := "📁"
	if node.Group.IsProbeNode {
		groupType = "📡"
	}
	sb.WriteString(fmt.Sprintf("%s%s %s %s\n", prefix, branch, groupType, node.Group.Name))

	// Prepare prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Devices in this group
	for i, device := range node.Devices {
		isLastDevice := i == len(node.Devices)-1 && len(node.Groups) == 0

		deviceBranch := "├── "
		if isLastDevice {
			deviceBranch = "└── "
		}

		statusInfo := ""
		if device.Device.SensorCount > 0 {
			statusInfo = fmt.Sprintf(" (%d sensors)", device.Device.SensorCount)
		}

		sb.WriteString(fmt.Sprintf("%s%s 🖥️  %s%s\n", childPrefix, deviceBranch, device.Device.Name, statusInfo))

		// Sensors if included
		if len(device.Sensors) > 0 {
			sensorPrefix := childPrefix
			if isLastDevice {
				sensorPrefix += "    "
			} else {
				sensorPrefix += "│   "
			}

			for j, sensor := range device.Sensors {
				isLastSensor := j == len(device.Sensors)-1
				sensorBranch := "├── "
				if isLastSensor {
					sensorBranch = "└── "
				}

				emoji := getStatusEmoji(sensor.Status)
				sb.WriteString(fmt.Sprintf("%s%s %s %s (%s)\n",
					sensorPrefix, sensorBranch, emoji, sensor.Name, sensor.StatusText))
			}
		}
	}

	// Child groups
	for i, childGroup := range node.Groups {
		isLastGroup := i == len(node.Groups)-1
		formatHierarchyNode(sb, childGroup, childPrefix, isLastGroup)
	}
}

// countHierarchyStats counts total devices and sensors in the hierarchy tree.
func countHierarchyStats(node *types.HierarchyNode) (devices, sensors int) {
	devices = len(node.Devices)

	for _, device := range node.Devices {
		sensors += len(device.Sensors)
	}

	for _, childGroup := range node.Groups {
		childDevices, childSensors := countHierarchyStats(childGroup)
		devices += childDevices
		sensors += childSensors
	}

	return devices, sensors
}

// formatSearchResponse formats universal search results in a visual format with full JSON data.
func formatSearchResponse(results *types.SearchResults, searchTerm string) string {
	var sb strings.Builder

	totalResults := len(results.Groups) + len(results.Devices) + len(results.Sensors)

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 🔍 Search Results for \"%s\"\n\n", searchTerm))
	sb.WriteString(fmt.Sprintf("Found **%d total result(s)** across all categories\n\n", totalResults))

	if totalResults == 0 {
		sb.WriteString("No results found. Try a different search term.\n")
		return sb.String()
	}

	// 2. Summary breakdown
	sb.WriteString("**Results by category:**\n")
	sb.WriteString(fmt.Sprintf("- 📁 **Groups:** %d\n", len(results.Groups)))
	sb.WriteString(fmt.Sprintf("- 🖥️  **Devices:** %d\n", len(results.Devices)))
	sb.WriteString(fmt.Sprintf("- 📊 **Sensors:** %d\n", len(results.Sensors)))
	sb.WriteString("\n")

	// 3. Groups section
	if len(results.Groups) > 0 {
		sb.WriteString("### 📁 Groups\n\n")
		sb.WriteString("| ID | Name | Type | Path |\n")
		sb.WriteString("|----|------|------|------|\n")

		displayCount := len(results.Groups)
		if displayCount > 20 {
			displayCount = 20
		}

		for i := 0; i < displayCount; i++ {
			group := results.Groups[i]
			groupType := "Group"
			if group.IsProbeNode {
				groupType = "Probe"
			}

			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
				group.ID,
				truncateString(group.Name, 30),
				groupType,
				truncateString(group.FullPath, 40),
			))
		}

		if len(results.Groups) > 20 {
			sb.WriteString(fmt.Sprintf("| ... | *%d more groups* | ... | ... |\n", len(results.Groups)-20))
		}
		sb.WriteString("\n")
	}

	// 4. Devices section
	if len(results.Devices) > 0 {
		sb.WriteString("### 🖥️  Devices\n\n")
		sb.WriteString("| ID | Name | Host | Group | Sensors |\n")
		sb.WriteString("|----|------|------|-------|----------|\n")

		displayCount := len(results.Devices)
		if displayCount > 20 {
			displayCount = 20
		}

		for i := 0; i < displayCount; i++ {
			device := results.Devices[i]

			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %d |\n",
				device.ID,
				truncateString(device.Name, 25),
				truncateString(device.Host, 20),
				truncateString(device.GroupName, 20),
				device.SensorCount,
			))
		}

		if len(results.Devices) > 20 {
			sb.WriteString(fmt.Sprintf("| ... | *%d more devices* | ... | ... | ... |\n", len(results.Devices)-20))
		}
		sb.WriteString("\n")
	}

	// 5. Sensors section
	if len(results.Sensors) > 0 {
		sb.WriteString("### 📊 Sensors\n\n")
		sb.WriteString("| ID | Name | Device | Type | Status |\n")
		sb.WriteString("|----|------|--------|------|--------|\n")

		displayCount := len(results.Sensors)
		if displayCount > 20 {
			displayCount = 20
		}

		for i := 0; i < displayCount; i++ {
			sensor := results.Sensors[i]
			statusEmoji := getStatusEmoji(sensor.Status)

			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s %s |\n",
				sensor.ID,
				truncateString(sensor.Name, 25),
				truncateString(sensor.DeviceName, 20),
				truncateString(sensor.SensorType, 15),
				statusEmoji,
				sensor.StatusText,
			))
		}

		if len(results.Sensors) > 20 {
			sb.WriteString(fmt.Sprintf("| ... | *%d more sensors* | ... | ... | ... |\n", len(results.Sensors)-20))
		}
		sb.WriteString("\n")
	}

	// 6. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete search results below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(results, "", "  ")
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
