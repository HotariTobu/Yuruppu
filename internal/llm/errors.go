package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/genai"
)

// MapAPIError maps Vertex AI API errors to custom LLM error types.
// FR-004: On LLM API error, return appropriate custom error type
// NFR-003: Error details are preserved for logging
func MapAPIError(err error) error {
	if err == nil {
		return nil
	}

	// Check for context errors first (timeout/cancellation)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &LLMTimeoutError{
			Message: fmt.Sprintf("LLM API timeout: %v", err),
		}
	}

	// Check for Vertex AI API errors (real SDK errors)
	// Note: genai.APIError has Error() on value receiver, so we use value type here
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		return mapHTTPStatusCode(apiErr.Code, apiErr.Message)
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &LLMNetworkError{
			Message: fmt.Sprintf("LLM API network error: %v", err),
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &LLMNetworkError{
			Message: fmt.Sprintf("LLM API network error: %v", err),
		}
	}

	// Check for URL errors (network-related)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return &LLMNetworkError{
			Message: fmt.Sprintf("LLM API network error: %v", err),
		}
	}

	// Default to response error for unknown errors
	return &LLMResponseError{
		Message: fmt.Sprintf("LLM API error: %v", err),
	}
}

// mapHTTPStatusCode maps HTTP status codes to appropriate error types.
func mapHTTPStatusCode(code int, message string) error {
	switch code {
	case 401, 403:
		return &LLMAuthError{
			Message:    fmt.Sprintf("LLM API auth error: %s", message),
			StatusCode: code,
		}
	case 429:
		return &LLMRateLimitError{
			Message: fmt.Sprintf("LLM API rate limit: %s", message),
		}
	case 500, 502, 503, 504:
		return &LLMResponseError{
			Message: fmt.Sprintf("LLM API server error: %s", message),
		}
	default:
		return &LLMResponseError{
			Message: fmt.Sprintf("LLM API error (HTTP %d): %s", code, message),
		}
	}
}
