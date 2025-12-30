package handlers

import (
	"fmt"
	"math"
	"strings"
)

// Sparkline generates a compact ASCII visualization of values
// Example: ▁▂▃▅▇▅▃▂▁
func Sparkline(values []float64, maxWidth int) string {
	if len(values) == 0 {
		return ""
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find min/max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	range_ := max - min
	if range_ == 0 {
		range_ = 1
	}

	// Sample if too many values
	step := 1
	if len(values) > maxWidth {
		step = len(values) / maxWidth
	}

	var result strings.Builder
	for i := 0; i < len(values); i += step {
		idx := int((values[i] - min) / range_ * 7)
		if idx > 7 {
			idx = 7
		}
		if idx < 0 {
			idx = 0
		}
		result.WriteRune(blocks[idx])
	}

	return result.String()
}

// TrendIndicator returns a trend indicator
// UP for rising, DOWN for falling, FLAT for stable
func TrendIndicator(values []float64) string {
	if len(values) < 2 {
		return "FLAT"
	}

	// Compare average of first half vs second half
	mid := len(values) / 2
	var firstHalf, secondHalf float64

	for i := 0; i < mid; i++ {
		firstHalf += values[i]
	}
	for i := mid; i < len(values); i++ {
		secondHalf += values[i]
	}

	firstAvg := firstHalf / float64(mid)
	secondAvg := secondHalf / float64(len(values)-mid)

	diff := (secondAvg - firstAvg) / firstAvg * 100

	if diff > 10 {
		return "UP"
	} else if diff < -10 {
		return "DOWN"
	}
	return "FLAT"
}

// CompactStats returns a compact statistical summary
func CompactStats(values []float64) string {
	if len(values) == 0 {
		return "No data"
	}

	min, max, sum := values[0], values[0], 0.0
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg := sum / float64(len(values))
	current := values[len(values)-1]

	return fmt.Sprintf("Min: %.1f | Max: %.1f | Avg: %.1f | Current: %.1f",
		min, max, avg, current)
}

// Anomaly represents an anomalous data point
type Anomaly struct {
	Index    int
	Value    float64
	Expected float64
}

// DetectAnomalies detects anomalous values (>2 standard deviations)
func DetectAnomalies(values []float64) []Anomaly {
	if len(values) < 10 {
		return nil
	}

	// Calculate mean and standard deviation
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	stdDev := math.Sqrt(variance / float64(len(values)))

	var anomalies []Anomaly
	threshold := 2.0 * stdDev

	for i, v := range values {
		if math.Abs(v-mean) > threshold {
			anomalies = append(anomalies, Anomaly{
				Index:    i,
				Value:    v,
				Expected: mean,
			})
		}
	}

	return anomalies
}

// HealthBar generates a visual health bar
// Example: [████████░░] 80%
func HealthBar(percentage float64, width int) string {
	if percentage > 100 {
		percentage = 100
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(percentage / 100 * float64(width))
	empty := width - filled

	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.Repeat("█", filled))
	sb.WriteString(strings.Repeat("░", empty))
	sb.WriteString("]")
	sb.WriteString(fmt.Sprintf(" %.0f%%", percentage))

	return sb.String()
}
