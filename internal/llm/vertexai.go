package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

const (
	// defaultRegion is the default GCP region for Vertex AI API calls.
	defaultRegion = "us-central1"

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
//
// The projectID parameter should typically come from the GCP_PROJECT_ID environment variable.
// Returns an error if projectID is empty or contains only whitespace.
func NewVertexAIClient(ctx context.Context, projectID string) (Provider, error) {
	// Validate projectID is not empty or whitespace
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("GCP_PROJECT_ID is missing or empty")
	}

	// Handle nil context gracefully (SDK may require non-nil context)
	if ctx == nil {
		ctx = context.Background()
	}

	// Determine region from Cloud Run metadata, with fallbacks
	// AC-001: Region derived from Cloud Run metadata
	region := GetRegion(metadataServerURL)

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
// SC-001: Remove hardcoded defaultRegion and use Cloud Run metadata
//
// It attempts to read the region from the Cloud Run metadata server.
// If that fails (timeout, error, malformed response), it falls back to:
// 1. GCP_REGION environment variable
// 2. Hardcoded default: us-central1
//
// The metadataServerURL parameter should be the base URL of the metadata server
// (e.g., "http://metadata.google.internal" in production).
// The function appends "/computeMetadata/v1/instance/region" to this URL.
func GetRegion(metadataServerURL string) string {
	// Try to get region from metadata server
	region := getRegionFromMetadata(metadataServerURL)
	if region != "" {
		return region
	}

	// Fallback to environment variable
	envRegion := os.Getenv("GCP_REGION")
	if envRegion != "" {
		return envRegion
	}

	// Fallback to default
	return defaultRegion
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
