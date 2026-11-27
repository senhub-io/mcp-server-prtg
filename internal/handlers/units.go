package handlers

import "strings"

// displayValue converts a raw database value to PRTG's display value for byte-size units.
// The Data Exporter stores raw bytes while the unit column holds PRTG's display unit (base 1024).
func displayValue(raw float64, unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "ko", "kb", "kib":
		return raw / 1024
	case "mo", "mb", "mib":
		return raw / (1024 * 1024)
	case "go", "gb", "gib":
		return raw / (1024 * 1024 * 1024)
	case "to", "tb", "tib":
		return raw / (1024 * 1024 * 1024 * 1024)
	}
	return raw
}
