package prtg

import (
	"strings"
	"unicode"
)

// FilterText returns a PRTG API v2 filter string literal for untrusted user
// input: the input trimmed of surrounding spaces and stripped of double
// quotes, backslashes, and control characters, then wrapped in double
// quotes. Quotes and backslashes are dropped because PRTG defines no escape
// syntax inside string literals, so a value cannot close the literal and
// inject filter operators.
func FilterText(s string) string {
	var b []rune
	for _, r := range s {
		if r == '"' || r == '\\' || unicode.IsControl(r) {
			continue
		}
		b = append(b, r)
	}
	return `"` + strings.TrimSpace(string(b)) + `"`
}
