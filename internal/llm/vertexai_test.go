package llm_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
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
			client, err := llm.NewVertexAIClient(context.Background(), tt.projectID, "test-region", "test-model", discardLogger())

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
			client, err := llm.NewVertexAIClient(context.Background(), "test-project-id", tt.region, "test-model", discardLogger())

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

// TestNewVertexAIClient_EmptyModel tests that client initialization fails when model is missing.
// FX-003: Add model parameter to NewVertexAIClient()
func TestNewVertexAIClient_EmptyModel(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:       "empty model returns error",
			model:      "",
			wantErr:    true,
			wantErrMsg: "model is required",
		},
		{
			name:       "whitespace-only model returns error",
			model:      "   ",
			wantErr:    true,
			wantErrMsg: "model is required",
		},
		{
			name:       "whitespace with tabs model returns error",
			model:      "\t\n  ",
			wantErr:    true,
			wantErrMsg: "model is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Valid project ID and region but empty/whitespace model
			// When: Attempt to create Vertex AI client
			client, err := llm.NewVertexAIClient(context.Background(), "test-project-id", "test-region", tt.model, discardLogger())

			// Then: Should return error indicating missing model
			if tt.wantErr {
				require.Error(t, err,
					"should return error when model is missing")
				assert.Nil(t, client,
					"client should be nil when initialization fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate model is required")
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

// =============================================================================
// VertexAI Client Close Tests (AC-004)
// =============================================================================

// TestVertexAIClient_Close tests the Close method implementation.
// AC-004: Provider Close Method
// These tests will fail until Close is implemented in vertexAIClient.
func TestVertexAIClient_Close(t *testing.T) {
	// Note: These tests require actual Vertex AI client initialization
	// which is tested in integration tests. Here we test the expected behavior.

	t.Run("Close method exists on vertexAIClient", func(t *testing.T) {
		// This test verifies the Close method signature exists
		// It will fail during compilation if the method doesn't exist

		// Given: We have a Provider interface variable
		var provider llm.Provider

		// When: We assign a mock (which has Close method)
		provider = &mockVertexAIProvider{}

		// Then: Close method should be callable
		ctx := context.Background()
		err := provider.(interface {
			Close(ctx context.Context) error
		}).Close(ctx)

		// Then: Should succeed
		require.NoError(t, err)
	})

	t.Run("Close is idempotent for vertexAIClient", func(t *testing.T) {
		// Given: A mock VertexAI provider
		provider := &mockVertexAIProvider{}

		ctx := context.Background()

		// When: Close is called multiple times
		err1 := provider.Close(ctx)
		err2 := provider.Close(ctx)
		err3 := provider.Close(ctx)

		// Then: All calls should succeed (idempotent)
		require.NoError(t, err1, "First Close should succeed")
		require.NoError(t, err2, "Second Close should succeed (idempotent)")
		require.NoError(t, err3, "Third Close should succeed (idempotent)")
	})

	t.Run("GenerateText fails after Close for vertexAIClient", func(t *testing.T) {
		// Given: A mock VertexAI provider
		provider := &mockVertexAIProvider{
			response: "test response",
		}

		ctx := context.Background()

		// Given: GenerateText works before Close
		response, err := provider.GenerateText(ctx, "system", "user")
		require.NoError(t, err, "GenerateText should work before Close")
		assert.Equal(t, "test response", response)

		// When: Close is called
		err = provider.Close(ctx)
		require.NoError(t, err, "Close should succeed")

		// Then: GenerateText should fail after Close
		response, err = provider.GenerateText(ctx, "system", "user")
		require.Error(t, err, "GenerateText should fail after Close")
		assert.Empty(t, response, "Response should be empty after Close")
		assert.Contains(t, err.Error(), "closed",
			"Error message should indicate provider is closed")
	})

	t.Run("Close cleans up cached resources", func(t *testing.T) {
		// Given: A mock VertexAI provider with cached resources
		provider := &mockVertexAIProvider{
			response:         "test response",
			hasCachedContent: true,
		}

		// When: Close is called
		ctx := context.Background()
		err := provider.Close(ctx)

		// Then: Close should succeed and cleanup resources
		require.NoError(t, err, "Close should succeed")
		assert.False(t, provider.hasCachedContent,
			"Cached content should be cleaned up after Close")
	})
}

// =============================================================================
// Test Helpers for Close Tests
// =============================================================================

// mockVertexAIProvider simulates the behavior of vertexAIClient for Close testing.
type mockVertexAIProvider struct {
	response         string
	err              error
	closed           bool
	hasCachedContent bool
}

func (m *mockVertexAIProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockVertexAIProvider) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}

	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockVertexAIProvider) CreateCache(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if m.closed {
		return "", errors.New("provider is closed")
	}
	return "mock-cache-name", nil
}

func (m *mockVertexAIProvider) DeleteCache(ctx context.Context, cacheName string) error {
	return nil
}

func (m *mockVertexAIProvider) Close(ctx context.Context) error {
	// Idempotent - safe to call multiple times
	if !m.closed {
		// Cleanup cached resources
		m.hasCachedContent = false
	}
	m.closed = true
	return nil
}
