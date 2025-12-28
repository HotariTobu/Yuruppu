package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

// disabledThinkingBudget disables the thinking feature in Gemini models.
// AC-003: Set to 0 to prevent the model from showing reasoning steps.
const disabledThinkingBudget = int32(0)

// vertexAIClient is an implementation of Provider using Google Vertex AI.
// AC-001: Provider is a pure API layer - no caching state management.
// Cache lifecycle is managed by the Agent component.
type vertexAIClient struct {
	client    *genai.Client
	projectID string
	model     string
	logger    *slog.Logger
	closed    bool
	mu        sync.RWMutex // Protects closed field
}

// NewVertexAIClient creates a new Vertex AI client.
// FR-003: Load LLM API credentials from environment variables
// AC-012: Bot initializes LLM client successfully when credentials are set
// AC-013: Bot fails to start during initialization if credentials are missing
//
// The projectID, region, and model parameters must be pre-resolved by the caller.
// Use gcp.MetadataClient to resolve projectID and region from Cloud Run metadata server
// with fallback to environment variables before calling this function.
// model is the Gemini model name to use.
// logger is the structured logger for the client.
// Returns an error if projectID, region, or model is empty or whitespace-only.
func NewVertexAIClient(ctx context.Context, projectID string, region string, model string, logger *slog.Logger) (Provider, error) {
	// Handle nil context gracefully (SDK may require non-nil context)
	if ctx == nil {
		ctx = context.Background()
	}

	// Normalize and validate inputs - trim whitespace before validation and storage
	projectID = strings.TrimSpace(projectID)
	region = strings.TrimSpace(region)
	model = strings.TrimSpace(model)

	if projectID == "" {
		return nil, errors.New("projectID is required")
	}

	if region == "" {
		return nil, errors.New("region is required")
	}

	if model == "" {
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
// AC-001: Pure API layer - no caching logic. The system prompt is sent directly with each request.
//
// The context can be used for timeout and cancellation.
// NFR-001: LLM API total request timeout should be configurable via context
func (v *vertexAIClient) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if err := v.checkClosed(); err != nil {
		return "", err
	}

	v.logger.Debug("generating text",
		slog.String("model", v.model),
		slog.Int("userMessageLength", len(userMessage)),
	)

	config := v.createGenerateConfig(&genai.Content{
		Parts: []*genai.Part{{Text: systemPrompt}},
	}, "")

	resp, err := v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)
	if err != nil {
		v.logger.Error("LLM API call failed",
			slog.String("model", v.model),
			slog.Any("error", err),
		)
		return "", MapAPIError(err)
	}

	return v.extractTextFromResponse(resp)
}

// GenerateTextCached generates a text response using a cached system prompt.
// AC-001: Uses provided cacheName directly (pure API layer, no internal state).
func (v *vertexAIClient) GenerateTextCached(ctx context.Context, cacheName, userMessage string) (string, error) {
	if err := v.checkClosed(); err != nil {
		return "", err
	}

	v.logger.Debug("generating text with cache",
		slog.String("model", v.model),
		slog.Int("userMessageLength", len(userMessage)),
		slog.String("cacheName", cacheName),
	)

	config := v.createGenerateConfig(nil, cacheName)

	resp, err := v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)
	if err != nil {
		v.logger.Error("LLM API call failed (cached)",
			slog.String("model", v.model),
			slog.String("cacheName", cacheName),
			slog.Any("error", err),
		)
		return "", MapAPIError(err)
	}

	return v.extractTextFromResponse(resp)
}

// checkClosed checks if the provider is closed and returns an error if so.
func (v *vertexAIClient) checkClosed() error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.closed {
		return &LLMClosedError{Message: "provider is closed"}
	}
	return nil
}

// createGenerateConfig creates a GenerateContentConfig with thinking disabled.
// Exactly one of systemInstruction or cacheName should be provided:
// - systemInstruction: for non-cached requests with inline system prompt
// - cacheName: for cached requests using pre-cached system prompt
func (v *vertexAIClient) createGenerateConfig(systemInstruction *genai.Content, cacheName string) *genai.GenerateContentConfig {
	if systemInstruction != nil && cacheName != "" {
		v.logger.Warn("both systemInstruction and cacheName provided, using cacheName")
	}

	budget := disabledThinkingBudget
	config := &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &budget,
		},
	}
	if cacheName != "" {
		config.CachedContent = cacheName
	} else if systemInstruction != nil {
		config.SystemInstruction = systemInstruction
	}
	return config
}

// extractTextFromResponse extracts text from LLM response.
// Validates the response structure: candidates -> content -> parts -> text.
func (v *vertexAIClient) extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
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

	part := resp.Candidates[0].Content.Parts[0]
	if part == nil {
		v.logger.Error("LLM response error",
			slog.String("reason", "response part is nil"),
		)
		return "", &LLMResponseError{Message: "response part is nil"}
	}

	text := part.Text
	if text == "" {
		v.logger.Error("LLM response error",
			slog.String("reason", "response part has no text"),
		)
		return "", &LLMResponseError{Message: "response part has no text"}
	}

	v.logger.Info("text generated successfully",
		slog.String("model", v.model),
		slog.String("modelVersion", resp.ModelVersion),
		slog.Int("responseLength", len(text)),
	)

	return text, nil
}

// CreateCache creates a cached content for the given system prompt.
// AC-001: Returns cacheName but does not store it internally (pure API layer).
func (v *vertexAIClient) CreateCache(ctx context.Context, systemPrompt string, ttl time.Duration) (string, error) {
	if err := v.checkClosed(); err != nil {
		return "", err
	}

	v.logger.Debug("creating cache",
		slog.String("model", v.model),
		slog.Int("systemPromptLength", len(systemPrompt)),
		slog.Duration("ttl", ttl),
	)

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: systemPrompt},
			},
		},
	}

	cache, err := v.client.Caches.Create(ctx, v.model, &genai.CreateCachedContentConfig{
		TTL:         ttl,
		Contents:    contents,
		DisplayName: "yuruppu-system-prompt",
	})
	if err != nil {
		v.logger.Error("cache creation failed",
			slog.String("model", v.model),
			slog.Any("error", err),
		)
		return "", err
	}

	v.logger.Info("cache created successfully",
		slog.String("cacheName", cache.Name),
	)

	return cache.Name, nil
}

// DeleteCache deletes the specified cache.
// AC-001: Deletes the cache but does not update internal state (pure API layer).
func (v *vertexAIClient) DeleteCache(ctx context.Context, cacheName string) error {
	// Note: DeleteCache can be called even after Close for cleanup purposes
	v.logger.Debug("deleting cache",
		slog.String("cacheName", cacheName),
	)

	_, err := v.client.Caches.Delete(ctx, cacheName, nil)
	if err != nil {
		v.logger.Warn("cache deletion failed",
			slog.String("cacheName", cacheName),
			slog.Any("error", err),
		)
		return err
	}

	v.logger.Debug("cache deleted successfully",
		slog.String("cacheName", cacheName),
	)

	return nil
}

// Close releases any resources held by the client.
// AC-004: Provider lifecycle management
// AC-001: Provider is a pure API layer - no cache state to clean up.
// Cache cleanup is the responsibility of the Agent component.
// - Close is idempotent (safe to call multiple times)
// - After Close, subsequent GenerateText calls return an error
func (v *vertexAIClient) Close(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Idempotent: if already closed, return immediately without error
	if v.closed {
		return nil
	}

	v.closed = true
	v.logger.Debug("provider closed")

	return nil
}
