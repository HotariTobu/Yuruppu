package llm_test

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// =============================================================================
// NewVertexAIClient Tests
// =============================================================================

// TestNewVertexAIClient_MissingProjectID tests that client initialization fails when projectID is missing.
// AC-013: Bot fails to start during initialization if credentials are missing
// FR-003: Load LLM API credentials from environment variables
func TestNewVertexAIClient_MissingProjectID(t *testing.T) {
	tests := []struct {
		name        string
		projectID   string
		wantErr     bool
		wantErrMsg  string
		wantErrType string
	}{
		{
			name:        "empty project ID returns error",
			projectID:   "",
			wantErr:     true,
			wantErrMsg:  "projectID is required",
			wantErrType: "config",
		},
		{
			name:        "whitespace-only project ID returns error",
			projectID:   "   ",
			wantErr:     true,
			wantErrMsg:  "projectID is required",
			wantErrType: "config",
		},
		{
			name:        "whitespace with tabs project ID returns error",
			projectID:   "\t\n  ",
			wantErr:     true,
			wantErrMsg:  "projectID is required",
			wantErrType: "config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Project ID is empty or whitespace-only
			// (projectID is set via function parameter, not env var for test isolation)

			// When: Attempt to create Vertex AI client
			client, err := llm.NewVertexAIClient(context.Background(), tt.projectID, "test-region", discardLogger())

			// Then: Should return error indicating missing projectID
			if tt.wantErr {
				require.Error(t, err,
					"should return error when projectID is missing")
				assert.Nil(t, client,
					"client should be nil when initialization fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate projectID is required")

				// Then: Error should be of appropriate type
				if tt.wantErrType == "config" {
					// Verify it's a configuration error (could use custom error type)
					assert.Contains(t, err.Error(), "projectID",
						"error should clearly indicate the missing parameter name")
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestNewVertexAIClient_EmptyRegion tests that client initialization fails when region is missing.
// AC-005: Add new test for empty region validation
// AC-007: Error is returned when region is empty
func TestNewVertexAIClient_EmptyRegion(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "empty region returns error",
			region:     "",
			wantErr:    true,
			wantErrMsg: "region is required",
		},
		{
			name:       "whitespace-only region returns error",
			region:     "   ",
			wantErr:    true,
			wantErrMsg: "region is required",
		},
		{
			name:       "whitespace with tabs region returns error",
			region:     "\t\n  ",
			wantErr:    true,
			wantErrMsg: "region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Valid project ID but empty/whitespace region
			// When: Attempt to create Vertex AI client
			client, err := llm.NewVertexAIClient(context.Background(), "test-project-id", tt.region, discardLogger())

			// Then: Should return error indicating missing region
			if tt.wantErr {
				require.Error(t, err,
					"should return error when region is missing")
				assert.Nil(t, client,
					"client should be nil when initialization fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate region is required")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// =============================================================================
// Error Mapping Tests
// =============================================================================

// TestMapHTTPStatusCode tests mapping of HTTP status codes to custom error types.
// FR-004: On LLM API error, return appropriate custom error type
func TestMapHTTPStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		httpCode       int
		message        string
		wantType       string
		wantContains   string
		wantStatusCode int
	}{
		{
			name:           "HTTP 401 maps to LLMAuthError",
			httpCode:       401,
			message:        "Unauthorized",
			wantType:       "*llm.LLMAuthError",
			wantContains:   "auth",
			wantStatusCode: 401,
		},
		{
			name:           "HTTP 403 maps to LLMAuthError",
			httpCode:       403,
			message:        "Forbidden",
			wantType:       "*llm.LLMAuthError",
			wantContains:   "auth",
			wantStatusCode: 403,
		},
		{
			name:         "HTTP 429 maps to LLMRateLimitError",
			httpCode:     429,
			message:      "Too Many Requests",
			wantType:     "*llm.LLMRateLimitError",
			wantContains: "rate limit",
		},
		{
			name:         "HTTP 500 maps to LLMResponseError",
			httpCode:     500,
			message:      "Internal Server Error",
			wantType:     "*llm.LLMResponseError",
			wantContains: "server",
		},
		{
			name:         "HTTP 503 maps to LLMResponseError",
			httpCode:     503,
			message:      "Service Unavailable",
			wantType:     "*llm.LLMResponseError",
			wantContains: "server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Map HTTP status code to error
			mappedErr := llm.MapHTTPStatusCode(tt.httpCode, tt.message)

			// Then: Should return correct error type
			require.NotNil(t, mappedErr)
			actualType := fmt.Sprintf("%T", mappedErr)
			assert.Equal(t, tt.wantType, actualType,
				"error should be mapped to %s, got %s", tt.wantType, actualType)

			// Then: Error message should contain expected text
			assert.Contains(t, strings.ToLower(mappedErr.Error()), tt.wantContains,
				"error message should contain '%s'", tt.wantContains)

			// Then: Verify type-specific fields
			if e, ok := mappedErr.(*llm.LLMAuthError); ok {
				if tt.wantStatusCode > 0 {
					assert.Equal(t, tt.wantStatusCode, e.StatusCode,
						"auth error should have status code %d", tt.wantStatusCode)
				}
			}
		})
	}
}

// TestMapHTTPStatusCode_PreservesOriginalErrorDetails tests that original error details are preserved.
// NFR-003: Log LLM API errors at ERROR level with error type and details
func TestMapHTTPStatusCode_PreservesOriginalErrorDetails(t *testing.T) {
	tests := []struct {
		name     string
		httpCode int
		message  string
		wantMsg  string
	}{
		{
			name:     "API error message is preserved",
			httpCode: 401,
			message:  "Invalid API key: abc123",
			wantMsg:  "Invalid API key: abc123",
		},
		{
			name:     "rate limit details are preserved",
			httpCode: 429,
			message:  "Quota exceeded for project",
			wantMsg:  "Quota exceeded for project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Map HTTP status code to error
			mappedErr := llm.MapHTTPStatusCode(tt.httpCode, tt.message)

			// Then: Original error details should be preserved in the message
			require.NotNil(t, mappedErr)
			assert.Contains(t, mappedErr.Error(), tt.wantMsg,
				"mapped error should preserve original message")
		})
	}
}

// TestMapAPIError tests mapping of Vertex AI API errors to custom error types.
// FR-004: On LLM API error, return appropriate custom error type
// NFR-003: Error details should be preserved for logging
func TestMapAPIError(t *testing.T) {
	tests := []struct {
		name         string
		apiError     error
		wantType     string
		wantContains string
	}{
		{
			name:         "context.DeadlineExceeded maps to LLMTimeoutError",
			apiError:     context.DeadlineExceeded,
			wantType:     "*llm.LLMTimeoutError",
			wantContains: "timeout",
		},
		{
			name:         "context.Canceled maps to LLMTimeoutError",
			apiError:     context.Canceled,
			wantType:     "*llm.LLMTimeoutError",
			wantContains: "timeout",
		},
		{
			name:         "net.OpError maps to LLMNetworkError",
			apiError:     &mockNetError{Msg: "connection refused"},
			wantType:     "*llm.LLMNetworkError",
			wantContains: "network",
		},
		{
			name:         "DNS error maps to LLMNetworkError",
			apiError:     &mockDNSError{Msg: "lookup failed"},
			wantType:     "*llm.LLMNetworkError",
			wantContains: "network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Map the API error
			mappedErr := llm.MapAPIError(tt.apiError)

			// Then: Should return correct error type
			require.NotNil(t, mappedErr)
			actualType := fmt.Sprintf("%T", mappedErr)
			assert.Equal(t, tt.wantType, actualType,
				"error should be mapped to %s, got %s", tt.wantType, actualType)

			// Then: Error message should contain expected text
			assert.Contains(t, strings.ToLower(mappedErr.Error()), tt.wantContains,
				"error message should contain '%s'", tt.wantContains)
		})
	}
}

// TestMapAPIError_PreservesOriginalErrorDetails tests that original error details are preserved.
// NFR-003: Log LLM API errors at ERROR level with error type and details
func TestMapAPIError_PreservesOriginalErrorDetails(t *testing.T) {
	tests := []struct {
		name     string
		apiError error
		wantMsg  string
	}{
		{
			name:     "context error message is preserved",
			apiError: context.DeadlineExceeded,
			wantMsg:  "context deadline exceeded",
		},
		{
			name:     "network error details are preserved",
			apiError: &mockNetError{Msg: "dial tcp: connection refused"},
			wantMsg:  "dial tcp: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Map the API error
			mappedErr := llm.MapAPIError(tt.apiError)

			// Then: Original error details should be preserved in the message
			require.NotNil(t, mappedErr)
			assert.Contains(t, mappedErr.Error(), tt.wantMsg,
				"mapped error should preserve original message")
		})
	}
}

// =============================================================================
// Test Helpers
// =============================================================================

// mockNetError is a mock network error for testing.
type mockNetError struct {
	Msg string
}

func (e *mockNetError) Error() string {
	return e.Msg
}

// Temporary implements net.Error interface
func (e *mockNetError) Temporary() bool {
	return true
}

// Timeout implements net.Error interface
func (e *mockNetError) Timeout() bool {
	return false
}

// mockDNSError is a mock DNS error for testing.
type mockDNSError struct {
	Msg string
}

func (e *mockDNSError) Error() string {
	return e.Msg
}

// Temporary implements net.Error interface
func (e *mockDNSError) Temporary() bool {
	return true
}

// Timeout implements net.Error interface
func (e *mockDNSError) Timeout() bool {
	return false
}
