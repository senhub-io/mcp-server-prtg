package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/prtg"
	"github.com/stretchr/testify/assert"
)

func TestFormatTimeSeriesForLLM_WithSparklines(t *testing.T) {
	t.Run("Empty data", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			ObjectID:   123,
			DataPoints: []prtg.TimeSeriesDataPoint{},
		}
		result := formatTimeSeriesForLLM(data)
		assert.Contains(t, result, "No data available")
	})

	t.Run("Data with sparklines and stats", func(t *testing.T) {
		now := time.Now()
		data := &prtg.TimeSeriesData{
			ObjectID: 123,
			TimeType: "short",
			Headers:  []string{"Timestamp", "Response Time"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{
					Timestamp: now.Add(-4 * time.Hour),
					Values:    map[string]interface{}{"Response Time": 10.0},
				},
				{
					Timestamp: now.Add(-3 * time.Hour),
					Values:    map[string]interface{}{"Response Time": 20.0},
				},
				{
					Timestamp: now.Add(-2 * time.Hour),
					Values:    map[string]interface{}{"Response Time": 30.0},
				},
				{
					Timestamp: now.Add(-1 * time.Hour),
					Values:    map[string]interface{}{"Response Time": 40.0},
				},
				{
					Timestamp: now,
					Values:    map[string]interface{}{"Response Time": 50.0},
				},
			},
		}

		result := formatTimeSeriesForLLM(data)

		// Check header
		assert.Contains(t, result, "Time Series Data - Sensor 123")
		assert.Contains(t, result, "short")

		// Check summary
		assert.Contains(t, result, "Total data points: 5")
		assert.Contains(t, result, "Response Time")

		// Check sparkline section
		assert.Contains(t, result, "## Quick Trend")
		assert.Contains(t, result, "**Sparkline (Response Time):**")
		assert.Contains(t, result, "**Stats:**")

		// Check trend indicator (should be UP for increasing values)
		assert.Contains(t, result, "UP")

		// Check stats
		assert.Contains(t, result, "Min:")
		assert.Contains(t, result, "Max:")
		assert.Contains(t, result, "Avg:")
		assert.Contains(t, result, "Current:")

		// Check measurements section
		assert.Contains(t, result, "## Measurements")
	})

	t.Run("Data with anomaly detection", func(t *testing.T) {
		now := time.Now()
		// Create data with a spike (anomaly)
		values := []interface{}{10.0, 10.0, 10.0, 10.0, 10.0, 100.0, 10.0, 10.0, 10.0, 10.0, 10.0}
		dataPoints := make([]prtg.TimeSeriesDataPoint, len(values))
		for i, v := range values {
			dataPoints[i] = prtg.TimeSeriesDataPoint{
				Timestamp: now.Add(time.Duration(-i) * time.Hour),
				Values:    map[string]interface{}{"Response Time": v},
			}
		}

		data := &prtg.TimeSeriesData{
			ObjectID:   456,
			TimeType:   "short",
			Headers:    []string{"Timestamp", "Response Time"},
			DataPoints: dataPoints,
		}

		result := formatTimeSeriesForLLM(data)

		// Check anomaly detection
		assert.Contains(t, result, "WARNING:")
		assert.Contains(t, result, "anomaly")
		assert.Contains(t, result, "expected:")
	})

	t.Run("Data with no numeric channels", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			ObjectID: 789,
			TimeType: "short",
			Headers:  []string{"Timestamp"}, // Only timestamp, no other channels
			DataPoints: []prtg.TimeSeriesDataPoint{
				{
					Timestamp: time.Now(),
					Values:    map[string]interface{}{},
				},
			},
		}

		result := formatTimeSeriesForLLM(data)

		// Should not contain sparkline section
		assert.NotContains(t, result, "## Quick Trend")
		assert.Contains(t, result, "Time Series Data")
	})
}

func TestExtractChannelValues(t *testing.T) {
	t.Run("Extract float64 values", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			Headers: []string{"Timestamp", "CPU"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Values: map[string]interface{}{"CPU": 10.5}},
				{Values: map[string]interface{}{"CPU": 20.5}},
				{Values: map[string]interface{}{"CPU": 30.5}},
			},
		}

		values := extractChannelValues(data, "CPU")
		assert.Equal(t, []float64{10.5, 20.5, 30.5}, values)
	})

	t.Run("Extract int values", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			Headers: []string{"Timestamp", "Count"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Values: map[string]interface{}{"Count": 10}},
				{Values: map[string]interface{}{"Count": 20}},
			},
		}

		values := extractChannelValues(data, "Count")
		assert.Equal(t, []float64{10.0, 20.0}, values)
	})

	t.Run("Extract int64 values", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			Headers: []string{"Timestamp", "BigCount"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Values: map[string]interface{}{"BigCount": int64(1000)}},
				{Values: map[string]interface{}{"BigCount": int64(2000)}},
			},
		}

		values := extractChannelValues(data, "BigCount")
		assert.Equal(t, []float64{1000.0, 2000.0}, values)
	})

	t.Run("Skip non-numeric values", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			Headers: []string{"Timestamp", "Status"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Values: map[string]interface{}{"Status": "OK"}},
				{Values: map[string]interface{}{"Status": "ERROR"}},
			},
		}

		values := extractChannelValues(data, "Status")
		assert.Empty(t, values)
	})

	t.Run("Missing channel returns empty", func(t *testing.T) {
		data := &prtg.TimeSeriesData{
			Headers: []string{"Timestamp", "CPU"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Values: map[string]interface{}{"CPU": 10.0}},
			},
		}

		values := extractChannelValues(data, "Memory")
		assert.Empty(t, values)
	})
}

func TestFormatTimeSeriesForLLM_SparklineIntegration(t *testing.T) {
	t.Run("Sparkline visible in output", func(t *testing.T) {
		now := time.Now()
		data := &prtg.TimeSeriesData{
			ObjectID: 100,
			TimeType: "medium",
			Headers:  []string{"Timestamp", "Bandwidth"},
			DataPoints: []prtg.TimeSeriesDataPoint{
				{Timestamp: now.Add(-2 * time.Hour), Values: map[string]interface{}{"Bandwidth": 100.0}},
				{Timestamp: now.Add(-1 * time.Hour), Values: map[string]interface{}{"Bandwidth": 200.0}},
				{Timestamp: now, Values: map[string]interface{}{"Bandwidth": 300.0}},
			},
		}

		result := formatTimeSeriesForLLM(data)

		// Verify sparkline characters are present
		hasSparkline := strings.ContainsAny(result, "▁▂▃▄▅▆▇█")
		assert.True(t, hasSparkline, "Output should contain sparkline characters")

		// Verify trend indicator
		hasTrend := strings.Contains(result, "UP") ||
			strings.Contains(result, "DOWN") ||
			strings.Contains(result, "FLAT")
		assert.True(t, hasTrend, "Output should contain trend indicator")
	})
}
