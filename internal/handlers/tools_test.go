package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/matthieu/mcp-server-prtg/internal/types"
)

// MockDB is a mock implementation of database.DB
type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetSensors(ctx context.Context, deviceName, sensorName string, status *int, tags string, limit int) ([]types.Sensor, error) {
	args := m.Called(ctx, deviceName, sensorName, status, tags, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Sensor), args.Error(1)
}

func (m *MockDB) GetSensorsExtended(ctx context.Context, deviceName, sensorName, sensorType, groupName string, status *int, tags, orderBy string, limit int) ([]types.Sensor, error) {
	args := m.Called(ctx, deviceName, sensorName, sensorType, groupName, status, tags, orderBy, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Sensor), args.Error(1)
}

func (m *MockDB) GetSensorByID(ctx context.Context, sensorID int) (*types.Sensor, error) {
	args := m.Called(ctx, sensorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Sensor), args.Error(1)
}

func (m *MockDB) GetAlerts(ctx context.Context, hours int, status *int, deviceName string) ([]types.Sensor, error) {
	args := m.Called(ctx, hours, status, deviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Sensor), args.Error(1)
}

func (m *MockDB) GetDeviceOverview(ctx context.Context, deviceName string) (*types.DeviceOverview, error) {
	args := m.Called(ctx, deviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DeviceOverview), args.Error(1)
}

func (m *MockDB) GetTopSensors(ctx context.Context, metric, sensorType string, limit, hours int) ([]types.Sensor, error) {
	args := m.Called(ctx, metric, sensorType, limit, hours)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Sensor), args.Error(1)
}

func (m *MockDB) GetHierarchy(ctx context.Context, groupName string, includeSensors bool, maxDepth int) (*types.HierarchyNode, error) {
	args := m.Called(ctx, groupName, includeSensors, maxDepth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.HierarchyNode), args.Error(1)
}

func (m *MockDB) Search(ctx context.Context, searchTerm string, limit int) (*types.SearchResults, error) {
	args := m.Called(ctx, searchTerm, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SearchResults), args.Error(1)
}

func (m *MockDB) ExecuteCustomQuery(ctx context.Context, query string, limit int) ([]map[string]interface{}, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockConfig is a mock implementation of Config interface
type MockConfig struct {
	allowCustomQueries bool
}

func (m *MockConfig) AllowCustomQueries() bool {
	return m.allowCustomQueries
}

// Helper to create test logger
func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

// Helper to create test request
func createTestRequest(arguments map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "test_tool",
			Arguments: arguments,
		},
	}
}

// Test parseArguments
func TestParseArguments(t *testing.T) {
	t.Run("Valid arguments", func(t *testing.T) {
		args := map[string]interface{}{
			"sensor_id": float64(123), // JSON numbers are float64
			"limit":     float64(50),
		}

		var target struct {
			SensorID int `json:"sensor_id"`
			Limit    int `json:"limit"`
		}

		err := parseArguments(args, &target)
		assert.NoError(t, err)
		assert.Equal(t, 123, target.SensorID)
		assert.Equal(t, 50, target.Limit)
	})

	t.Run("Invalid JSON - circular reference", func(t *testing.T) {
		// Create a circular reference that cannot be marshalled
		type Node struct {
			Next *Node
		}
		node := &Node{}
		node.Next = node // Circular reference

		var target struct {
			Data string `json:"data"`
		}

		// This should fail during json.Marshal
		err := parseArguments(node, &target)
		assert.Error(t, err)
	})

	t.Run("Type mismatch", func(t *testing.T) {
		args := map[string]interface{}{
			"sensor_id": "not-a-number", // String instead of int
		}

		var target struct {
			SensorID int `json:"sensor_id"`
		}

		err := parseArguments(args, &target)
		assert.Error(t, err)
	})
}

// Test formatResult
func TestFormatResult(t *testing.T) {
	t.Run("Format single sensor", func(t *testing.T) {
		sensor := types.Sensor{
			ID:   123,
			Name: "Test Sensor",
		}

		result, err := formatResult(sensor, 1)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, "Found 1 result(s)")
		assert.Contains(t, textContent.Text, "Test Sensor")
	})

	t.Run("Format multiple sensors", func(t *testing.T) {
		sensors := []types.Sensor{
			{ID: 1, Name: "Sensor 1"},
			{ID: 2, Name: "Sensor 2"},
		}

		result, err := formatResult(sensors, 2)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(mcp.TextContent)
		assert.True(t, ok)
		assert.Contains(t, textContent.Text, "Found 2 result(s)")
		assert.Contains(t, textContent.Text, "Sensor 1")
		assert.Contains(t, textContent.Text, "Sensor 2")
	})

	t.Run("Format unmarshallable data", func(t *testing.T) {
		// Channels cannot be marshalled to JSON
		data := make(chan int)

		result, err := formatResult(data, 1)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to marshal result")
	})
}

// Test handleGetSensorStatus
func TestHandleGetSensorStatus(t *testing.T) {
	t.Run("Valid sensor ID", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensor := &types.Sensor{
			ID:   123,
			Name: "Test Sensor",
		}

		mockDB.On("GetSensorByID", mock.Anything, 123).Return(expectedSensor, nil)

		request := createTestRequest(map[string]interface{}{
			"sensor_id": float64(123),
		})

		result, err := handler.handleGetSensorStatus(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Invalid sensor ID - zero", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{
			"sensor_id": float64(0),
		})

		result, err := handler.handleGetSensorStatus(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sensor_id must be greater than 0")

		mockDB.AssertNotCalled(t, "GetSensorByID")
	})

	t.Run("Invalid sensor ID - negative", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{
			"sensor_id": float64(-5),
		})

		result, err := handler.handleGetSensorStatus(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sensor_id must be greater than 0")

		mockDB.AssertNotCalled(t, "GetSensorByID")
	})

	t.Run("Missing sensor_id argument", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{})

		result, err := handler.handleGetSensorStatus(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "sensor_id must be greater than 0")

		mockDB.AssertNotCalled(t, "GetSensorByID")
	})
}

// Test handleDeviceOverview
func TestHandleDeviceOverview(t *testing.T) {
	t.Run("Valid device name", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedOverview := &types.DeviceOverview{
			Device: types.Device{
				Name: "Server1",
			},
			TotalSensors: 10,
			UpSensors:    8,
			DownSensors:  2,
		}

		mockDB.On("GetDeviceOverview", mock.Anything, "Server1").Return(expectedOverview, nil)

		request := createTestRequest(map[string]interface{}{
			"device_name": "Server1",
		})

		result, err := handler.handleDeviceOverview(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Empty device name", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{
			"device_name": "",
		})

		result, err := handler.handleDeviceOverview(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "device_name is required")

		mockDB.AssertNotCalled(t, "GetDeviceOverview")
	})

	t.Run("Missing device_name argument", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{})

		result, err := handler.handleDeviceOverview(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "device_name is required")

		mockDB.AssertNotCalled(t, "GetDeviceOverview")
	})
}

// Test handleCustomQuery - SECURITY CRITICAL
func TestHandleCustomQuery_Security(t *testing.T) {
	t.Run("Custom queries disabled - SECURITY", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false} // DISABLED
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{
			"query": "SELECT * FROM prtg_sensor",
			"limit": float64(100),
		})

		result, err := handler.handleCustomQuery(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "custom SQL queries are disabled")
		assert.Contains(t, err.Error(), "allow_custom_queries: true")

		// Database should NOT be called when queries are disabled
		mockDB.AssertNotCalled(t, "ExecuteCustomQuery")
	})

	t.Run("Custom queries enabled - valid query", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: true} // ENABLED
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedResults := []map[string]interface{}{
			{"id": 1, "name": "Sensor1"},
			{"id": 2, "name": "Sensor2"},
		}

		mockDB.On("ExecuteCustomQuery", mock.Anything, "SELECT * FROM prtg_sensor", 100).
			Return(expectedResults, nil)

		request := createTestRequest(map[string]interface{}{
			"query": "SELECT * FROM prtg_sensor",
			"limit": float64(100),
		})

		result, err := handler.handleCustomQuery(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Empty query", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: true}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		request := createTestRequest(map[string]interface{}{
			"query": "",
		})

		result, err := handler.handleCustomQuery(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "query is required")

		mockDB.AssertNotCalled(t, "ExecuteCustomQuery")
	})

	t.Run("Default limit applied", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: true}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedResults := []map[string]interface{}{}

		// Should use default limit of 100
		mockDB.On("ExecuteCustomQuery", mock.Anything, "SELECT * FROM prtg_sensor", 100).
			Return(expectedResults, nil)

		request := createTestRequest(map[string]interface{}{
			"query": "SELECT * FROM prtg_sensor",
			// No limit specified
		})

		result, err := handler.handleCustomQuery(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})
}

// Test handleGetSensors - default values
func TestHandleGetSensors_Defaults(t *testing.T) {
	t.Run("Default limit applied", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{
			{ID: 1, Name: "Sensor1"},
		}

		// Should use default limit of 1000 when limit <= 0
		mockDB.On("GetSensorsExtended", mock.Anything, "", "", "", "", (*int)(nil), "", "name", 1000).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			"limit": float64(0), // Zero or negative should default to 1000
		})

		result, err := handler.handleGetSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Negative limit corrected", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		mockDB.On("GetSensorsExtended", mock.Anything, "", "", "", "", (*int)(nil), "", "name", 1000).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			"limit": float64(-10),
		})

		result, err := handler.handleGetSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})
}

// Test handleGetAlerts - default values
func TestHandleGetAlerts_Defaults(t *testing.T) {
	t.Run("Default hours applied", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		// Should use default hours of 24
		mockDB.On("GetAlerts", mock.Anything, 24, (*int)(nil), "").
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			// No hours specified
		})

		result, err := handler.handleGetAlerts(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})
}

// Test handleTopSensors - default values and validation
func TestHandleTopSensors_Defaults(t *testing.T) {
	t.Run("All defaults applied", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		// Should use defaults: metric="downtime", limit=10, hours=24
		mockDB.On("GetTopSensors", mock.Anything, "downtime", "", 10, 24).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			// No arguments specified
		})

		result, err := handler.handleTopSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Custom metric", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		mockDB.On("GetTopSensors", mock.Anything, "uptime", "", 10, 24).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			"metric": "uptime",
		})

		result, err := handler.handleTopSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Negative limit corrected", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		// Should correct negative limit to default 10
		mockDB.On("GetTopSensors", mock.Anything, "downtime", "", 10, 24).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			"limit": float64(-5),
		})

		result, err := handler.handleTopSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})

	t.Run("Negative hours corrected", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		expectedSensors := []types.Sensor{}

		// Should correct negative hours to default 24
		mockDB.On("GetTopSensors", mock.Anything, "downtime", "", 10, 24).
			Return(expectedSensors, nil)

		request := createTestRequest(map[string]interface{}{
			"hours": float64(-10),
		})

		result, err := handler.handleTopSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})
}

// Test context timeout is applied
func TestHandleGetSensors_ContextTimeout(t *testing.T) {
	t.Run("Context timeout is applied", func(t *testing.T) {
		mockDB := new(MockDB)
		mockConfig := &MockConfig{allowCustomQueries: false}
		logger := newTestLogger()

		handler := NewToolHandler(mockDB, mockConfig, logger)

		// Mock should receive a context with timeout
		mockDB.On("GetSensorsExtended", mock.MatchedBy(func(ctx context.Context) bool {
			deadline, ok := ctx.Deadline()
			if !ok {
				return false
			}
			// Should have a deadline within ~30 seconds from now
			timeUntilDeadline := time.Until(deadline)
			return timeUntilDeadline > 29*time.Second && timeUntilDeadline <= 30*time.Second
		}), "", "", "", "", (*int)(nil), "", "name", 1000).
			Return([]types.Sensor{}, nil)

		request := createTestRequest(map[string]interface{}{})

		result, err := handler.handleGetSensors(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		mockDB.AssertExpectations(t)
	})
}

// Benchmark tests
func BenchmarkParseArguments(b *testing.B) {
	args := map[string]interface{}{
		"sensor_id": float64(123),
		"limit":     float64(50),
	}

	var target struct {
		SensorID int `json:"sensor_id"`
		Limit    int `json:"limit"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseArguments(args, &target)
	}
}

func BenchmarkFormatResult(b *testing.B) {
	sensors := []types.Sensor{
		{ID: 1, Name: "Sensor 1"},
		{ID: 2, Name: "Sensor 2"},
		{ID: 3, Name: "Sensor 3"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = formatResult(sensors, len(sensors))
	}
}
