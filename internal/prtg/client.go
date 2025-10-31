package prtg

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Client is a client for the PRTG API v2.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *zerolog.Logger
}

// ClientConfig holds configuration for creating a new PRTG client.
type ClientConfig struct {
	BaseURL   string
	Token     string
	Timeout   time.Duration
	VerifySSL bool
	Logger    *zerolog.Logger
}

// NewClient creates a new PRTG API client.
func NewClient(config ClientConfig) (*Client, error) {
	if config.BaseURL == "" {
		return nil, ErrInvalidBaseURL
	}

	if config.Token == "" {
		return nil, ErrInvalidToken
	}

	// Validate and normalize base URL
	baseURL := strings.TrimSuffix(config.BaseURL, "/")
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBaseURL, err)
	}

	// Configure HTTP client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySSL, // #nosec G402 -- User-configurable via config.yaml
		},
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	client := &Client{
		baseURL:    baseURL,
		token:      config.Token,
		httpClient: httpClient,
		logger:     config.Logger,
	}

	client.logger.Info().
		Str("base_url", baseURL).
		Dur("timeout", config.Timeout).
		Bool("verify_ssl", config.VerifySSL).
		Msg("PRTG API client initialized")

	return client, nil
}

// GetTimeSeries retrieves time series data for a predefined time period.
// objectID: The PRTG object ID (sensor/device/group)
// timeType: The time period type (live, short, medium, long)
func (c *Client) GetTimeSeries(ctx context.Context, objectID int, timeType TimeSeriesType) (*TimeSeriesData, error) {
	endpoint := fmt.Sprintf("/api/v2/experimental/timeseries/%d/%s", objectID, timeType)

	// PRTG API returns array of arrays directly [[timestamp, val1, val2, ...], ...]
	var rawData [][]interface{}
	if err := c.doRequest(ctx, "GET", endpoint, nil, &rawData); err != nil {
		return nil, err
	}

	// Get channel names from /channels endpoint
	channels, err := c.GetChannelsBySensor(ctx, objectID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Could not fetch channel names, using generic names")
		channels = nil
	}

	return c.parseRawTimeSeriesData(objectID, timeType, nil, nil, rawData, channels)
}

// GetTimeSeriesCustom retrieves time series data for a custom time range.
// objectID: The PRTG object ID
// start: Start time (RFC3339)
// end: End time (RFC3339)
func (c *Client) GetTimeSeriesCustom(ctx context.Context, objectID int, start, end time.Time) (*TimeSeriesData, error) {
	endpoint := fmt.Sprintf("/api/v2/experimental/timeseries/%d", objectID)

	// Add query parameters for custom time range
	params := url.Values{}
	params.Set("start", start.Format(time.RFC3339))
	params.Set("end", end.Format(time.RFC3339))

	// PRTG API returns array of arrays directly
	var rawData [][]interface{}
	if err := c.doRequest(ctx, "GET", endpoint+"?"+params.Encode(), nil, &rawData); err != nil {
		return nil, err
	}

	// Get channel names from /channels endpoint
	channels, err := c.GetChannelsBySensor(ctx, objectID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("Could not fetch channel names, using generic names")
		channels = nil
	}

	return c.parseRawTimeSeriesData(objectID, "", &start, &end, rawData, channels)
}

// GetChannels retrieves all channels with optional filters.
func (c *Client) GetChannels(ctx context.Context, filters map[string]string) ([]Channel, error) {
	endpoint := "/api/v2/experimental/channels"

	// Build query parameters from filters
	params := url.Values{}
	for key, value := range filters {
		params.Set(key, value)
	}

	queryString := params.Encode()
	if queryString != "" {
		endpoint += "?" + queryString
	}

	// PRTG API returns array directly, not wrapped in object
	var channels []Channel
	if err := c.doRequest(ctx, "GET", endpoint, nil, &channels); err != nil {
		return nil, err
	}

	return channels, nil
}

// GetChannelsBySensor retrieves all channels for a specific sensor.
func (c *Client) GetChannelsBySensor(ctx context.Context, sensorID int) ([]Channel, error) {
	filters := map[string]string{
		"filter_objid": fmt.Sprintf("%d", sensorID),
	}
	return c.GetChannels(ctx, filters)
}

// parseRawTimeSeriesData parses raw time series data from PRTG API.
// rawData: [[timestamp, val1, val2, ...], ...]
// channels: Channel info to get names (optional, will use generic names if nil)
func (c *Client) parseRawTimeSeriesData(
	objectID int,
	timeType TimeSeriesType,
	start, end *time.Time,
	rawData [][]interface{},
	channels []Channel,
) (*TimeSeriesData, error) {
	if len(rawData) == 0 {
		return &TimeSeriesData{
			ObjectID:   objectID,
			TimeType:   timeType,
			StartTime:  start,
			EndTime:    end,
			Headers:    []string{"timestamp"},
			DataPoints: []TimeSeriesDataPoint{},
		}, nil
	}

	// Build headers: timestamp + channel names
	headers := []string{"timestamp"}
	numChannels := len(rawData[0]) - 1 // First column is timestamp

	// Try to get channel names from channels info
	if channels != nil && len(channels) >= numChannels {
		for i := 0; i < numChannels; i++ {
			headers = append(headers, channels[i].Name)
		}
	} else {
		// Use generic names if we don't have channel info
		for i := 0; i < numChannels; i++ {
			headers = append(headers, fmt.Sprintf("Channel %d", i))
		}
	}

	// Parse data points
	dataPoints := make([]TimeSeriesDataPoint, 0, len(rawData))

	for _, row := range rawData {
		if len(row) == 0 {
			continue
		}

		// First column is timestamp
		timestamp, err := parseTimestamp(row[0])
		if err != nil {
			c.logger.Warn().
				Interface("value", row[0]).
				Err(err).
				Msg("Failed to parse timestamp, skipping row")
			continue
		}

		// Remaining columns are channel values
		values := make(map[string]interface{})
		for i := 1; i < len(row) && i-1 < len(headers)-1; i++ {
			channelName := headers[i] // headers[0] is "timestamp", so headers[i] matches row[i]
			values[channelName] = row[i]
		}

		dataPoints = append(dataPoints, TimeSeriesDataPoint{
			Timestamp: timestamp,
			Values:    values,
		})
	}

	return &TimeSeriesData{
		ObjectID:   objectID,
		TimeType:   timeType,
		StartTime:  start,
		EndTime:    end,
		Headers:    headers,
		DataPoints: dataPoints,
	}, nil
}

// parseTimestamp parses various timestamp formats from PRTG API.
func parseTimestamp(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		// Try RFC3339 format first
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, nil
		}
		// Try other common formats
		formats := []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", v)
	case float64:
		// Unix timestamp
		return time.Unix(int64(v), 0), nil
	case int64:
		// Unix timestamp
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("unexpected timestamp type: %T", value)
	}
}

// doRequest performs an HTTP request to the PRTG API.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader, result interface{}) error {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.Debug().
		Str("method", method).
		Str("url", fullURL).
		Msg("Sending PRTG API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAPIRequest, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Handle HTTP errors (accept any 2xx status as success)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.handleHTTPError(resp.StatusCode, endpoint, respBody)
	}

	// Parse JSON response (only if status is 200 and there's content)
	if result != nil && resp.StatusCode == http.StatusOK && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			c.logger.Error().
				Str("endpoint", endpoint).
				Str("body", string(respBody)).
				Err(err).
				Msg("Failed to parse PRTG API response")
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	c.logger.Debug().
		Str("endpoint", endpoint).
		Int("status", resp.StatusCode).
		Msg("PRTG API request successful")

	return nil
}

// handleHTTPError converts HTTP status codes to appropriate errors.
func (c *Client) handleHTTPError(statusCode int, endpoint string, body []byte) error {
	message := string(body)
	if message == "" {
		message = http.StatusText(statusCode)
	}

	c.logger.Warn().
		Int("status", statusCode).
		Str("endpoint", endpoint).
		Str("message", message).
		Msg("PRTG API request failed")

	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, message)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("%w: %s", ErrServerError, message)
	default:
		return NewAPIError(statusCode, endpoint, message)
	}
}

// Ping checks if the PRTG API is reachable and authenticated.
func (c *Client) Ping(ctx context.Context) error {
	endpoint := "/api/v2/health"

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAPIRequest, err)
	}
	defer resp.Body.Close()

	// Accept any 2xx status code as success (200, 204, etc.)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PRTG API health check failed with status %d", resp.StatusCode)
	}

	c.logger.Info().Int("status", resp.StatusCode).Msg("PRTG API connection successful")
	return nil
}
