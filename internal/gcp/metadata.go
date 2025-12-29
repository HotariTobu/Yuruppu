package gcp

import (
	// Standard library
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// DefaultMetadataServerURL is the Cloud Run metadata server URL.
const DefaultMetadataServerURL = "http://metadata.google.internal"

// MetadataClient fetches project ID and region from GCP metadata server.
type MetadataClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new MetadataClient.
// The baseURL should be DefaultMetadataServerURL in production.
// The httpClient should have an appropriate timeout configured.
// The logger is used for error logging.
func NewClient(baseURL string, httpClient *http.Client, logger *slog.Logger) *MetadataClient {
	return &MetadataClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

// GetProjectID fetches the project ID from the metadata server.
// If the metadata server is unavailable or returns an error, it returns the fallback.
func (c *MetadataClient) GetProjectID(fallback string) string {
	projectID := c.fetchMetadata("/computeMetadata/v1/project/project-id", parseProjectIDFromResponse)
	if projectID != "" {
		return projectID
	}
	return fallback
}

// GetRegion fetches the region from the metadata server.
// If the metadata server is unavailable or returns an error, it returns the fallback.
func (c *MetadataClient) GetRegion(fallback string) string {
	region := c.fetchMetadata("/computeMetadata/v1/instance/region", parseRegionFromResponse)
	if region != "" {
		return region
	}
	return fallback
}

// fetchMetadata fetches a value from the Cloud Run metadata server.
// Returns empty string on any failure (timeout, HTTP error, malformed response).
// Errors are logged at error level.
// The parser function is applied to the response body to extract the desired value.
func (c *MetadataClient) fetchMetadata(endpoint string, parser func(string) string) string {
	url := c.baseURL + endpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("failed to create metadata request", slog.String("url", url), slog.Any("error", err))
		return ""
	}

	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("metadata request failed", slog.String("url", url), slog.Any("error", err))
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("metadata server returned non-OK status", slog.String("url", url), slog.Int("status", resp.StatusCode))
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("failed to read metadata response", slog.String("url", url), slog.Any("error", err))
		return ""
	}

	return parser(string(body))
}

// parseProjectIDFromResponse extracts the project ID from the metadata server response.
// The response is plain text containing just the project ID.
// Returns empty string if the response is empty or contains invalid characters.
func parseProjectIDFromResponse(response string) string {
	// Trim only trailing newlines/carriage returns
	response = strings.TrimRight(response, "\n\r")

	// Reject responses with leading or trailing spaces
	// (only newlines are acceptable for trimming)
	if strings.TrimSpace(response) != response {
		return ""
	}

	// Project ID must not be empty
	if response == "" {
		return ""
	}

	return response
}

// parseRegionFromResponse extracts the region from the metadata server response.
// Expected format: "projects/PROJECT-NUMBER/regions/REGION"
// Returns empty string if format is invalid.
func parseRegionFromResponse(response string) string {
	// Trim only trailing newlines/carriage returns
	response = strings.TrimRight(response, "\n\r")

	// Reject responses with leading or trailing spaces
	// (only newlines are acceptable for trimming)
	if strings.TrimSpace(response) != response {
		return ""
	}

	// Split by "/"
	parts := strings.Split(response, "/")

	// Expected format: [projects, PROJECT-NUMBER, regions, REGION]
	// Must have exactly 4 parts
	if len(parts) != 4 {
		return ""
	}

	// Validate format
	if parts[0] != "projects" || parts[2] != "regions" {
		return ""
	}

	// Validate project number is not empty
	if parts[1] == "" {
		return ""
	}

	// Extract region (last part)
	region := parts[3]

	// Region must not be empty
	if region == "" {
		return ""
	}

	return region
}
