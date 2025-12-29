package agent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/genai"
)

// TimeoutError represents an API timeout error.
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	return e.Message
}

// RateLimitError represents an API rate limit error.
type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// NetworkError represents a network error during API call.
type NetworkError struct {
	Message string
}

func (e *NetworkError) Error() string {
	return e.Message
}

// ResponseError represents an invalid or malformed response.
type ResponseError struct {
	Message string
}

func (e *ResponseError) Error() string {
	return e.Message
}

// AuthError represents an authentication/authorization error.
type AuthError struct {
	Message    string
	StatusCode int // HTTP status code (401 or 403)
}

func (e *AuthError) Error() string {
	return e.Message
}

// ClosedError represents an error when using a closed agent.
type ClosedError struct {
	Message string
}

func (e *ClosedError) Error() string {
	return e.Message
}

// NotConfiguredError represents an error when Configure has not been called.
type NotConfiguredError struct {
	Message string
}

func (e *NotConfiguredError) Error() string {
	return e.Message
}

// mapAPIError maps Vertex AI API errors to custom error types.
func mapAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Check for context errors first (timeout/cancellation)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &TimeoutError{
			Message: fmt.Sprintf("API timeout: %v", err),
		}
	}

	// Check for Vertex AI API errors
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return mapHTTPStatusCode(apiErr.Code, apiErr.Message)
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &NetworkError{
			Message: fmt.Sprintf("API network error: %v", err),
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &NetworkError{
			Message: fmt.Sprintf("API network error: %v", err),
		}
	}

	// Check for URL errors (network-related)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &NetworkError{
			Message: fmt.Sprintf("API network error: %v", err),
		}
	}

	// Default to response error for unknown errors
	return &ResponseError{
		Message: fmt.Sprintf("API error: %v", err),
	}
}

// mapHTTPStatusCode maps HTTP status codes to appropriate error types.
func mapHTTPStatusCode(code int, message string) error {
	switch code {
	case 401, 403:
		return &AuthError{
			Message:    fmt.Sprintf("API auth error: %s", message),
			StatusCode: code,
		}
	case 429:
		return &RateLimitError{
			Message: fmt.Sprintf("API rate limit: %s", message),
		}
	case 500, 502, 503, 504:
		return &ResponseError{
			Message: fmt.Sprintf("API server error: %s", message),
		}
	default:
		return &ResponseError{
			Message: fmt.Sprintf("API error (HTTP %d): %s", code, message),
		}
	}
}
