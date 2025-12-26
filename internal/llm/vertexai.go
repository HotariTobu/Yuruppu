package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

const (
	// metadataServerURL is the Cloud Run metadata server URL.
	metadataServerURL = "http://metadata.google.internal"

	// geminiModel is the Gemini model to use for text generation.
	// ADR: 20251225-gemini-model-selection.md - Using Gemini 2.5 Flash-Lite
	geminiModel = "gemini-2.5-flash-lite"
)

// vertexAIClient is an implementation of Provider using Google Vertex AI.
type vertexAIClient struct {
	client    *genai.Client
	projectID string
}

// NewVertexAIClient creates a new Vertex AI client.
// FR-003: Load LLM API credentials from environment variables
// AC-012: Bot initializes LLM client successfully when credentials are set
// AC-013: Bot fails to start during initialization if credentials are missing
// SC-003: Accept fallbackRegion as parameter instead of reading from environment
// FX-001: Auto-detect project ID from Cloud Run metadata server
//
// The fallbackProjectID parameter should come from the GCP_PROJECT_ID environment variable (optional on Cloud Run).
// The fallbackRegion parameter should come from the GCP_REGION environment variable (via Config.GCPRegion).
// On Cloud Run, project ID and region are auto-detected from metadata server.
// Returns an error if project ID or region cannot be determined from either metadata or fallback.
func NewVertexAIClient(ctx context.Context, fallbackProjectID string, fallbackRegion string) (Provider, error) {
	// Handle nil context gracefully (SDK may require non-nil context)
	if ctx == nil {
		ctx = context.Background()
	}

	// Determine project ID from Cloud Run metadata, with fallback to provided project ID
	projectID := GetProjectID(metadataServerURL, fallbackProjectID)

	// Determine region from Cloud Run metadata, with fallback to provided region
	region := GetRegion(metadataServerURL, fallbackRegion)

	// Validate projectID is not empty or whitespace
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("GCP_PROJECT_ID is missing or empty")
	}

	// Validate region is not empty or whitespace
	if strings.TrimSpace(region) == "" {
		return nil, errors.New("GCP_REGION is missing or empty")
	}

	// Create Vertex AI client
	// ADR: 20251224-llm-provider.md - Uses Application Default Credentials (ADC)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: region,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	return &vertexAIClient{
		client:    client,
		projectID: projectID,
	}, nil
}

// GenerateText generates a text response given a system prompt and user message.
// TR-002: Implements Provider interface for LLM abstraction
//
// The context can be used for timeout and cancellation.
// NFR-001: LLM API total request timeout should be configurable via context
func (v *vertexAIClient) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// Configure generation with system instruction
	// ADR: 20251225-gemini-model-selection.md - Gemini 2.5 Flash-Lite
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
	}

	// Generate content
	resp, err := v.client.Models.GenerateContent(ctx, geminiModel, genai.Text(userMessage), config)
	if err != nil {
		// FR-004: Map specific errors to custom error types
		return "", MapAPIError(err)
	}

	// Extract text from response
	if len(resp.Candidates) == 0 {
		return "", &LLMResponseError{Message: "no candidates in response"}
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", &LLMResponseError{Message: "no content parts in response"}
	}

	// Extract text from first part
	text := resp.Candidates[0].Content.Parts[0].Text
	if text == "" {
		return "", &LLMResponseError{Message: "response part has no text"}
	}

	return text, nil
}

// GetRegion determines the GCP region to use for Vertex AI API calls.
// AC-001: Region derived from Cloud Run metadata
// SC-004: Accept fallbackRegion as parameter instead of reading from environment
//
// It attempts to read the region from the Cloud Run metadata server.
// If that fails (timeout, error, malformed response), it falls back to the provided fallbackRegion.
//
// The metadataServerURL parameter should be the base URL of the metadata server
// (e.g., "http://metadata.google.internal" in production).
// The function appends "/computeMetadata/v1/instance/region" to this URL.
// The fallbackRegion parameter should come from the GCP_REGION environment variable (via Config.GCPRegion).
func GetRegion(metadataServerURL string, fallbackRegion string) string {
	// Try to get region from metadata server
	region := getRegionFromMetadata(metadataServerURL)
	if region != "" {
		return region
	}

	// Fallback to provided region
	return fallbackRegion
}

// getRegionFromMetadata attempts to retrieve the region from the Cloud Run metadata server.
// Returns empty string on any failure (timeout, HTTP error, malformed response).
func getRegionFromMetadata(baseURL string) string {
	// Create HTTP client with 2-second timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Construct metadata endpoint URL
	url := baseURL + "/computeMetadata/v1/instance/region"

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	// Add required header
	req.Header.Set("Metadata-Flavor", "Google")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Parse response format: projects/PROJECT-NUMBER/regions/REGION
	// parseRegionFromResponse will handle trimming and validation
	return parseRegionFromResponse(string(body))
}

// GetProjectID determines the GCP project ID to use for Vertex AI API calls.
// FX-001: Add GetProjectID function following the GetRegion() pattern
// AC-001: Auto-detect project ID on Cloud Run
// AC-002: Fallback to env var when metadata unavailable
//
// It attempts to read the project ID from the Cloud Run metadata server.
// If that fails (timeout, error, empty response), it falls back to the provided fallbackProjectID.
//
// The metadataServerURL parameter should be the base URL of the metadata server
// (e.g., "http://metadata.google.internal" in production).
// The function appends "/computeMetadata/v1/project/project-id" to this URL.
// The fallbackProjectID parameter should come from the GCP_PROJECT_ID environment variable (via Config.GCPProjectID).
func GetProjectID(metadataServerURL string, fallbackProjectID string) string {
	// Try to get project ID from metadata server
	projectID := getProjectIDFromMetadata(metadataServerURL)
	if projectID != "" {
		return projectID
	}

	// Fallback to provided project ID
	return fallbackProjectID
}

// getProjectIDFromMetadata attempts to retrieve the project ID from the Cloud Run metadata server.
// Returns empty string on any failure (timeout, HTTP error, empty response).
func getProjectIDFromMetadata(baseURL string) string {
	// Create HTTP client with 2-second timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Construct metadata endpoint URL
	url := baseURL + "/computeMetadata/v1/project/project-id"

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	// Add required header
	req.Header.Set("Metadata-Flavor", "Google")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	// Parse response - project ID is returned as plain text
	return parseProjectIDFromResponse(string(body))
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
