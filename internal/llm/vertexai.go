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

// DefaultCacheTTL is the default time-to-live for cached content.
// CH-001: Context caching for system prompts with reasonable TTL
const DefaultCacheTTL = 60 * time.Minute

// vertexAIClient is an implementation of Provider using Google Vertex AI.
type vertexAIClient struct {
	client       *genai.Client
	projectID    string
	model        string
	logger       *slog.Logger
	closed       bool
	mu           sync.RWMutex // Protects closed and cacheName fields
	systemPrompt string       // CH-001: System prompt for caching and fallback
	cacheName    string       // CH-001: Name of cached content (empty if caching not used)
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

// NewVertexAIClientWithCache creates a new Vertex AI client with context caching enabled.
// CH-001: Add context caching for system prompt
// AC-001: System prompt is cached for reuse across requests with reasonable TTL
// AC-007: Caching is skipped gracefully when token count is insufficient
//
// The systemPrompt is cached for reuse across all GenerateText calls.
// If caching fails (e.g., system prompt below minimum token requirement of 32K),
// the client falls back to non-cached mode gracefully.
func NewVertexAIClientWithCache(ctx context.Context, projectID, region, model, systemPrompt string, logger *slog.Logger) (Provider, error) {
	// Handle nil context gracefully
	if ctx == nil {
		ctx = context.Background()
	}

	// Normalize and validate inputs
	projectID = strings.TrimSpace(projectID)
	region = strings.TrimSpace(region)
	model = strings.TrimSpace(model)
	systemPrompt = strings.TrimSpace(systemPrompt)

	if projectID == "" {
		return nil, errors.New("projectID is required")
	}

	if region == "" {
		return nil, errors.New("region is required")
	}

	if model == "" {
		return nil, errors.New("model is required")
	}

	if systemPrompt == "" {
		return nil, errors.New("systemPrompt is required")
	}

	// Create Vertex AI client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: region,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	v := &vertexAIClient{
		client:       client,
		projectID:    projectID,
		model:        model,
		logger:       logger,
		systemPrompt: systemPrompt,
	}

	// Attempt to create cache
	// AC-001: Cache has a reasonable TTL (60 minutes)
	// AC-007: If caching fails, continue without caching
	cacheName, err := v.createCache(ctx)
	if err != nil {
		// AC-007: Caching is skipped gracefully, log info and continue
		logger.Info("context caching not available, using fallback mode",
			slog.String("reason", err.Error()),
		)
	} else {
		v.cacheName = cacheName
		logger.Info("context cache created successfully",
			slog.String("cacheName", cacheName),
		)
	}

	return v, nil
}

// createCache creates a cached content for the system prompt.
// Returns the cache name on success, or an error if caching fails.
func (v *vertexAIClient) createCache(ctx context.Context) (string, error) {
	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: v.systemPrompt},
			},
		},
	}

	cache, err := v.client.Caches.Create(ctx, v.model, &genai.CreateCachedContentConfig{
		TTL:         DefaultCacheTTL,
		Contents:    contents,
		DisplayName: "yuruppu-system-prompt",
	})
	if err != nil {
		return "", err
	}

	return cache.Name, nil
}

// GenerateText generates a text response given a system prompt and user message.
// TR-002: Implements Provider interface for LLM abstraction
// AC-002: Uses cached system prompt when available
// AC-006: Handles cache expiration by recreating cache
//
// The context can be used for timeout and cancellation.
// NFR-001: LLM API total request timeout should be configurable via context
func (v *vertexAIClient) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	// AC-004: Check if provider is closed before generating text
	v.mu.RLock()
	if v.closed {
		v.mu.RUnlock()
		return "", &LLMClosedError{Message: "provider is closed"}
	}
	cacheName := v.cacheName
	v.mu.RUnlock()

	v.logger.Debug("generating text",
		slog.String("model", v.model),
		slog.Int("userMessageLength", len(userMessage)),
		slog.Bool("usingCache", cacheName != ""),
	)

	// Configure generation
	config := v.buildGenerateConfig(cacheName, systemPrompt)

	// Generate content
	resp, err := v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)

	// AC-006: Handle cache expiration - check for cache-related errors and retry
	if err != nil && cacheName != "" && v.isCacheError(err) {
		resp, err = v.handleCacheErrorAndRetry(ctx, err, systemPrompt, userMessage)
	}

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
		slog.Bool("usedCache", cacheName != ""),
	)

	return text, nil
}

// buildGenerateConfig creates the GenerateContentConfig for API calls.
// AC-002: Uses cached content when available, otherwise uses system instruction.
func (v *vertexAIClient) buildGenerateConfig(cacheName, systemPrompt string) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if cacheName != "" {
		// AC-002: Use cached content
		config.CachedContent = cacheName
	} else {
		// Fallback: Use system instruction directly
		// This is used when:
		// - Client was created without caching (NewVertexAIClient)
		// - Cache creation failed (AC-007)
		// - Cache expired and recreation failed (AC-006)
		effectivePrompt := systemPrompt
		if v.systemPrompt != "" {
			// For cached clients in fallback mode, use stored system prompt
			effectivePrompt = v.systemPrompt
		}
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: effectivePrompt}},
		}
	}

	return config
}

// isCacheError checks if an error is related to cache (expired, not found, etc.).
// AC-006: Cache expiration handling
func (v *vertexAIClient) isCacheError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common cache-related error messages
	errStr := err.Error()
	return strings.Contains(errStr, "cache") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "expired")
}

// handleCacheErrorAndRetry attempts to recreate the cache and retry the request.
// AC-006: Cache is recreated automatically when expired or deleted.
func (v *vertexAIClient) handleCacheErrorAndRetry(ctx context.Context, originalErr error, systemPrompt, userMessage string) (*genai.GenerateContentResponse, error) {
	v.logger.Warn("cache error detected, attempting to recreate cache",
		slog.Any("error", originalErr),
	)

	// Try to recreate cache
	newCacheName, cacheErr := v.createCache(ctx)
	if cacheErr == nil {
		v.mu.Lock()
		v.cacheName = newCacheName
		v.mu.Unlock()
		v.logger.Info("cache recreated successfully",
			slog.String("cacheName", newCacheName),
		)
		// Retry with new cache
		config := v.buildGenerateConfig(newCacheName, systemPrompt)
		return v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)
	}

	// Cache recreation failed, fall back to non-cached mode
	v.logger.Warn("cache recreation failed, using fallback mode",
		slog.Any("error", cacheErr),
	)
	v.mu.Lock()
	v.cacheName = ""
	v.mu.Unlock()
	config := v.buildGenerateConfig("", systemPrompt)
	return v.client.Models.GenerateContent(ctx, v.model, genai.Text(userMessage), config)
}

// Close releases any resources held by the client.
// AC-004: Provider lifecycle management
// - Close is idempotent (safe to call multiple times)
// - After Close, subsequent GenerateText calls return an error
// - Cached resources are cleaned up
func (v *vertexAIClient) Close(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Idempotent: if already closed, return immediately without error
	if v.closed {
		return nil
	}

	// AC-004: Clean up cached content if present
	if v.cacheName != "" {
		_, err := v.client.Caches.Delete(ctx, v.cacheName, nil)
		if err != nil {
			v.logger.Warn("failed to delete cached content during close",
				slog.String("cacheName", v.cacheName),
				slog.Any("error", err),
			)
			// Continue with close even if cache deletion fails
		} else {
			v.logger.Debug("cached content deleted",
				slog.String("cacheName", v.cacheName),
			)
		}
		v.cacheName = ""
	}

	v.closed = true
	v.logger.Debug("provider closed")

	return nil
}
