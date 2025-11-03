// Package prtg provides a client for PRTG API v2 to access historical metrics and time series data.
package prtg

import (
	"errors"
	"fmt"
)

var (
	// ErrClientNotConfigured is returned when PRTG client is accessed but not enabled in configuration.
	ErrClientNotConfigured = errors.New("PRTG API client is not configured or enabled")

	// ErrInvalidBaseURL is returned when the base URL is empty or invalid.
	ErrInvalidBaseURL = errors.New("invalid PRTG base URL")

	// ErrInvalidToken is returned when the API token is empty.
	ErrInvalidToken = errors.New("invalid PRTG API token")

	// ErrAPIRequest is returned when an API request fails.
	ErrAPIRequest = errors.New("PRTG API request failed")

	// ErrUnauthorized is returned when authentication fails (401).
	ErrUnauthorized = errors.New("PRTG API authentication failed - check API token")

	// ErrNotFound is returned when a resource is not found (404).
	ErrNotFound = errors.New("PRTG resource not found")

	// ErrRateLimited is returned when rate limit is exceeded (429).
	ErrRateLimited = errors.New("PRTG API rate limit exceeded")

	// ErrServerError is returned when PRTG server returns 5xx error.
	ErrServerError = errors.New("PRTG server error")
)

// APIError represents an error from the PRTG API.
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("PRTG API error (status %d) on %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

// NewAPIError creates a new APIError.
func NewAPIError(statusCode int, endpoint, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Endpoint:   endpoint,
	}
}
