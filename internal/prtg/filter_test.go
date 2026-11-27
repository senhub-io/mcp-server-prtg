package prtg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "Core Switch", `"Core Switch"`},
		{"injection", `x" or status="down`, `"x or status=down"`},
		{"backslash", `a\b`, `"ab"`},
		{"control char", "line\nbreak", `"linebreak"`},
		{"padding", "  pad  ", `"pad"`},
		{"empty", "", `""`},
		{"unicode", "Sécurité", `"Sécurité"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FilterText(tt.in))
		})
	}
}
