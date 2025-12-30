package handlers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSparkline(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		maxWidth int
		wantLen  int
	}{
		{"empty", []float64{}, 10, 0},
		{"single", []float64{5}, 10, 1},
		{"ascending", []float64{1, 2, 3, 4, 5}, 10, 5},
		{"descending", []float64{5, 4, 3, 2, 1}, 10, 5},
		{"sampled", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 5, 5},
		{"all_same", []float64{5, 5, 5, 5, 5}, 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sparkline(tt.values, tt.maxWidth)
			runeCount := len([]rune(got))
			assert.Equal(t, tt.wantLen, runeCount, "Sparkline length mismatch")
		})
	}
}

func TestSparklineContent(t *testing.T) {
	t.Run("Ascending values show progression", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5}
		result := Sparkline(values, 10)
		assert.NotEmpty(t, result)
		// Should contain block characters
		assert.True(t, strings.ContainsAny(result, "▁▂▃▄▅▆▇█"))
	})

	t.Run("Max width limits output", func(t *testing.T) {
		values := make([]float64, 100)
		for i := range values {
			values[i] = float64(i)
		}
		result := Sparkline(values, 20)
		assert.LessOrEqual(t, len([]rune(result)), 20)
	})
}

func TestTrendIndicator(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   string
	}{
		{"empty", []float64{}, "FLAT"},
		{"single", []float64{10}, "FLAT"},
		{"stable", []float64{10, 10, 10, 10}, "FLAT"},
		{"rising", []float64{10, 10, 20, 20}, "UP"},
		{"falling", []float64{20, 20, 10, 10}, "DOWN"},
		{"slight_rise", []float64{10, 10, 11, 11}, "FLAT"},
		{"steep_rise", []float64{10, 10, 25, 30}, "UP"},
		{"steep_fall", []float64{30, 25, 10, 10}, "DOWN"},
		{"zero_to_positive", []float64{0, 0, 5, 5}, "UP"},
		{"zero_to_negative", []float64{0, 0, -5, -5}, "DOWN"},
		{"all_zeros", []float64{0, 0, 0, 0}, "FLAT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrendIndicator(tt.values)
			assert.Equal(t, tt.want, got, "TrendIndicator mismatch")
		})
	}
}

func TestCompactStats(t *testing.T) {
	t.Run("Empty values", func(t *testing.T) {
		result := CompactStats([]float64{})
		assert.Equal(t, "No data", result)
	})

	t.Run("Single value", func(t *testing.T) {
		result := CompactStats([]float64{42.5})
		assert.Contains(t, result, "42.5")
		assert.Contains(t, result, "Min:")
		assert.Contains(t, result, "Max:")
		assert.Contains(t, result, "Avg:")
		assert.Contains(t, result, "Current:")
	})

	t.Run("Multiple values", func(t *testing.T) {
		result := CompactStats([]float64{10, 20, 30, 40, 50})
		assert.Contains(t, result, "Min: 10.0")
		assert.Contains(t, result, "Max: 50.0")
		assert.Contains(t, result, "Avg: 30.0")
		assert.Contains(t, result, "Current: 50.0")
	})

	t.Run("Decimal values", func(t *testing.T) {
		result := CompactStats([]float64{1.1, 2.2, 3.3})
		assert.Contains(t, result, "1.1")
		assert.Contains(t, result, "3.3")
	})
}

func TestDetectAnomalies(t *testing.T) {
	t.Run("Not enough data", func(t *testing.T) {
		anomalies := DetectAnomalies([]float64{1, 2, 3})
		assert.Nil(t, anomalies)
	})

	t.Run("No anomalies in uniform data", func(t *testing.T) {
		values := []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10}
		anomalies := DetectAnomalies(values)
		assert.Empty(t, anomalies)
	})

	t.Run("Detects spike", func(t *testing.T) {
		values := []float64{10, 10, 10, 10, 100, 10, 10, 10, 10, 10}
		anomalies := DetectAnomalies(values)
		assert.NotEmpty(t, anomalies)
		assert.Equal(t, 4, anomalies[0].Index)
		assert.Equal(t, 100.0, anomalies[0].Value)
	})

	t.Run("Detects drop", func(t *testing.T) {
		values := []float64{100, 100, 100, 100, 10, 100, 100, 100, 100, 100}
		anomalies := DetectAnomalies(values)
		assert.NotEmpty(t, anomalies)
		assert.Equal(t, 4, anomalies[0].Index)
		assert.Equal(t, 10.0, anomalies[0].Value)
	})

	t.Run("Multiple anomalies", func(t *testing.T) {
		values := []float64{10, 10, 10, 10, 10, 100, 10, 10, 10, 100, 10}
		anomalies := DetectAnomalies(values)
		assert.GreaterOrEqual(t, len(anomalies), 1, "Should detect at least one anomaly")
	})
}

func TestHealthBar(t *testing.T) {
	t.Run("Empty bar at 0%", func(t *testing.T) {
		result := HealthBar(0, 10)
		assert.Contains(t, result, "░░░░░░░░░░")
		assert.Contains(t, result, "0%")
	})

	t.Run("Full bar at 100%", func(t *testing.T) {
		result := HealthBar(100, 10)
		assert.Contains(t, result, "██████████")
		assert.Contains(t, result, "100%")
	})

	t.Run("Half bar at 50%", func(t *testing.T) {
		result := HealthBar(50, 10)
		assert.Contains(t, result, "█████")
		assert.Contains(t, result, "░░░░░")
		assert.Contains(t, result, "50%")
	})

	t.Run("Percentage over 100 is capped", func(t *testing.T) {
		result := HealthBar(150, 10)
		assert.Contains(t, result, "100%")
		assert.Contains(t, result, "██████████")
	})

	t.Run("Negative percentage is capped to 0", func(t *testing.T) {
		result := HealthBar(-50, 10)
		assert.Contains(t, result, "0%")
		assert.Contains(t, result, "░░░░░░░░░░")
	})

	t.Run("Different widths", func(t *testing.T) {
		result5 := HealthBar(50, 5)
		result20 := HealthBar(50, 20)
		assert.NotEqual(t, result5, result20)
		assert.Contains(t, result5, "50%")
		assert.Contains(t, result20, "50%")
	})

	t.Run("Format structure", func(t *testing.T) {
		result := HealthBar(75, 10)
		assert.True(t, strings.HasPrefix(result, "["))
		assert.True(t, strings.Contains(result, "]"))
		assert.True(t, strings.HasSuffix(result, "%"))
	})
}

func TestAnomalyStruct(t *testing.T) {
	t.Run("Anomaly fields", func(t *testing.T) {
		anomaly := Anomaly{
			Index:    5,
			Value:    100.0,
			Expected: 50.0,
		}
		assert.Equal(t, 5, anomaly.Index)
		assert.Equal(t, 100.0, anomaly.Value)
		assert.Equal(t, 50.0, anomaly.Expected)
	})
}

func TestSparklineEdgeCases(t *testing.T) {
	t.Run("All zeros", func(t *testing.T) {
		result := Sparkline([]float64{0, 0, 0, 0}, 10)
		assert.NotEmpty(t, result)
		assert.Len(t, []rune(result), 4)
	})

	t.Run("Negative values", func(t *testing.T) {
		result := Sparkline([]float64{-10, -5, 0, 5, 10}, 10)
		assert.NotEmpty(t, result)
		assert.Len(t, []rune(result), 5)
	})

	t.Run("Very large numbers", func(t *testing.T) {
		result := Sparkline([]float64{1000000, 2000000, 3000000}, 10)
		assert.NotEmpty(t, result)
		assert.Len(t, []rune(result), 3)
	})

	t.Run("Very small differences", func(t *testing.T) {
		result := Sparkline([]float64{1.001, 1.002, 1.003}, 10)
		assert.NotEmpty(t, result)
		assert.Len(t, []rune(result), 3)
	})
}
