package gcp_test

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"yuruppu/internal/gcp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
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

// =============================================================================
// NewClient Tests
// =============================================================================

// TestNewClient tests creating MetadataClient.
// AC-001: MetadataClient can be created with configurable timeout and logger
func TestNewClient(t *testing.T) {
	t.Run("creates client with provided parameters", func(t *testing.T) {
		// Given: Parameters for MetadataClient
		httpClient := &http.Client{Timeout: 5 * time.Second}
		logger := discardLogger()

		// When: Create MetadataClient
		client := gcp.NewClient(gcp.DefaultMetadataServerURL, httpClient, logger)

		// Then: Should create client successfully
		assert.NotNil(t, client, "client should not be nil")
	})
}

// TestDefaultMetadataServerURL tests the default metadata server URL constant.
// AC-001: Default metadata server URL is http://metadata.google.internal
func TestDefaultMetadataServerURL(t *testing.T) {
	t.Run("default URL is correct", func(t *testing.T) {
		assert.Equal(t, "http://metadata.google.internal", gcp.DefaultMetadataServerURL)
	})
}

// =============================================================================
// GetProjectID Tests
// =============================================================================

// TestMetadataClient_GetProjectID tests project ID retrieval.
// AC-001: MetadataClient can fetch project ID with fallback
func TestMetadataClient_GetProjectID(t *testing.T) {
	tests := []struct {
		name              string
		metadataServer    *metadataServerMock
		fallbackProjectID string
		want              string
	}{
		{
			name: "metadata server returns valid project ID",
			metadataServer: &metadataServerMock{
				response:   "test-metadata-project",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "test-fallback-project",
			want:              "test-metadata-project",
		},
		{
			name: "metadata server returns different project ID",
			metadataServer: &metadataServerMock{
				response:   "test-another-project",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "test-fallback-project",
			want:              "test-another-project",
		},
		{
			name: "metadata server unavailable - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 500,
				delay:      0,
			},
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name: "metadata server returns 404 - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 404,
				delay:      0,
			},
			fallbackProjectID: "test-local-project",
			want:              "test-local-project",
		},
		{
			name: "empty response - fallback to provided project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 200,
				delay:      0,
			},
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name: "metadata unavailable - use provided fallback project ID",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 503,
				delay:      0,
			},
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server and MetadataClient
			server := tt.metadataServer.Start(t)
			defer server.Close()

			// Create client with test server URL
			client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

			// When: Get project ID with fallback
			got := client.GetProjectID(tt.fallbackProjectID)

			// Then: Should return expected project ID
			assert.Equal(t, tt.want, got,
				"project ID should match expected value")
		})
	}
}

// TestMetadataClient_GetProjectID_EmptyFallback tests behavior when fallback is empty.
// AC-001: GetProjectID with empty fallback returns empty
func TestMetadataClient_GetProjectID_EmptyFallback(t *testing.T) {
	t.Run("metadata fails and fallback is empty - returns empty", func(t *testing.T) {
		// Given: Mock metadata server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		defer server.Close()

		client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

		// When: Get project ID with empty fallback
		got := client.GetProjectID("")

		// Then: Should return empty string
		assert.Equal(t, "", got,
			"should return empty string when metadata fails and fallback is empty")
	})
}

// TestMetadataClient_GetProjectID_MetadataHeaders tests that metadata requests include required headers.
// AC-001: Metadata request requires header: Metadata-Flavor: Google
func TestMetadataClient_GetProjectID_MetadataHeaders(t *testing.T) {
	t.Run("metadata request includes Metadata-Flavor: Google header", func(t *testing.T) {
		// Given: Mock metadata server that captures request headers
		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(200)
			w.Write([]byte("test-metadata-project"))
		}))
		defer server.Close()

		client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

		// When: Get project ID
		_ = client.GetProjectID("test-fallback-project")

		// Then: Should include Metadata-Flavor header
		require.NotNil(t, capturedHeaders, "request should have been made")
		assert.Equal(t, "Google", capturedHeaders.Get("Metadata-Flavor"),
			"should include Metadata-Flavor: Google header")
	})
}

// TestMetadataClient_GetProjectID_ProductionEndpoint tests the correct endpoint path is used.
// AC-001: Production endpoint format for project ID
func TestMetadataClient_GetProjectID_ProductionEndpoint(t *testing.T) {
	t.Run("uses correct metadata endpoint path for project ID", func(t *testing.T) {
		// Given: Mock server that verifies the request path
		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(200)
			w.Write([]byte("test-metadata-project"))
		}))
		defer server.Close()

		client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

		// When: Get project ID
		_ = client.GetProjectID("test-fallback-project")

		// Then: Should request the correct path
		expectedPath := "/computeMetadata/v1/project/project-id"
		assert.Equal(t, expectedPath, requestedPath,
			"should request metadata endpoint path: %s", expectedPath)
	})
}

// TestMetadataClient_GetProjectID_EdgeCases tests edge cases in project ID extraction.
// AC-001: Project ID format validation and edge cases
func TestMetadataClient_GetProjectID_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		metadataResponse  string
		fallbackProjectID string
		want              string
	}{
		{
			name:              "response with trailing newline - trim and use",
			metadataResponse:  "test-metadata-project\n",
			fallbackProjectID: "test-fallback-project",
			want:              "test-metadata-project",
		},
		{
			name:              "response with CRLF - trim and use",
			metadataResponse:  "test-metadata-project\r\n",
			fallbackProjectID: "test-fallback-project",
			want:              "test-metadata-project",
		},
		{
			name:              "response with leading spaces - fallback (malformed)",
			metadataResponse:  "  test-metadata-project",
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name:              "response with trailing spaces - fallback (malformed)",
			metadataResponse:  "test-metadata-project  ",
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name:              "whitespace only - fallback",
			metadataResponse:  "   ",
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name:              "empty string - fallback",
			metadataResponse:  "",
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
		},
		{
			name:              "newline only - fallback",
			metadataResponse:  "\n",
			fallbackProjectID: "test-fallback-project",
			want:              "test-fallback-project",
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

			client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

			// When: Get project ID with fallback
			got := client.GetProjectID(tt.fallbackProjectID)

			// Then: Should handle edge case correctly
			assert.Equal(t, tt.want, got,
				"should handle edge case correctly")
		})
	}
}

// TestMetadataClient_GetProjectID_Timeout tests timeout behavior using synctest.
// AC-002: Timeout tests using synctest work correctly
// ADR: 20251227-fake-time-testing.md - Uses testing/synctest for deterministic timeout testing
func TestMetadataClient_GetProjectID_Timeout(t *testing.T) {
	t.Run("request completes before timeout - use metadata", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetProjectIDTimeout(t, 1*time.Second, "test-fallback-project", "test-metadata-project")
		})
	})

	t.Run("request takes exactly 2 seconds - timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetProjectIDTimeout(t, 2*time.Second, "test-fallback-project", "test-fallback-project")
		})
	})

	t.Run("request takes longer than timeout - fallback", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetProjectIDTimeout(t, 5*time.Second, "test-fallback-project", "test-fallback-project")
		})
	})
}

// testGetProjectIDTimeout tests GetProjectID timeout behavior using synctest fake time.
// It creates an in-memory HTTP server that delays the response by serverDelay.
func testGetProjectIDTimeout(t *testing.T, serverDelay time.Duration, fallbackProjectID, want string) {
	t.Helper()

	// Create in-memory connection pair
	srvConn, cliConn := net.Pipe()

	// Configure HTTP client with custom transport that uses the pipe
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return cliConn, nil
			},
		},
	}

	// Create MetadataClient with test HTTP client (baseURL doesn't matter with custom transport)
	client := gcp.NewClient("http://test", httpClient, discardLogger())

	// Channel to signal server goroutine completion
	serverDone := make(chan struct{})

	// Channel to cancel the server delay
	cancelDelay := make(chan struct{})

	// Start fake server goroutine
	go func() {
		defer close(serverDone)
		defer srvConn.Close()

		// Read the HTTP request
		req, err := http.ReadRequest(bufio.NewReader(srvConn))
		if err != nil {
			// Client closed connection (timeout case)
			return
		}
		req.Body.Close()

		// Wait for delay or cancellation
		timer := time.NewTimer(serverDelay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Timer expired, send response
		case <-cancelDelay:
			// Cancelled, exit without sending response
			return
		}

		// Write response
		resp := &http.Response{
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       http.NoBody,
		}
		resp.Header.Set("Content-Type", "text/plain")

		// For project ID endpoint, return plain project ID
		body := "test-metadata-project"
		resp.Body = io.NopCloser(strings.NewReader(body))
		resp.ContentLength = int64(len(body))

		resp.Write(srvConn)
	}()

	// Call GetProjectID (uses any URL since we override transport)
	got := client.GetProjectID(fallbackProjectID)

	// Close client connection and cancel server delay to unblock server
	cliConn.Close()
	close(cancelDelay)

	// Wait for server goroutine to finish
	<-serverDone

	assert.Equal(t, want, got)
}

// =============================================================================
// GetRegion Tests
// =============================================================================

// TestMetadataClient_GetRegion tests region retrieval.
// AC-001: MetadataClient can fetch region with fallback
func TestMetadataClient_GetRegion(t *testing.T) {
	tests := []struct {
		name           string
		metadataServer *metadataServerMock
		fallbackRegion string
		want           string
	}{
		{
			name: "metadata server returns valid response - extract region correctly",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789/regions/test-metadata-region",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "test-region",
			want:           "test-metadata-region",
		},
		{
			name: "metadata server returns different region - extract correctly",
			metadataServer: &metadataServerMock{
				response:   "projects/987654321/regions/test-region",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "test-region",
			want:           "test-region",
		},
		{
			name: "metadata server unavailable - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 500,
				delay:      0,
			},
			fallbackRegion: "test-region-eu",
			want:           "test-region-eu",
		},
		{
			name: "metadata server returns 404 - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 404,
				delay:      0,
			},
			fallbackRegion: "test-region",
			want:           "test-region",
		},
		{
			name: "malformed response (not projects/*/regions/*) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "invalid-format",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "test-region-west2",
			want:           "test-region-west2",
		},
		{
			name: "malformed response (missing region part) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "projects/123456789",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "test-region-south1",
			want:           "test-region-south1",
		},
		{
			name: "malformed response (empty string) - fallback to provided region",
			metadataServer: &metadataServerMock{
				response:   "",
				statusCode: 200,
				delay:      0,
			},
			fallbackRegion: "test-region-east4",
			want:           "test-region-east4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server and MetadataClient
			server := tt.metadataServer.Start(t)
			defer server.Close()

			client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

			// When: Get region with fallback
			got := client.GetRegion(tt.fallbackRegion)

			// Then: Should return expected region
			assert.Equal(t, tt.want, got,
				"region should match expected value")
		})
	}
}

// TestMetadataClient_GetRegion_ParsingFormats tests region parsing from different formats.
// AC-001: Parsing region from "projects/123/regions/us-central1" format
func TestMetadataClient_GetRegion_ParsingFormats(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		fallbackRegion string
		want           string
	}{
		{
			name:           "standard format with us-central1",
			response:       "projects/123456789/regions/us-central1",
			fallbackRegion: "test-fallback-region",
			want:           "us-central1",
		},
		{
			name:           "standard format with asia-northeast1",
			response:       "projects/987654321/regions/asia-northeast1",
			fallbackRegion: "test-fallback-region",
			want:           "asia-northeast1",
		},
		{
			name:           "standard format with europe-west1",
			response:       "projects/111111111/regions/europe-west1",
			fallbackRegion: "test-fallback-region",
			want:           "europe-west1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Mock metadata server returning formatted response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

			// When: Get region
			got := client.GetRegion(tt.fallbackRegion)

			// Then: Should parse region correctly
			assert.Equal(t, tt.want, got,
				"should parse region from format correctly")
		})
	}
}

// TestMetadataClient_GetRegion_MetadataHeaders tests that metadata requests include required headers.
// AC-001: Metadata request requires header: Metadata-Flavor: Google
func TestMetadataClient_GetRegion_MetadataHeaders(t *testing.T) {
	t.Run("metadata request includes Metadata-Flavor: Google header", func(t *testing.T) {
		// Given: Mock metadata server that captures request headers
		var capturedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeaders = r.Header
			w.WriteHeader(200)
			w.Write([]byte("projects/123456789/regions/test-metadata-region"))
		}))
		defer server.Close()

		client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

		// When: Get region
		_ = client.GetRegion("test-region")

		// Then: Should include Metadata-Flavor header
		require.NotNil(t, capturedHeaders, "request should have been made")
		assert.Equal(t, "Google", capturedHeaders.Get("Metadata-Flavor"),
			"should include Metadata-Flavor: Google header")
	})
}

// TestMetadataClient_GetRegion_ProductionEndpoint tests the correct endpoint path is used.
// AC-001: Production endpoint format
func TestMetadataClient_GetRegion_ProductionEndpoint(t *testing.T) {
	t.Run("uses correct metadata endpoint path", func(t *testing.T) {
		// Given: Mock server that verifies the request path
		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(200)
			w.Write([]byte("projects/123456789/regions/test-metadata-region"))
		}))
		defer server.Close()

		client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

		// When: Get region
		_ = client.GetRegion("test-region")

		// Then: Should request the correct path
		expectedPath := "/computeMetadata/v1/instance/region"
		assert.Equal(t, expectedPath, requestedPath,
			"should request metadata endpoint path: %s", expectedPath)
	})
}

// TestMetadataClient_GetRegion_EdgeCases tests edge cases in region extraction.
// AC-001: Region format validation and edge cases
func TestMetadataClient_GetRegion_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		metadataResponse string
		fallbackRegion   string
		want             string
	}{
		{
			name:             "response with extra slashes",
			metadataResponse: "projects/123456789/regions/test-metadata-region/",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region", // malformed, fallback
		},
		{
			name:             "response with leading spaces",
			metadataResponse: "  projects/123456789/regions/test-metadata-region",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region", // malformed, fallback
		},
		{
			name:             "response with trailing spaces",
			metadataResponse: "projects/123456789/regions/test-metadata-region  ",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region", // malformed, fallback
		},
		{
			name:             "response with newline",
			metadataResponse: "projects/123456789/regions/test-metadata-region\n",
			fallbackRegion:   "test-fallback-region",
			want:             "test-metadata-region", // trimmed newline is acceptable
		},
		{
			name:             "response with only project",
			metadataResponse: "projects/123456789",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region",
		},
		{
			name:             "response with wrong format (regions first)",
			metadataResponse: "regions/test-metadata-region/projects/123456789",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region",
		},
		{
			name:             "response with region but no project number",
			metadataResponse: "projects//regions/test-metadata-region",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region", // malformed, fallback
		},
		{
			name:             "empty region name",
			metadataResponse: "projects/123456789/regions/",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region",
		},
		{
			name:             "only slashes",
			metadataResponse: "///",
			fallbackRegion:   "test-fallback-region",
			want:             "test-fallback-region",
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

			client := gcp.NewClient(server.URL, http.DefaultClient, discardLogger())

			// When: Get region with fallback
			got := client.GetRegion(tt.fallbackRegion)

			// Then: Should handle edge case correctly
			assert.Equal(t, tt.want, got,
				"should handle edge case correctly")
		})
	}
}

// TestMetadataClient_GetRegion_Timeout tests timeout behavior using synctest.
// AC-002: Timeout tests using synctest work correctly
// ADR: 20251227-fake-time-testing.md - Uses testing/synctest for deterministic timeout testing
func TestMetadataClient_GetRegion_Timeout(t *testing.T) {
	t.Run("request completes before timeout - use metadata", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetRegionTimeout(t, 1*time.Second, "test-fallback-region", "test-metadata-region")
		})
	})

	t.Run("request takes exactly 2 seconds - timeout", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetRegionTimeout(t, 2*time.Second, "test-fallback-region", "test-fallback-region")
		})
	})

	t.Run("request takes longer than timeout - fallback", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			testGetRegionTimeout(t, 5*time.Second, "test-region-east1", "test-region-east1")
		})
	})
}

// testGetRegionTimeout tests GetRegion timeout behavior using synctest fake time.
// It creates an in-memory HTTP server that delays the response by serverDelay.
func testGetRegionTimeout(t *testing.T, serverDelay time.Duration, fallbackRegion, want string) {
	t.Helper()

	// Create in-memory connection pair
	srvConn, cliConn := net.Pipe()

	// Configure HTTP client with custom transport that uses the pipe
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return cliConn, nil
			},
		},
	}

	// Create MetadataClient with test HTTP client (baseURL doesn't matter with custom transport)
	client := gcp.NewClient("http://test", httpClient, discardLogger())

	// Channel to signal server goroutine completion
	serverDone := make(chan struct{})

	// Channel to cancel the server delay
	cancelDelay := make(chan struct{})

	// Start fake server goroutine
	go func() {
		defer close(serverDone)
		defer srvConn.Close()

		// Read the HTTP request
		req, err := http.ReadRequest(bufio.NewReader(srvConn))
		if err != nil {
			// Client closed connection (timeout case)
			return
		}
		req.Body.Close()

		// Wait for delay or cancellation
		timer := time.NewTimer(serverDelay)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Timer expired, send response
		case <-cancelDelay:
			// Cancelled, exit without sending response
			return
		}

		// Write response
		resp := &http.Response{
			StatusCode: 200,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       http.NoBody,
		}
		resp.Header.Set("Content-Type", "text/plain")

		// For region endpoint, return the metadata format
		body := "projects/123456789/regions/test-metadata-region"
		resp.Body = io.NopCloser(strings.NewReader(body))
		resp.ContentLength = int64(len(body))

		resp.Write(srvConn)
	}()

	// Call GetRegion
	got := client.GetRegion(fallbackRegion)

	// Close client connection and cancel server delay to unblock server
	cliConn.Close()
	close(cancelDelay)

	// Wait for server goroutine to finish
	<-serverDone

	assert.Equal(t, want, got)
}
