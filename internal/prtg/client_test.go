package prtg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// setupTestClient creates a test PRTG client with a mock HTTP server.
func setupTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	logger := zerolog.Nop()
	config := ClientConfig{
		BaseURL:   server.URL,
		Token:     "test-token",
		Timeout:   5 * time.Second,
		VerifySSL: true,
		Logger:    &logger,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client, server
}

func TestNewClient(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name    string
		config  ClientConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: ClientConfig{
				BaseURL:   "https://prtg.example.com",
				Token:     "valid-token",
				Timeout:   30 * time.Second,
				VerifySSL: true,
				Logger:    &logger,
			},
			wantErr: nil,
		},
		{
			name: "empty base URL",
			config: ClientConfig{
				BaseURL:   "",
				Token:     "valid-token",
				Timeout:   30 * time.Second,
				VerifySSL: true,
				Logger:    &logger,
			},
			wantErr: ErrInvalidBaseURL,
		},
		{
			name: "empty token",
			config: ClientConfig{
				BaseURL:   "https://prtg.example.com",
				Token:     "",
				Timeout:   30 * time.Second,
				VerifySSL: true,
				Logger:    &logger,
			},
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if err != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetTimeSeries(t *testing.T) {
	mockResponse := TimeSeriesResponse{
		Headers: []string{"timestamp", "CPU Load", "Memory Usage"},
		Data: [][]interface{}{
			{"2025-10-31T10:00:00Z", 45.2, 2048.0},
			{"2025-10-31T10:05:00Z", 48.5, 2100.0},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/experimental/timeseries/1234/short" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("Missing or incorrect Authorization header")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}

	client, server := setupTestClient(t, handler)
	defer server.Close()

	ctx := context.Background()
	data, err := client.GetTimeSeries(ctx, 1234, TimeSeriesShort)
	if err != nil {
		t.Fatalf("GetTimeSeries() error = %v", err)
	}

	if data.ObjectID != 1234 {
		t.Errorf("ObjectID = %d, want 1234", data.ObjectID)
	}

	if data.TimeType != TimeSeriesShort {
		t.Errorf("TimeType = %s, want %s", data.TimeType, TimeSeriesShort)
	}

	if len(data.DataPoints) != 2 {
		t.Errorf("len(DataPoints) = %d, want 2", len(data.DataPoints))
	}

	if len(data.Headers) != 3 {
		t.Errorf("len(Headers) = %d, want 3", len(data.Headers))
	}
}

func TestClient_GetTimeSeriesCustom(t *testing.T) {
	mockResponse := TimeSeriesResponse{
		Headers: []string{"timestamp", "Traffic In", "Traffic Out"},
		Data: [][]interface{}{
			{"2025-10-30T00:00:00Z", 1024.0, 512.0},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if !contains(r.URL.Path, "/api/v2/experimental/timeseries/5678") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Check query parameters
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		if start == "" || end == "" {
			t.Error("Missing start or end query parameters")
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}

	client, server := setupTestClient(t, handler)
	defer server.Close()

	ctx := context.Background()
	start := time.Date(2025, 10, 30, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)

	data, err := client.GetTimeSeriesCustom(ctx, 5678, start, end)
	if err != nil {
		t.Fatalf("GetTimeSeriesCustom() error = %v", err)
	}

	if data.ObjectID != 5678 {
		t.Errorf("ObjectID = %d, want 5678", data.ObjectID)
	}

	if data.StartTime == nil || data.EndTime == nil {
		t.Error("StartTime or EndTime is nil")
	}

	if len(data.DataPoints) != 1 {
		t.Errorf("len(DataPoints) = %d, want 1", len(data.DataPoints))
	}
}

func TestClient_GetChannelsBySensor(t *testing.T) {
	mockResponse := ChannelsResponse{
		Channels: []Channel{
			{
				ID:        1,
				ObjectID:  1234,
				Name:      "CPU Load",
				LastValue: "45.2 %",
				Unit:      "%",
			},
			{
				ID:        2,
				ObjectID:  1234,
				Name:      "Memory Usage",
				LastValue: "2048 MB",
				Unit:      "MB",
			},
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/experimental/channels" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Check filter parameter
		filterObjID := r.URL.Query().Get("filter_objid")
		if filterObjID != "1234" {
			t.Errorf("filter_objid = %s, want 1234", filterObjID)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Fatalf("Failed to encode response: %v", err)
		}
	}

	client, server := setupTestClient(t, handler)
	defer server.Close()

	ctx := context.Background()
	channels, err := client.GetChannelsBySensor(ctx, 1234)
	if err != nil {
		t.Fatalf("GetChannelsBySensor() error = %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("len(channels) = %d, want 2", len(channels))
	}

	if channels[0].Name != "CPU Load" {
		t.Errorf("channels[0].Name = %s, want CPU Load", channels[0].Name)
	}
}

func TestClient_HandleHTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    ErrUnauthorized,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantErr:    ErrNotFound,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			wantErr:    ErrRateLimited,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    ErrServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("Error message"))
			}

			client, server := setupTestClient(t, handler)
			defer server.Close()

			ctx := context.Background()
			_, err := client.GetTimeSeries(ctx, 1234, TimeSeriesShort)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check if error wraps the expected error type
			if !contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("Error = %v, want error containing %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Ping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v2/health" {
					t.Errorf("Unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
			}

			client, server := setupTestClient(t, handler)
			defer server.Close()

			ctx := context.Background()
			err := client.Ping(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "RFC3339 string",
			input:   "2025-10-31T10:00:00Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with offset",
			input:   "2025-10-31T10:00:00+01:00",
			wantErr: false,
		},
		{
			name:    "Unix timestamp float",
			input:   float64(1698753600),
			wantErr: false,
		},
		{
			name:    "Unix timestamp int",
			input:   int64(1698753600),
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "not-a-timestamp",
			wantErr: true,
		},
		{
			name:    "invalid type",
			input:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
