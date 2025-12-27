package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/genai"
)

// vertexAIClient is an implementation of Provider using Google Vertex AI.
type vertexAIClient struct {
	client    *genai.Client
	projectID string
	model     string
	logger    *slog.Logger
}

// NewVertexAIClient creates a new Vertex AI client.
// FR-003: Load LLM API credentials from environment variables
// AC-012: Bot initializes LLM client successfully when credentials are set
// AC-013: Bot fails to start during initialization if credentials are missing
//
// The projectID, region, and model parameters must be pre-resolved by the caller.
// Use gcp.MetadataClient to resolve projectID and region from Cloud Run metadata server
// with fallback to environment variables before calling this function.
// model is the Gemini model name to use (e.g., "gemini-2.5-flash-lite").
// logger is the structured logger for the client.
// Returns an error if projectID, region, or model is empty or whitespace-only.
func NewVertexAIClient(ctx context.Context, projectID string, region string, model string, logger *slog.Logger) (Provider, error) {
	// Handle nil context gracefully (SDK may require non-nil context)
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate projectID is not empty or whitespace
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("projectID is required")
	}

	// Validate region is not empty or whitespace
	if strings.TrimSpace(region) == "" {
		return nil, errors.New("region is required")
	}

	// Validate model is not empty or whitespace
	if strings.TrimSpace(model) == "" {
		return nil, errors.New("model is required")
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
		model:     model,
		logger:    logger,
	}, nil
}

// GenerateText generates a text response given a system prompt and user message.
// TR-002: Implements Provider interface for LLM abstraction
//
// The context can be used for timeout and cancellation.
// NFR-001: LLM API total request timeout should be configurable via context
func (v *vertexAIClient) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	v.logger.Debug("generating text",
		slog.String("model", v.model),
		slog.Int("userMessageLength", len(userMessage)),
	)

	// Configure generation with system instruction
	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		},
	}

	// Generate content
	resp, err := v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)
	if err != nil {
		v.logger.Error("LLM API call failed",
			slog.String("model", v.model),
			slog.Any("error", err),
		)
		// FR-004: Map specific errors to custom error types
		return "", MapAPIError(err)
	}

	// Extract text from response
	if len(resp.Candidates) == 0 {
		v.logger.Error("LLM response error",
			slog.String("reason", "no candidates in response"),
		)
		return "", &LLMResponseError{Message: "no candidates in response"}
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		v.logger.Error("LLM response error",
			slog.String("reason", "no content parts in response"),
		)
		return "", &LLMResponseError{Message: "no content parts in response"}
	}

	// Extract text from first part
	text := resp.Candidates[0].Content.Parts[0].Text
	if text == "" {
		v.logger.Error("LLM response error",
			slog.String("reason", "response part has no text"),
		)
		return "", &LLMResponseError{Message: "response part has no text"}
	}

	// AC-002: Log resp.ModelVersion for verification that correct model is used
	v.logger.Info("text generated successfully",
		slog.String("model", v.model),
		slog.String("modelVersion", resp.ModelVersion),
		slog.Int("responseLength", len(text)),
	)

	return text, nil
}
