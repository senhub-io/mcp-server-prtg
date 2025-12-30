package prtg

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GetSensors retrieves sensors with optional filters.
func (c *Client) GetSensors(ctx context.Context, filters map[string]interface{}) ([]Sensor, error) {
	endpoint := "/api/v2/experimental/sensors"

	// Build query parameters
	params := buildQueryParams(filters)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var sensors []Sensor
	if err := c.doRequest(ctx, "GET", endpoint, nil, &sensors); err != nil {
		return nil, err
	}

	return sensors, nil
}

// GetSensorByID retrieves a single sensor by ID.
func (c *Client) GetSensorByID(ctx context.Context, sensorID int) (*Sensor, error) {
	endpoint := fmt.Sprintf("/api/v2/sensors/%d", sensorID)

	var sensor Sensor
	if err := c.doRequest(ctx, "GET", endpoint, nil, &sensor); err != nil {
		return nil, err
	}

	return &sensor, nil
}

// GetDeviceByID retrieves a single device by ID with optional includes.
func (c *Client) GetDeviceByID(ctx context.Context, deviceID int, includes ...string) (*Device, error) {
	endpoint := fmt.Sprintf("/api/v2/devices/%d", deviceID)

	if len(includes) > 0 {
		params := url.Values{}
		params.Set("include", strings.Join(includes, ","))
		endpoint += "?" + params.Encode()
	}

	var device Device
	if err := c.doRequest(ctx, "GET", endpoint, nil, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

// GetGroups retrieves groups with optional filters.
func (c *Client) GetGroups(ctx context.Context, filters map[string]interface{}) ([]Group, error) {
	endpoint := "/api/v2/experimental/groups"

	// Build query parameters
	params := buildQueryParams(filters)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var groups []Group
	if err := c.doRequest(ctx, "GET", endpoint, nil, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// GetObjects retrieves generic PRTG objects with filters.
// Can return sensors, devices, groups, or probes depending on filters.
func (c *Client) GetObjects(ctx context.Context, filters map[string]interface{}) ([]PRTGObject, error) {
	endpoint := "/api/v2/experimental/objects"

	// Build query parameters
	params := buildQueryParams(filters)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	var objects []PRTGObject
	if err := c.doRequest(ctx, "GET", endpoint, nil, &objects); err != nil {
		return nil, err
	}

	return objects, nil
}

// GetObjectCounts retrieves counts of objects by type.
func (c *Client) GetObjectCounts(ctx context.Context) (*SensorCount, error) {
	endpoint := "/api/v2/objects/count"

	var counts SensorCount
	if err := c.doRequest(ctx, "GET", endpoint, nil, &counts); err != nil {
		return nil, err
	}

	return &counts, nil
}

// GetStatusSummary retrieves global status summary.
func (c *Client) GetStatusSummary(ctx context.Context) (*StatusSummary, error) {
	endpoint := "/api/v2/sensor-status-summary"

	var summary StatusSummary
	if err := c.doRequest(ctx, "GET", endpoint, nil, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

// buildQueryParams converts a map of filters to URL query parameters.
// Supports:
// - filter_<field>: field=value (e.g., filter_status=3)
// - filter: complex filter expression (e.g., status=3&name~test)
// - limit: maximum results
// - offset: pagination offset
// - sort: sort field
func buildQueryParams(filters map[string]interface{}) url.Values {
	params := url.Values{}

	for key, value := range filters {
		if value == nil {
			continue
		}

		// Handle special parameters
		switch key {
		case "limit":
			if v, ok := toInt(value); ok && v > 0 {
				params.Set("limit", strconv.Itoa(v))
			}
		case "offset":
			if v, ok := toInt(value); ok && v >= 0 {
				params.Set("offset", strconv.Itoa(v))
			}
		case "sort":
			if v, ok := value.(string); ok && v != "" {
				params.Set("sort", v)
			}
		default:
			// Handle filter parameters
			if strings.HasPrefix(key, "filter_") {
				// filter_status=3 -> filter=status=3
				field := strings.TrimPrefix(key, "filter_")
				filterValue := fmt.Sprintf("%s=%v", field, value)
				params.Add("filter", filterValue)
			} else if key == "filter" {
				// Direct filter expression
				if v, ok := value.(string); ok && v != "" {
					params.Add("filter", v)
				}
			}
		}
	}

	return params
}

// toInt attempts to convert an interface{} to int.
func toInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	}
	return 0, false
}
