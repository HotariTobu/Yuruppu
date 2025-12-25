package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

const (
	// defaultRegion is the default GCP region for Vertex AI API calls.
	defaultRegion = "us-central1"

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

	// Create Vertex AI client
	// ADR: 20251224-llm-provider.md - Uses Application Default Credentials (ADC)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: defaultRegion,
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
