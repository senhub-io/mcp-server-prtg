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

	// 4. Contextual suggestions
	sb.WriteString("\n---\n\n")
	sb.WriteString("**Next suggested actions:**\n")

	// Find the most critical sensor (highest priority Down sensor)
	var mostCriticalID int
	highestPriority := 0
	hasDownSensors := false

	for _, alert := range alerts {
		if alert.Status == 5 { // Down
			hasDownSensors = true
			if alert.Priority > highestPriority {
				highestPriority = alert.Priority
				mostCriticalID = alert.ID
			}
		}
	}

	if hasDownSensors && mostCriticalID > 0 {
		sb.WriteString(fmt.Sprintf("- Investigate critical sensor: `prtg_get_sensor_status sensor_id=%d`\n", mostCriticalID))
		sb.WriteString(fmt.Sprintf("- View historical data: `prtg_get_sensor_timeseries sensor_id=%d time_type=short`\n", mostCriticalID))
		sb.WriteString(fmt.Sprintf("- Check current values: `prtg_get_channel_current_values sensor_id=%d`\n", mostCriticalID))
	} else if len(alerts) > 0 {
		// If only warnings, suggest checking trends
		sb.WriteString("- Check sensor trends: `prtg_top_sensors metric=downtime hours=24`\n")
		sb.WriteString("- Review overall statistics: `prtg_get_statistics`\n")
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("**Complete dataset below** (downloadable for further analysis)\n\n")

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

	// 4. Contextual suggestions
	sb.WriteString("\n---\n\n")
	sb.WriteString("**Next suggested actions:**\n")

	// Analyze the sensor list to provide relevant suggestions
	downCount := statusCount[5]   // Down
	warnCount := statusCount[4]   // Warning
	pausedCount := statusCount[7] // Paused

	if downCount > 0 {
		sb.WriteString("- View critical alerts: `prtg_get_alerts status=5`\n")
		sb.WriteString("- Check top problematic sensors: `prtg_top_sensors metric=downtime hours=24`\n")
	} else if warnCount > 0 {
		sb.WriteString("- Review warning sensors: `prtg_get_alerts status=4`\n")
		sb.WriteString("- Analyze sensor trends: `prtg_top_sensors metric=alerts hours=24`\n")
	} else if pausedCount > 0 {
		sb.WriteString("- Review paused sensors and consider resuming monitoring\n")
		sb.WriteString("- Check overall health: `prtg_get_statistics`\n")
	}

	// If we have sensor data, suggest inspecting a specific sensor
	if len(sensors) > 0 {
		firstSensorID := sensors[0].ID
		sb.WriteString(fmt.Sprintf("- Inspect specific sensor: `prtg_get_sensor_status sensor_id=%d`\n", firstSensorID))
	}

	// General suggestions
	if len(sensors) > 20 {
		sb.WriteString("- Narrow results with filters: device_name, sensor_type, or tags\n")
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("**Complete dataset below** (downloadable for further analysis)\n\n")

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
	if overview.Device.GroupName != "" {
		sb.WriteString(fmt.Sprintf("- **Group:** %s\n", overview.Device.GroupName))
	}
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

	// 4. Tag summary
	if len(overview.Sensors) > 0 {
		tagMap := make(map[string]int)
		for _, sensor := range overview.Sensors {
			if sensor.Tags != "" {
				tags := strings.Split(sensor.Tags, ",")
				for _, tag := range tags {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						tagMap[tag]++
					}
				}
			}
		}

		if len(tagMap) > 0 {
			sb.WriteString("**Tags used on this device:**\n")
			// Sort tags by usage count
			type tagCount struct {
				tag   string
				count int
			}
			tagCounts := make([]tagCount, 0, len(tagMap))
			for tag, count := range tagMap {
				tagCounts = append(tagCounts, tagCount{tag, count})
			}
			// Sort by count descending
			for i := 0; i < len(tagCounts); i++ {
				for j := i + 1; j < len(tagCounts); j++ {
					if tagCounts[j].count > tagCounts[i].count {
						tagCounts[i], tagCounts[j] = tagCounts[j], tagCounts[i]
					}
				}
			}
			// Show top 10 tags
			displayCount := len(tagCounts)
			if displayCount > 10 {
				displayCount = 10
			}
			for i := 0; i < displayCount; i++ {
				sb.WriteString(fmt.Sprintf("- 🏷️ **%s** (%d sensor%s)\n",
					tagCounts[i].tag,
					tagCounts[i].count,
					func() string {
						if tagCounts[i].count > 1 {
							return "s"
						}
						return ""
					}(),
				))
			}
			if len(tagCounts) > 10 {
				sb.WriteString(fmt.Sprintf("- *...and %d more tags*\n", len(tagCounts)-10))
			}
			sb.WriteString("\n")
		}
	}

	// 5. Sensors table
	if len(overview.Sensors) > 0 {
		sb.WriteString("**Sensors:**\n\n")
		sb.WriteString("| Name | Status | Type | Last Check | Tags |\n")
		sb.WriteString("|------|--------|------|------------|------|\n")

		displayCount := len(overview.Sensors)
		if displayCount > 50 {
			displayCount = 50
		}

		for i := 0; i < displayCount; i++ {
			sensor := overview.Sensors[i]
			statusEmoji := getStatusEmoji(sensor.Status)
			lastCheck := "-"
			if sensor.LastCheckUTC != nil {
				lastCheck = sensor.LastCheckUTC.Format("2006-01-02 15:04")
			}

			tags := "-"
			if sensor.Tags != "" {
				tags = truncateString(strings.ReplaceAll(sensor.Tags, ",", ", "), 30)
			}

			sb.WriteString(fmt.Sprintf("| %s | %s %s | %s | %s | %s |\n",
				truncateString(sensor.Name, 30),
				statusEmoji,
				sensor.StatusText,
				truncateString(sensor.SensorType, 15),
				lastCheck,
				tags,
			))
		}

		if len(overview.Sensors) > 50 {
			sb.WriteString(fmt.Sprintf("| ... | *%d more sensors* | ... | ... | ... |\n", len(overview.Sensors)-50))
		}
	}

	// 6. Contextual suggestions
	sb.WriteString("\n---\n\n")
	sb.WriteString("**Next suggested actions:**\n")

	if overview.DownSensors > 0 {
		sb.WriteString(fmt.Sprintf("- View down sensors on this device: `prtg_get_alerts device_name=\"%s\" status=5`\n", overview.Device.Name))
		sb.WriteString(fmt.Sprintf("- Check historical issues: `prtg_get_sensors device_name=\"%s\" order_by=status`\n", overview.Device.Name))
	} else if overview.WarnSensors > 0 {
		sb.WriteString(fmt.Sprintf("- Review warning sensors: `prtg_get_alerts device_name=\"%s\" status=4`\n", overview.Device.Name))
	}

	// Suggest inspecting specific sensors if available
	if len(overview.Sensors) > 0 {
		firstSensorID := overview.Sensors[0].ID
		sb.WriteString(fmt.Sprintf("- Inspect a sensor in detail: `prtg_get_sensor_status sensor_id=%d`\n", firstSensorID))
		sb.WriteString(fmt.Sprintf("- View sensor trends: `prtg_get_sensor_timeseries sensor_id=%d time_type=short`\n", firstSensorID))
	}

	// Suggest exploring related devices
	if overview.Device.GroupName != "" {
		sb.WriteString(fmt.Sprintf("- Explore other devices in group: `prtg_get_sensors group_name=\"%s\"`\n", overview.Device.GroupName))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("**Complete data below** (downloadable)\n\n")
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

	// 6. Contextual suggestions
	sb.WriteString("---\n\n")
	sb.WriteString("**Next suggested actions:**\n")

	// Suggest actions based on what was found
	if len(results.Sensors) > 0 {
		firstSensorID := results.Sensors[0].ID
		sb.WriteString(fmt.Sprintf("- Inspect first sensor: `prtg_get_sensor_status sensor_id=%d`\n", firstSensorID))
		sb.WriteString(fmt.Sprintf("- View sensor history: `prtg_get_sensor_timeseries sensor_id=%d time_type=short`\n", firstSensorID))
	}

	if len(results.Devices) > 0 {
		firstDeviceName := results.Devices[0].Name
		sb.WriteString(fmt.Sprintf("- Explore device sensors: `prtg_device_overview device_name=\"%s\"`\n", firstDeviceName))
	}

	if len(results.Groups) > 0 {
		firstGroupName := results.Groups[0].Name
		sb.WriteString(fmt.Sprintf("- View group hierarchy: `prtg_get_hierarchy group_name=\"%s\"`\n", firstGroupName))
	}

	// Suggest refining the search if too many results
	if totalResults > 60 {
		sb.WriteString("- Refine search with more specific terms for better results\n")
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("**Complete search results below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(results, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatGroupsResponse formats groups in a visual format with full JSON data.
func formatGroupsResponse(groups []types.Group) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 📁 PRTG Groups\n\n"))
	sb.WriteString(fmt.Sprintf("Found **%d group(s)**\n\n", len(groups)))

	if len(groups) == 0 {
		sb.WriteString("No groups found matching the criteria.\n")
		return sb.String()
	}

	// 2. Breakdown by type
	probeCount := 0
	groupCount := 0
	for _, group := range groups {
		if group.IsProbeNode {
			probeCount++
		} else {
			groupCount++
		}
	}

	sb.WriteString("**Breakdown by type:**\n")
	sb.WriteString(fmt.Sprintf("- 📡 **Probes:** %d\n", probeCount))
	sb.WriteString(fmt.Sprintf("- 📁 **Groups:** %d\n", groupCount))
	sb.WriteString("\n")

	// 3. Groups table
	sb.WriteString("| ID | Name | Type | Tree Depth | Path |\n")
	sb.WriteString("|----|------|------|------------|------|\n")

	displayCount := len(groups)
	if displayCount > 50 {
		displayCount = 50
	}

	for i := 0; i < displayCount; i++ {
		group := groups[i]
		groupType := "Group"
		typeIcon := "📁"
		if group.IsProbeNode {
			groupType = "Probe"
			typeIcon = "📡"
		}

		sb.WriteString(fmt.Sprintf("| %d | %s | %s %s | %d | %s |\n",
			group.ID,
			truncateString(group.Name, 30),
			typeIcon,
			groupType,
			group.TreeDepth,
			truncateString(group.FullPath, 50),
		))
	}

	if len(groups) > 50 {
		sb.WriteString(fmt.Sprintf("| ... | *%d more groups* | ... | ... | ... |\n", len(groups)-50))
	}
	sb.WriteString("\n")

	// 4. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete groups data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(groups, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatTagsResponse formats tags data with visual summary and JSON export.
func formatTagsResponse(tags []types.Tag) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 🏷️ PRTG Tags\n\n"))
	sb.WriteString(fmt.Sprintf("Found **%d tag(s)**\n\n", len(tags)))

	if len(tags) == 0 {
		sb.WriteString("No tags found matching the criteria.\n")
		return sb.String()
	}

	// 2. Statistics
	totalSensorCount := 0
	for _, tag := range tags {
		totalSensorCount += tag.SensorCount
	}

	sb.WriteString("**Tag usage statistics:**\n")
	sb.WriteString(fmt.Sprintf("- 📊 **Total sensor associations:** %d\n", totalSensorCount))
	if len(tags) > 0 {
		avgUsage := float64(totalSensorCount) / float64(len(tags))
		sb.WriteString(fmt.Sprintf("- 📈 **Average sensors per tag:** %.1f\n", avgUsage))
	}
	sb.WriteString("\n")

	// 3. Tags table
	sb.WriteString("| ID | Tag Name | Sensor Count |\n")
	sb.WriteString("|----|----------|-------------|\n")

	displayCount := len(tags)
	if displayCount > 50 {
		displayCount = 50
	}

	for i := 0; i < displayCount; i++ {
		tag := tags[i]
		sb.WriteString(fmt.Sprintf("| %d | %s | %d |\n",
			tag.ID,
			truncateString(tag.Name, 40),
			tag.SensorCount,
		))
	}

	if len(tags) > 50 {
		sb.WriteString(fmt.Sprintf("| ... | *%d more tags* | ... |\n", len(tags)-50))
	}
	sb.WriteString("\n")

	// 4. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete tags data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(tags, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatBusinessProcessesResponse formats business process sensors with visual summary and JSON export.
func formatBusinessProcessesResponse(processes []types.Sensor) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("## 📊 PRTG Business Processes\n\n"))
	sb.WriteString(fmt.Sprintf("Found **%d business process(es)**\n\n", len(processes)))

	if len(processes) == 0 {
		sb.WriteString("No business process sensors found matching the criteria.\n")
		sb.WriteString("\n💡 **Note:** Business Process sensors aggregate the status of multiple source sensors to monitor complete workflows.\n")
		return sb.String()
	}

	// 2. Status breakdown
	statusCounts := make(map[int]int)
	for _, process := range processes {
		statusCounts[process.Status]++
	}

	sb.WriteString("**Status breakdown:**\n")
	if count, ok := statusCounts[types.StatusUp]; ok && count > 0 {
		sb.WriteString(fmt.Sprintf("- ✅ **Up:** %d\n", count))
	}
	if count, ok := statusCounts[types.StatusWarning]; ok && count > 0 {
		sb.WriteString(fmt.Sprintf("- ⚠️ **Warning:** %d\n", count))
	}
	if count, ok := statusCounts[types.StatusDown]; ok && count > 0 {
		sb.WriteString(fmt.Sprintf("- ❌ **Down:** %d\n", count))
	}
	// Show other statuses if present
	for status, count := range statusCounts {
		if status != types.StatusUp && status != types.StatusWarning && status != types.StatusDown && count > 0 {
			sb.WriteString(fmt.Sprintf("- 🔵 **%s:** %d\n", types.GetStatusText(status), count))
		}
	}
	sb.WriteString("\n")

	// 3. Business processes table
	sb.WriteString("| ID | Name | Status | Priority | Device | Last Check | Message |\n")
	sb.WriteString("|----|------|--------|----------|--------|------------|----------|\n")

	displayCount := len(processes)
	if displayCount > 50 {
		displayCount = 50
	}

	for i := 0; i < displayCount; i++ {
		process := processes[i]
		statusEmoji := getStatusEmoji(process.Status)

		lastCheck := "Never"
		if process.LastCheckUTC != nil {
			lastCheck = process.LastCheckUTC.Format("2006-01-02 15:04")
		}

		sb.WriteString(fmt.Sprintf("| %d | %s | %s %s | %d | %s | %s | %s |\n",
			process.ID,
			truncateString(process.Name, 30),
			statusEmoji,
			process.StatusText,
			process.Priority,
			truncateString(process.DeviceName, 20),
			lastCheck,
			truncateString(process.Message, 30),
		))
	}

	if len(processes) > 50 {
		sb.WriteString(fmt.Sprintf("| ... | *%d more processes* | ... | ... | ... | ... | ... |\n", len(processes)-50))
	}
	sb.WriteString("\n")

	// 4. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete business processes data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(processes, "", "  ")
	sb.WriteString(string(jsonData))
	sb.WriteString("\n```\n")

	return sb.String()
}

// formatStatisticsResponse formats PRTG server statistics with visual summary and JSON export.
func formatStatisticsResponse(stats *types.Statistics) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString("## 📊 PRTG Server Statistics\n\n")

	// 2. Overall counts
	sb.WriteString("**Overall Infrastructure:**\n")
	sb.WriteString(fmt.Sprintf("- 🖥️ **Devices:** %d\n", stats.TotalDevices))
	sb.WriteString(fmt.Sprintf("- 📁 **Groups:** %d\n", stats.TotalGroups))
	sb.WriteString(fmt.Sprintf("- 📡 **Probes:** %d\n", stats.TotalProbes))
	sb.WriteString(fmt.Sprintf("- 🔍 **Sensors:** %d\n", stats.TotalSensors))
	sb.WriteString(fmt.Sprintf("- 🏷️ **Tags:** %d\n", stats.TotalTags))
	sb.WriteString(fmt.Sprintf("- 📈 **Avg Sensors/Device:** %.1f\n", stats.AvgSensorsPerDevice))
	sb.WriteString("\n")

	// 3. Status breakdown
	if len(stats.SensorsByStatus) > 0 {
		sb.WriteString("**Sensor Status Breakdown:**\n")

		// Order: Up, Warning, Down, then others
		statusOrder := []string{"Up", "Warning", "Down"}
		for _, status := range statusOrder {
			if count, ok := stats.SensorsByStatus[status]; ok {
				emoji := ""
				switch status {
				case "Up":
					emoji = "✅"
				case "Warning":
					emoji = "⚠️"
				case "Down":
					emoji = "❌"
				}
				percentage := 0.0
				if stats.TotalSensors > 0 {
					percentage = float64(count) / float64(stats.TotalSensors) * 100
				}
				sb.WriteString(fmt.Sprintf("- %s **%s:** %d (%.1f%%)\n", emoji, status, count, percentage))
			}
		}

		// Show other statuses
		for status, count := range stats.SensorsByStatus {
			if status != "Up" && status != "Warning" && status != "Down" {
				percentage := 0.0
				if stats.TotalSensors > 0 {
					percentage = float64(count) / float64(stats.TotalSensors) * 100
				}
				sb.WriteString(fmt.Sprintf("- 🔵 **%s:** %d (%.1f%%)\n", status, count, percentage))
			}
		}
		sb.WriteString("\n")
	}

	// 4. Top sensor types
	if len(stats.TopSensorTypes) > 0 {
		sb.WriteString("**Top Sensor Types:**\n\n")
		sb.WriteString("| Rank | Sensor Type | Count | % of Total |\n")
		sb.WriteString("|------|-------------|-------|------------|\n")

		displayCount := len(stats.TopSensorTypes)
		if displayCount > 15 {
			displayCount = 15
		}

		for i := 0; i < displayCount; i++ {
			st := stats.TopSensorTypes[i]
			percentage := 0.0
			if stats.TotalSensors > 0 {
				percentage = float64(st.Count) / float64(stats.TotalSensors) * 100
			}
			sb.WriteString(fmt.Sprintf("| #%d | %s | %d | %.1f%% |\n",
				i+1,
				truncateString(st.Type, 40),
				st.Count,
				percentage,
			))
		}
		sb.WriteString("\n")
	}

	// 5. Full JSON data
	sb.WriteString("---\n\n")
	sb.WriteString("💾 **Complete statistics data below** (downloadable)\n\n")
	sb.WriteString("```json\n")
	jsonData, _ := json.MarshalIndent(stats, "", "  ")
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
