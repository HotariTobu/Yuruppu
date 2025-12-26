package llm_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// TestNewVertexAIClient_MissingProjectID tests that client initialization fails when GCP_PROJECT_ID is missing.
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
			wantErrMsg:  "GCP_PROJECT_ID",
			wantErrType: "config",
		},
		{
			name:        "whitespace-only project ID returns error",
			projectID:   "   ",
			wantErr:     true,
			wantErrMsg:  "GCP_PROJECT_ID",
			wantErrType: "config",
		},
		{
			name:        "whitespace with tabs project ID returns error",
			projectID:   "\t\n  ",
			wantErr:     true,
			wantErrMsg:  "GCP_PROJECT_ID",
			wantErrType: "config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Project ID is empty or whitespace-only
			// (projectID is set via function parameter, not env var for test isolation)

			// When: Attempt to create Vertex AI client
			// SC-003: Pass fallback region as parameter
			client, err := llm.NewVertexAIClient(context.Background(), tt.projectID, "us-central1", discardLogger())

			// Then: Should return error indicating missing GCP_PROJECT_ID
			if tt.wantErr {
				require.Error(t, err,
					"should return error when GCP_PROJECT_ID is missing")
				assert.Nil(t, client,
					"client should be nil when initialization fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate which variable is missing")

				// Then: Error should be of appropriate type
				if tt.wantErrType == "config" {
					// Verify it's a configuration error (could use custom error type)
					assert.Contains(t, err.Error(), "GCP_PROJECT_ID",
						"error should clearly indicate the missing variable name")
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestNewVertexAIClient_ErrorMessage tests error message clarity.
// AC-013: Error message indicates which variable is missing
// FR-003: Bot fails to start during initialization if credentials are missing
func TestNewVertexAIClient_ErrorMessage(t *testing.T) {
	tests := []struct {
		name            string
		projectID       string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:      "error message contains variable name",
			projectID: "",
			wantContains: []string{
				"GCP_PROJECT_ID",
			},
			wantNotContains: []string{},
		},
		{
			name:      "error message is clear and actionable",
			projectID: "",
			wantContains: []string{
				"GCP_PROJECT_ID",
				"missing",
			},
			wantNotContains: []string{
				"unknown error",
				"unexpected",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Invalid project ID
			// When: Attempt to create client
			// SC-003: Pass fallback region as parameter
			_, err := llm.NewVertexAIClient(context.Background(), tt.projectID, "us-central1", discardLogger())

			// Then: Error message should be clear
			require.Error(t, err)
			errMsg := err.Error()

			for _, want := range tt.wantContains {
				assert.Contains(t, errMsg, want,
					"error message should contain '%s'", want)
			}

			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, errMsg, notWant,
					"error message should not contain '%s'", notWant)
			}
		})
	}
}

// TestNewVertexAIClient_FromEnvironment tests loading credentials from environment variables.
// FR-003: Load LLM API credentials from environment variables
func TestNewVertexAIClient_FromEnvironment(t *testing.T) {
	t.Run("returns error when environment variable is not set", func(t *testing.T) {
		// Given: GCP_PROJECT_ID is not set in environment
		originalEnv := os.Getenv("GCP_PROJECT_ID")
		os.Unsetenv("GCP_PROJECT_ID")
		t.Cleanup(func() {
			if originalEnv != "" {
				os.Setenv("GCP_PROJECT_ID", originalEnv)
			}
		})

		projectID := os.Getenv("GCP_PROJECT_ID") // Will be empty string

		// When: Attempt to create client
		// SC-003: Pass fallback region as parameter
		client, err := llm.NewVertexAIClient(context.Background(), projectID, "us-central1", discardLogger())

		// Then: Should return error
		require.Error(t, err,
			"should return error when GCP_PROJECT_ID is not set")
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "GCP_PROJECT_ID",
			"error should mention the missing environment variable")
	})
}

// TestGetRegion tests region determination logic for Cloud Run metadata.
// AC-001: Region derived from Cloud Run metadata
// SC-004: GetRegion accepts fallback region as parameter instead of reading from environment
func TestGetRegion(t *testing.T) {
	tests := []struct {
		name           string
		metadataServer *metadataServerMock
		fallbackRegion string
		want           string
	}{
		{
			name: "metadata server returns valid response - extract region correctly",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789/regions/us-west1",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "us-central1",
			want:           "us-west1",
		},
		{
			name: "metadata server returns different region - extract correctly",
			metadataServer: &metadataServerMock{
				response:   "projects/987654321/regions/asia-northeast1",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "us-central1",
			want:           "asia-northeast1",
		},
		{
			name: "metadata server timeout (2s) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789/regions/us-west1",
				statusCode: 200,
				delay:      3000, // 3 seconds - exceeds 2s timeout
			},
			fallbackRegion: "us-east1",
			want:           "us-east1",
		},
		{
			name: "metadata server unavailable - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 500,
				delay:      0,
			},
			fallbackRegion: "europe-west1",
			want:           "europe-west1",
		},
		{
			name: "metadata server returns 404 - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 404,
				delay:      0,
			},
			fallbackRegion: "us-central1",
			want:           "us-central1",
		},
		{
			name: "malformed response (not projects/*/regions/*) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "invalid-format",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "us-west2",
			want:           "us-west2",
		},
		{
			name: "malformed response (missing region part) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "asia-south1",
			want:           "asia-south1",
		},
		{
			name: "malformed response (empty string) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "us-east4",
			want:           "us-east4",
		},
		{
			name: "metadata unavailable - use provided fallback region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 503,
				delay:      0,
			},
			fallbackRegion: "us-south1",
			want:           "us-south1",
		},
		{
			name: "metadata timeout - use provided fallback region",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789/regions/us-west1",
				statusCode: 200,
				delay:      3000, // 3 seconds - exceeds 2s timeout
			},
			fallbackRegion: "us-central1",
			want:           "us-central1",
		},
		{
			name: "malformed response - use provided fallback region",
			metadataServer: &metadataServerMock{
				response:   "malformed",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "us-central1",
			want:           "us-central1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server and fallback region
			server := tt.metadataServer.Start(t)
			defer server.Close()

			// When: Get region with fallback
			// SC-004: Pass fallback region as parameter
			got := llm.GetRegion(server.URL, tt.fallbackRegion)

			// Then: Should return expected region
			assert.Equal(t, tt.want, got,
				"region should match expected value")
		})
	}
}

// TestGetRegion_MetadataHeaders tests that metadata server requests include required headers.
// AC-001: Metadata request requires header: Metadata-Flavor: Google
func TestGetRegion_MetadataHeaders(t *testing.T) {
	t.Run("metadata request includes Metadata-Flavor: Google header", func(t *testing.T) {
		// Given: Mock metadata server that captures request headers
		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(200)
			w.Write([]byte("projects/123456789/regions/us-west1"))
		}))
		defer server.Close()

		// When: Get region
		// SC-004: Pass fallback region as parameter
		_ = llm.GetRegion(server.URL, "us-central1")

		// Then: Should include Metadata-Flavor header
		require.NotNil(t, capturedHeaders, "request should have been made")
		assert.Equal(t, "Google", capturedHeaders.Get("Metadata-Flavor"),
			"should include Metadata-Flavor: Google header")
	})

	t.Run("metadata request without proper header should be rejected by real server", func(t *testing.T) {
		// Given: Mock metadata server that requires Metadata-Flavor header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Metadata-Flavor") != "Google" {
				w.WriteHeader(403)
				w.Write([]byte("Forbidden"))
				return
			}
			w.WriteHeader(200)
			w.Write([]byte("projects/123456789/regions/us-west1"))
		}))
		defer server.Close()

		// When: Get region (implementation should include header)
		// SC-004: Pass fallback region as parameter
		got := llm.GetRegion(server.URL, "fallback-region")

		// Then: Should succeed (implementation includes header)
		// If implementation doesn't include header, it will fallback to provided region
		assert.NotEmpty(t, got, "should return a region")
	})
}

// TestGetRegion_Timeout tests that metadata server request has 2-second timeout.
// AC-001: Metadata server request has a 2-second timeout
// SC-004: GetRegion now accepts fallback region as parameter
func TestGetRegion_Timeout(t *testing.T) {
	tests := []struct {
		name           string
		serverDelay    int // milliseconds
		fallbackRegion string
		want           string
	}{
		{
			name:           "request completes within 1 second - use metadata",
			serverDelay:    1000,
			fallbackRegion: "fallback-region",
			want:           "us-west1", // from metadata
		},
		{
			name:           "request completes within 1.9 seconds - use metadata",
			serverDelay:    1900,
			fallbackRegion: "fallback-region",
			want:           "us-west1", // from metadata
		},
		{
			name:           "request takes exactly 2 seconds - may timeout",
			serverDelay:    2000,
			fallbackRegion: "fallback-region",
			want:           "fallback-region", // depends on implementation, may timeout
		},
		{
			name:           "request takes 2.1 seconds - fallback to provided region",
			serverDelay:    2100,
			fallbackRegion: "fallback-region",
			want:           "fallback-region",
		},
		{
			name:           "request takes 5 seconds - fallback to provided region",
			serverDelay:    5000,
			fallbackRegion: "us-east1",
			want:           "us-east1",
		},
		{
			name:           "timeout with us-central1 fallback",
			serverDelay:    3000,
			fallbackRegion: "us-central1",
			want:           "us-central1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server with delay
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(tt.serverDelay) * time.Millisecond)
				w.WriteHeader(200)
				w.Write([]byte("projects/123456789/regions/us-west1"))
			}))
			defer server.Close()

			// When: Get region with fallback
			// SC-004: Pass fallback region as parameter
			got := llm.GetRegion(server.URL, tt.fallbackRegion)

			// Then: Should handle timeout correctly
			assert.Equal(t, tt.want, got,
				"should fallback to provided region when timeout occurs")
		})
	}
}

// TestGetRegion_EdgeCases tests edge cases in region extraction.
// AC-001: Region format validation and edge cases
// SC-004: GetRegion now accepts fallback region as parameter
func TestGetRegion_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		metadataResponse string
		fallbackRegion   string
		want             string
	}{
		{
			name:             "response with extra slashes",
			metadataResponse: "projects/123456789/regions/us-west1/",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region", // malformed, fallback
		},
		{
			name:             "response with leading spaces",
			metadataResponse: "  projects/123456789/regions/us-west1",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region", // malformed, fallback
		},
		{
			name:             "response with trailing spaces",
			metadataResponse: "projects/123456789/regions/us-west1  ",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region", // malformed, fallback
		},
		{
			name:             "response with newline",
			metadataResponse: "projects/123456789/regions/us-west1\n",
			fallbackRegion:   "fallback-region",
			want:             "us-west1", // trimmed newline is acceptable
		},
		{
			name:             "response with only project",
			metadataResponse: "projects/123456789",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region",
		},
		{
			name:             "response with wrong format (regions first)",
			metadataResponse: "regions/us-west1/projects/123456789",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region",
		},
		{
			name:             "response with region but no project number",
			metadataResponse: "projects//regions/us-west1",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region", // malformed, fallback
		},
		{
			name:             "empty region name",
			metadataResponse: "projects/123456789/regions/",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region",
		},
		{
			name:             "only slashes",
			metadataResponse: "///",
			fallbackRegion:   "fallback-region",
			want:             "fallback-region",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server returning edge case response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte(tt.metadataResponse))
			}))
			defer server.Close()

			// When: Get region with fallback
			// SC-004: Pass fallback region as parameter
			got := llm.GetRegion(server.URL, tt.fallbackRegion)

			// Then: Should handle edge case correctly
			assert.Equal(t, tt.want, got,
				"should handle edge case correctly")
		})
	}
}

// TestGetRegion_ProductionEndpoint tests using actual Cloud Run metadata endpoint path.
// AC-001: Production endpoint format
func TestGetRegion_ProductionEndpoint(t *testing.T) {
	t.Run("uses correct metadata endpoint path", func(t *testing.T) {
		// Given: Mock server that verifies the request path
		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(200)
			w.Write([]byte("projects/123456789/regions/us-west1"))
		}))
		defer server.Close()

		// When: Get region with base URL (production would use http://metadata.google.internal)
		// SC-004: Pass fallback region as parameter
		_ = llm.GetRegion(server.URL, "us-central1")

		// Then: Should request the correct path
		expectedPath := "/computeMetadata/v1/instance/region"
		assert.Equal(t, expectedPath, requestedPath,
			"should request metadata endpoint path: %s", expectedPath)
	})
}

// metadataServerMock is a helper struct for mocking Cloud Run metadata server.
type metadataServerMock struct {
	response   string
	statusCode int
	delay      int // milliseconds
}

// Start creates and starts an httptest server with the configured mock behavior.
func (m *metadataServerMock) Start(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate delay if configured
		if m.delay > 0 {
			time.Sleep(time.Duration(m.delay) * time.Millisecond)
		}

		// Return configured status code and response
		w.WriteHeader(m.statusCode)
		if m.response != "" {
			w.Write([]byte(m.response))
		}
	}))
}

// TestMapAPIError tests mapping of Vertex AI API errors to custom error types.
// FR-004: On LLM API error, return appropriate custom error type
// NFR-003: Error details should be preserved for logging
func TestMapAPIError(t *testing.T) {
	tests := []struct {
		name           string
		apiError       error
		wantType       string
		wantContains   string
		wantStatusCode int
	}{
		{
			name: "HTTP 401 maps to LLMAuthError",
			apiError: &llm.MockAPIError{
				HTTPCode: 401,
				Msg:      "Unauthorized",
			},
			wantType:       "*llm.LLMAuthError",
			wantContains:   "auth",
			wantStatusCode: 401,
		},
		{
			name: "HTTP 403 maps to LLMAuthError",
			apiError: &llm.MockAPIError{
				HTTPCode: 403,
				Msg:      "Forbidden",
			},
			wantType:       "*llm.LLMAuthError",
			wantContains:   "auth",
			wantStatusCode: 403,
		},
		{
			name: "HTTP 429 maps to LLMRateLimitError",
			apiError: &llm.MockAPIError{
				HTTPCode: 429,
				Msg:      "Too Many Requests",
			},
			wantType:     "*llm.LLMRateLimitError",
			wantContains: "rate limit",
		},
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
			name: "HTTP 500 maps to LLMResponseError",
			apiError: &llm.MockAPIError{
				HTTPCode: 500,
				Msg:      "Internal Server Error",
			},
			wantType:     "*llm.LLMResponseError",
			wantContains: "server",
		},
		{
			name: "HTTP 503 maps to LLMResponseError",
			apiError: &llm.MockAPIError{
				HTTPCode: 503,
				Msg:      "Service Unavailable",
			},
			wantType:     "*llm.LLMResponseError",
			wantContains: "server",
		},
		{
			name:         "net.OpError maps to LLMNetworkError",
			apiError:     &llm.MockNetError{Msg: "connection refused"},
			wantType:     "*llm.LLMNetworkError",
			wantContains: "network",
		},
		{
			name:         "DNS error maps to LLMNetworkError",
			apiError:     &llm.MockDNSError{Msg: "lookup failed"},
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

			// Then: Verify type-specific fields
			switch e := mappedErr.(type) {
			case *llm.LLMAuthError:
				if tt.wantStatusCode > 0 {
					assert.Equal(t, tt.wantStatusCode, e.StatusCode,
						"auth error should have status code %d", tt.wantStatusCode)
				}
			}
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
			name: "API error message is preserved",
			apiError: &llm.MockAPIError{
				HTTPCode: 401,
				Msg:      "Invalid API key: abc123",
			},
			wantMsg: "Invalid API key: abc123",
		},
		{
			name: "rate limit details are preserved",
			apiError: &llm.MockAPIError{
				HTTPCode: 429,
				Msg:      "Quota exceeded for project",
			},
			wantMsg: "Quota exceeded for project",
		},
		{
			name:     "context error message is preserved",
			apiError: context.DeadlineExceeded,
			wantMsg:  "context deadline exceeded",
		},
		{
			name:     "network error details are preserved",
			apiError: &llm.MockNetError{Msg: "dial tcp: connection refused"},
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

// TestGetProjectID tests project ID determination logic for Cloud Run metadata.
// FX-001: Add GetProjectID function following the GetRegion() pattern
// AC-001: Auto-detect project ID on Cloud Run
// AC-002: Fallback to env var when metadata unavailable
// AC-003: Error when no project ID available
// AC-004: Regression - existing functionality preserved
func TestGetProjectID(t *testing.T) {
	tests := []struct {
		name              string
		metadataServer    *metadataServerMock
		fallbackProjectID string
		want              string
	}{
		{
			name: "metadata server returns valid project ID",
			metadataServer: &metadataServerMock{
				response:   "my-project-123",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "fallback-project",
			want:              "my-project-123",
		},
		{
			name: "metadata server returns different project ID",
			metadataServer: &metadataServerMock{
				response:   "another-gcp-project",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "fallback-project",
			want:              "another-gcp-project",
		},
		{
			name: "metadata server timeout (2s) - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "my-project-123",
				statusCode: 200,
				delay:      3000, // 3 seconds - exceeds 2s timeout
			},
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name: "metadata server unavailable - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 500,
				delay:      0,
			},
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name: "metadata server returns 404 - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 404,
				delay:      0,
			},
			fallbackProjectID: "my-local-project",
			want:              "my-local-project",
		},
		{
			name: "empty response - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name: "metadata unavailable - use provided fallback project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 503,
				delay:      0,
			},
			fallbackProjectID: "my-dev-project",
			want:              "my-dev-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server and fallback project ID
			server := tt.metadataServer.Start(t)
			defer server.Close()

			// When: Get project ID with fallback
			got := llm.GetProjectID(server.URL, tt.fallbackProjectID)

			// Then: Should return expected project ID
			assert.Equal(t, tt.want, got,
				"project ID should match expected value")
		})
	}
}

// TestGetProjectID_MetadataHeaders tests that metadata server requests include required headers.
// AC-001: Metadata request requires header: Metadata-Flavor: Google
func TestGetProjectID_MetadataHeaders(t *testing.T) {
	t.Run("metadata request includes Metadata-Flavor: Google header", func(t *testing.T) {
		// Given: Mock metadata server that captures request headers
		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(200)
			w.Write([]byte("my-project-123"))
		}))
		defer server.Close()

		// When: Get project ID
		_ = llm.GetProjectID(server.URL, "fallback-project")

		// Then: Should include Metadata-Flavor header
		require.NotNil(t, capturedHeaders, "request should have been made")
		assert.Equal(t, "Google", capturedHeaders.Get("Metadata-Flavor"),
			"should include Metadata-Flavor: Google header")
	})
}

// TestGetProjectID_ProductionEndpoint tests using actual Cloud Run metadata endpoint path.
// AC-001: Production endpoint format for project ID
func TestGetProjectID_ProductionEndpoint(t *testing.T) {
	t.Run("uses correct metadata endpoint path for project ID", func(t *testing.T) {
		// Given: Mock server that verifies the request path
		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(200)
			w.Write([]byte("my-project-123"))
		}))
		defer server.Close()

		// When: Get project ID with base URL (production would use http://metadata.google.internal)
		_ = llm.GetProjectID(server.URL, "fallback-project")

		// Then: Should request the correct path
		expectedPath := "/computeMetadata/v1/project/project-id"
		assert.Equal(t, expectedPath, requestedPath,
			"should request metadata endpoint path: %s", expectedPath)
	})
}

// TestGetProjectID_EdgeCases tests edge cases in project ID extraction.
// AC-001: Project ID format validation and edge cases
func TestGetProjectID_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		metadataResponse  string
		fallbackProjectID string
		want              string
	}{
		{
			name:              "response with trailing newline - trim and use",
			metadataResponse:  "my-project-123\n",
			fallbackProjectID: "fallback-project",
			want:              "my-project-123",
		},
		{
			name:              "response with CRLF - trim and use",
			metadataResponse:  "my-project-123\r\n",
			fallbackProjectID: "fallback-project",
			want:              "my-project-123",
		},
		{
			name:              "response with leading spaces - fallback (malformed)",
			metadataResponse:  "  my-project-123",
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name:              "response with trailing spaces - fallback (malformed)",
			metadataResponse:  "my-project-123  ",
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name:              "whitespace only - fallback",
			metadataResponse:  "   ",
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name:              "empty string - fallback",
			metadataResponse:  "",
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name:              "newline only - fallback",
			metadataResponse:  "\n",
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server returning edge case response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte(tt.metadataResponse))
			}))
			defer server.Close()

			// When: Get project ID with fallback
			got := llm.GetProjectID(server.URL, tt.fallbackProjectID)

			// Then: Should handle edge case correctly
			assert.Equal(t, tt.want, got,
				"should handle edge case correctly")
		})
	}
}

// TestGetProjectID_Timeout tests that metadata server request has 2-second timeout.
// AC-001: Metadata server request has a 2-second timeout
func TestGetProjectID_Timeout(t *testing.T) {
	tests := []struct {
		name              string
		serverDelay       int // milliseconds
		fallbackProjectID string
		want              string
	}{
		{
			name:              "request completes within 1 second - use metadata",
			serverDelay:       1000,
			fallbackProjectID: "fallback-project",
			want:              "my-project-123", // from metadata
		},
		{
			name:              "request takes 2.1 seconds - fallback to provided project ID",
			serverDelay:       2100,
			fallbackProjectID: "fallback-project",
			want:              "fallback-project",
		},
		{
			name:              "request takes 5 seconds - fallback to provided project ID",
			serverDelay:       5000,
			fallbackProjectID: "my-dev-project",
			want:              "my-dev-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server with delay
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(tt.serverDelay) * time.Millisecond)
				w.WriteHeader(200)
				w.Write([]byte("my-project-123"))
			}))
			defer server.Close()

			// When: Get project ID with fallback
			got := llm.GetProjectID(server.URL, tt.fallbackProjectID)

			// Then: Should handle timeout correctly
			assert.Equal(t, tt.want, got,
				"should fallback to provided project ID when timeout occurs")
		})
	}
}

// TestNewVertexAIClient_InitializationFailure tests initialization failure scenarios.
// FR-003: Bot fails to start during initialization if credentials are missing
func TestNewVertexAIClient_InitializationFailure(t *testing.T) {
	tests := []struct {
		name        string
		projectID   string
		errContains string
	}{
		{
			name:        "empty project ID fails immediately",
			projectID:   "",
			errContains: "GCP_PROJECT_ID",
		},
		{
			name:        "whitespace project ID fails immediately",
			projectID:   "   ",
			errContains: "GCP_PROJECT_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Create client with invalid configuration
			// SC-003: Pass fallback region as parameter
			client, err := llm.NewVertexAIClient(context.Background(), tt.projectID, "us-central1", discardLogger())

			// Then: Should return error for invalid configuration
			require.Error(t, err,
				"should return error for invalid configuration")
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), tt.errContains,
				"error message should contain '%s'", tt.errContains)
		})
	}
}

// TestNewVertexAIClient_EmptyGCPRegion tests that client initialization fails when GCP_REGION is missing.
// AC-005: Add new test for empty GCP_REGION validation
// AC-007: Error is returned when GCP_REGION is empty and metadata unavailable
func TestNewVertexAIClient_EmptyGCPRegion(t *testing.T) {
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
			wantErrMsg: "GCP_REGION is missing or empty",
		},
		{
			name:       "whitespace-only region returns error",
			region:     "   ",
			wantErr:    true,
			wantErrMsg: "GCP_REGION is missing or empty",
		},
		{
			name:       "whitespace with tabs region returns error",
			region:     "\t\n  ",
			wantErr:    true,
			wantErrMsg: "GCP_REGION is missing or empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Valid project ID but empty/whitespace region
			// When: Attempt to create Vertex AI client
			client, err := llm.NewVertexAIClient(context.Background(), "valid-project-id", tt.region, discardLogger())

			// Then: Should return error indicating missing GCP_REGION
			if tt.wantErr {
				require.Error(t, err,
					"should return error when GCP_REGION is missing")
				assert.Nil(t, client,
					"client should be nil when initialization fails")
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate GCP_REGION is missing or empty")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}
