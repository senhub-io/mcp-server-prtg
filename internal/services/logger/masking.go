package logger

import (
	"io"
	"regexp"
	"strings"
)

// Sensitive patterns to mask in logs.
//
//nolint:gochecknoglobals // Regex patterns are compile-time constants used across all log operations.
var sensitivePatterns = []*regexp.Regexp{
	// Passwords in various formats
	regexp.MustCompile(`(?i)"(password|passwd|pwd)"\s*:\s*"([^"]+)"`),
	regexp.MustCompile(`(?i)(password|passwd|pwd)=([^\s&]+)`),

	// API keys and tokens
	regexp.MustCompile(`(?i)"(token|api[-_]?key|secret|authentication[-_]?key)"\s*:\s*"([^"]+)"`),
	regexp.MustCompile(`(?i)(api[-_]?key|token|secret)=([^\s&]+)`),

	// Authorization headers
	regexp.MustCompile(`(?i)(Authorization|X-API-Key):\s*(Bearer|Basic)?\s*([a-zA-Z0-9+/=._-]+)`),

	// Database connection strings
	regexp.MustCompile(`(?i)(postgres|postgresql|mysql)://([^:]+):([^@]+)@`),

	// UUIDs (potential keys)
	regexp.MustCompile(`([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`),
}

// MaskSensitiveData masks sensitive information in log output.
func MaskSensitiveData(input string) string {
	masked := input

	for _, pattern := range sensitivePatterns {
		masked = pattern.ReplaceAllStringFunc(masked, func(match string) string {
			parts := pattern.FindStringSubmatch(match)
			if len(parts) >= 3 {
				value := parts[len(parts)-1]
				maskedValue := maskValue(value)

				return strings.Replace(match, value, maskedValue, 1)
			}

			return match
		})
	}

	return masked
}

// maskValue masks a sensitive value keeping only first and last characters.
func maskValue(value string) string {
	if value == "" {
		return value
	}

	if len(value) <= 4 {
		return "***"
	}

	// Keep first 2 and last 2 characters
	return value[:2] + "***" + value[len(value)-2:]
}

// MaskingWriter wraps an io.Writer and masks sensitive data before writing.
type MaskingWriter struct {
	writer io.Writer
}

// NewMaskingWriter creates a new masking writer.
func NewMaskingWriter(w io.Writer) *MaskingWriter {
	return &MaskingWriter{writer: w}
}

// Write implements io.Writer interface with automatic masking.
func (m *MaskingWriter) Write(p []byte) (n int, err error) {
	masked := MaskSensitiveData(string(p))
	return m.writer.Write([]byte(masked))
}
