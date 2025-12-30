package handlers

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestErrorGuidance(t *testing.T) {
	t.Run("Error with suggestions", func(t *testing.T) {
		result := ErrorGuidance("Test Error", "This is a test", []string{"Do this", "Try that"})
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, "ERROR: Test Error")
		assert.Contains(t, textContent.Text, "This is a test")
		assert.Contains(t, textContent.Text, "How to resolve:")
		assert.Contains(t, textContent.Text, "1. Do this")
		assert.Contains(t, textContent.Text, "2. Try that")
	})

	t.Run("Error without suggestions", func(t *testing.T) {
		result := ErrorGuidance("Simple Error", "No suggestions", []string{})
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "ERROR: Simple Error")
		assert.Contains(t, textContent.Text, "No suggestions")
		assert.NotContains(t, textContent.Text, "How to resolve")
	})

	t.Run("Error with nil suggestions", func(t *testing.T) {
		result := ErrorGuidance("Another Error", "Test explanation", nil)
		assert.NotNil(t, result)

		textContent := result.Content[0].(mcp.TextContent)
		assert.NotContains(t, textContent.Text, "How to resolve")
	})

	t.Run("Error with multiple suggestions", func(t *testing.T) {
		suggestions := []string{"First step", "Second step", "Third step", "Fourth step"}
		result := ErrorGuidance("Multi-step Error", "Complex issue", suggestions)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "1. First step")
		assert.Contains(t, textContent.Text, "2. Second step")
		assert.Contains(t, textContent.Text, "3. Third step")
		assert.Contains(t, textContent.Text, "4. Fourth step")
	})
}

func TestErrInvalidSensorID(t *testing.T) {
	result := ErrInvalidSensorID()
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Invalid sensor_id")
	assert.Contains(t, textContent.Text, "positive integer")
	assert.Contains(t, textContent.Text, "prtg_search")
	assert.Contains(t, textContent.Text, "prtg_get_alerts")
	assert.Contains(t, textContent.Text, "prtg_device_overview")
}

func TestErrDeviceNotFound(t *testing.T) {
	t.Run("Device not found with name", func(t *testing.T) {
		deviceName := "MyServer"
		result := ErrDeviceNotFound(deviceName)
		assert.NotNil(t, result)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "MyServer")
		assert.Contains(t, textContent.Text, "not found")
		assert.Contains(t, textContent.Text, "prtg_search")
		assert.Contains(t, textContent.Text, "prtg_get_hierarchy")
	})

	t.Run("Device not found with special characters", func(t *testing.T) {
		deviceName := "Server-123.local"
		result := ErrDeviceNotFound(deviceName)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "Server-123.local")
	})
}

func TestErrSensorNotFound(t *testing.T) {
	t.Run("Sensor not found with ID", func(t *testing.T) {
		sensorID := 12345
		result := ErrSensorNotFound(sensorID)
		assert.NotNil(t, result)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "12345")
		assert.Contains(t, textContent.Text, "not found")
		assert.Contains(t, textContent.Text, "does not exist")
	})

	t.Run("Sensor not found with zero ID", func(t *testing.T) {
		result := ErrSensorNotFound(0)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "0")
	})
}

func TestErrEmptySearchTerm(t *testing.T) {
	result := ErrEmptySearchTerm()
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Empty search term")
	assert.Contains(t, textContent.Text, "required")
	assert.Contains(t, textContent.Text, "prtg_search")
	assert.Contains(t, textContent.Text, "server")
	assert.Contains(t, textContent.Text, "ping")
	assert.Contains(t, textContent.Text, "production")
}

func TestErrInvalidTimeRange(t *testing.T) {
	result := ErrInvalidTimeRange()
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Invalid time range")
	assert.Contains(t, textContent.Text, "end_time must be after start_time")
	assert.Contains(t, textContent.Text, "RFC3339")
	assert.Contains(t, textContent.Text, "time_type")
	assert.Contains(t, textContent.Text, "live")
	assert.Contains(t, textContent.Text, "short")
	assert.Contains(t, textContent.Text, "medium")
	assert.Contains(t, textContent.Text, "long")
}

func TestErrCustomQueriesDisabled(t *testing.T) {
	result := ErrCustomQueriesDisabled()
	assert.NotNil(t, result)
	assert.Len(t, result.Content, 1)

	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Custom SQL queries disabled")
	assert.Contains(t, textContent.Text, "security")
	assert.Contains(t, textContent.Text, "allow_custom_queries")
	assert.Contains(t, textContent.Text, "config.yaml")
	assert.Contains(t, textContent.Text, "prtg_get_sensors")
	assert.Contains(t, textContent.Text, "prtg_get_alerts")
}

func TestErrorGuidanceFormat(t *testing.T) {
	t.Run("Check error format structure", func(t *testing.T) {
		result := ErrorGuidance("Title", "Explanation", []string{"Step 1", "Step 2"})
		textContent := result.Content[0].(mcp.TextContent)

		// Should start with ERROR:
		assert.Contains(t, textContent.Text, "ERROR:")

		// Should have proper spacing
		lines := len(textContent.Text)
		assert.Greater(t, lines, 0)

		// Should be properly structured
		assert.NotEmpty(t, textContent.Text)
	})

	t.Run("Empty title and explanation", func(t *testing.T) {
		result := ErrorGuidance("", "", []string{"Suggestion"})
		assert.NotNil(t, result)

		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "1. Suggestion")
	})
}

func TestAllPredefinedErrorsReturnNonNil(t *testing.T) {
	t.Run("All predefined errors return valid results", func(t *testing.T) {
		errors := []func() *mcp.CallToolResult{
			ErrInvalidSensorID,
			ErrEmptySearchTerm,
			ErrInvalidTimeRange,
			ErrCustomQueriesDisabled,
		}

		for _, errFunc := range errors {
			result := errFunc()
			assert.NotNil(t, result, "Error function should not return nil")
			assert.Len(t, result.Content, 1, "Error should have exactly 1 content item")
			assert.IsType(t, mcp.TextContent{}, result.Content[0], "Content should be TextContent")
		}
	})

	t.Run("Parameterized errors return valid results", func(t *testing.T) {
		assert.NotNil(t, ErrDeviceNotFound("test"))
		assert.NotNil(t, ErrSensorNotFound(123))
	})
}
