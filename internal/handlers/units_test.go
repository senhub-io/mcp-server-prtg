package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayValue(t *testing.T) {
	tests := []struct {
		name string
		raw  float64
		unit string
		want float64
	}{
		{"field example go", 19199197184, "Go", 17.88},
		{"field example sum go", 63845691392, "go", 59.46},
		{"gb", 1024 * 1024 * 1024, "GB", 1},
		{"gib mixed case", 2 * 1024 * 1024 * 1024, "GiB", 2},
		{"kb", 2048, "kb", 2},
		{"kib", 3072, "KiB", 3},
		{"mo", 5 * 1024 * 1024, "mo", 5},
		{"mb padded", 7 * 1024 * 1024, " MB ", 7},
		{"to", 1024 * 1024 * 1024 * 1024, "To", 1},
		{"tb", 2 * 1024 * 1024 * 1024 * 1024, "TB", 2},
		{"percent unchanged", 30.07, "%", 30.07},
		{"kbit/s unchanged", 123.4, "kbit/s", 123.4},
		{"mbit/s unchanged", 9.9, "Mbit/s", 9.9},
		{"msec unchanged", 42, "msec", 42},
		{"celsius unchanged", 21.5, "°C", 21.5},
		{"count unchanged", 7, "#", 7},
		{"empty unchanged", 42, "", 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, displayValue(tt.raw, tt.unit), 0.01)
		})
	}
}
