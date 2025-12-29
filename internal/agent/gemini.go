package agent

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
const disabledThinkingBudget = int32(0)

// GeminiAgent is an implementation of Agent using Google Gemini via Vertex AI.
type GeminiAgent struct {
	client   *genai.Client
	model    string
	cacheTTL time.Duration
	logger   *slog.Logger

	cacheName     string       // Set after Configure. Empty means not configured.
	cacheNameMu   sync.Mutex   // Protects cacheName
	configureOnce sync.Once    // Ensures Configure runs only once
	configureErr  error        // Stores error from Configure
	closedMu      sync.RWMutex // Protects closed field
	closed        bool
}

// NewGeminiAgent creates a new GeminiAgent with Vertex AI backend.
// projectID, region: GCP credentials
// model: Gemini model name
// cacheTTL: TTL for the cached system prompt
// logger: Structured logger (if nil, a discard logger is created)
func NewGeminiAgent(ctx context.Context, projectID, region, model string, cacheTTL time.Duration, logger *slog.Logger) (Agent, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Normalize and validate inputs
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

	// Handle nil logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
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

	return &GeminiAgent{
		client:   client,
		model:    model,
		cacheTTL: cacheTTL,
		logger:   logger,
	}, nil
}

// Configure sets up the system prompt and creates cache.
// Must be called before GenerateText.
// Configure is idempotent - subsequent calls return the same result as the first call.
func (g *GeminiAgent) Configure(ctx context.Context, systemPrompt string) error {
	if err := g.checkClosed(); err != nil {
		return err
	}

	// Validate input
	if strings.TrimSpace(systemPrompt) == "" {
		return errors.New("systemPrompt is required")
	}

	g.configureOnce.Do(func() {
		g.logger.Debug("creating cache",
			slog.String("model", g.model),
			slog.Int("systemPromptLength", len(systemPrompt)),
			slog.Duration("ttl", g.cacheTTL),
		)

		cache, err := g.client.Caches.Create(ctx, g.model, &genai.CreateCachedContentConfig{
			TTL:               g.cacheTTL,
			SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
			DisplayName:       "yuruppu-system-prompt",
		})
		if err != nil {
			g.logger.Error("cache creation failed",
				slog.String("model", g.model),
				slog.Any("error", err),
			)
			g.configureErr = fmt.Errorf("failed to create cache: %w", err)
			return
		}

		g.cacheNameMu.Lock()
		g.cacheName = cache.Name
		g.cacheNameMu.Unlock()

		g.logger.Info("cache created successfully",
			slog.String("cacheName", cache.Name),
		)
	})

	return g.configureErr
}

// GenerateText generates a text response for the conversation history.
// The last message in history must be the user message to respond to.
// Returns NotConfiguredError if Configure has not been called.
func (g *GeminiAgent) GenerateText(ctx context.Context, history []Message) (string, error) {
	if err := g.checkClosed(); err != nil {
		return "", err
	}

	// Validate input
	if len(history) == 0 {
		return "", errors.New("history is required")
	}
	lastMsg := history[len(history)-1]
	if lastMsg.Role != "user" {
		return "", errors.New("last message in history must be from user")
	}

	g.cacheNameMu.Lock()
	cacheName := g.cacheName
	g.cacheNameMu.Unlock()

	if cacheName == "" {
		return "", &NotConfiguredError{Message: "agent is not configured: call Configure first"}
	}

	g.logger.Debug("generating text with cache",
		slog.String("model", g.model),
		slog.Int("historyLength", len(history)),
		slog.String("cacheName", cacheName),
	)

	contents := g.buildContentsFromHistory(history)

	budget := disabledThinkingBudget
	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: &budget,
		},
		CachedContent: cacheName,
	})
	if err != nil {
		g.logger.Error("LLM API call failed",
			slog.String("model", g.model),
			slog.String("cacheName", cacheName),
			slog.Any("error", err),
		)
		return "", mapAPIError(err)
	}

	return g.extractTextFromResponse(resp)
}

// Close releases any resources held by the agent.
// Deletes the cache if it was created.
func (g *GeminiAgent) Close(ctx context.Context) error {
	g.closedMu.Lock()
	if g.closed {
		g.closedMu.Unlock()
		return nil
	}
	g.closed = true
	g.closedMu.Unlock()

	// Get cache name
	g.cacheNameMu.Lock()
	cacheName := g.cacheName
	g.cacheNameMu.Unlock()

	// Delete cache if it exists
	if cacheName != "" {
		_, err := g.client.Caches.Delete(ctx, cacheName, nil)
		if err != nil {
			g.logger.Warn("cache deletion failed during close",
				slog.String("cacheName", cacheName),
				slog.Any("error", err),
			)
		} else {
			g.logger.Debug("cache deleted during close",
				slog.String("cacheName", cacheName),
			)
		}
	}

	g.logger.Debug("agent closed", slog.String("model", g.model))
	return nil
}

// checkClosed checks if the agent is closed and returns an error if so.
func (g *GeminiAgent) checkClosed() error {
	g.closedMu.RLock()
	defer g.closedMu.RUnlock()
	if g.closed {
		return &ClosedError{Message: "agent is closed"}
	}
	return nil
}

// buildContentsFromHistory builds the conversation contents from history.
func (g *GeminiAgent) buildContentsFromHistory(history []Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history))

	for _, msg := range history {
		role := msg.Role
		// Gemini uses "model" for assistant role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: msg.Content}},
		})
	}

	return contents
}

// extractTextFromResponse extracts text from LLM response.
func (g *GeminiAgent) extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "no candidates in response"),
		)
		return "", &ResponseError{Message: "no candidates in response"}
	}

	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "no content parts in response"),
		)
		return "", &ResponseError{Message: "no content parts in response"}
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part == nil {
		g.logger.Error("LLM response error",
			slog.String("reason", "response part is nil"),
		)
		return "", &ResponseError{Message: "response part is nil"}
	}

	text := part.Text
	if text == "" {
		g.logger.Error("LLM response error",
			slog.String("reason", "response part has no text"),
		)
		return "", &ResponseError{Message: "response part has no text"}
	}

	g.logger.Info("text generated successfully",
		slog.String("model", g.model),
		slog.String("modelVersion", resp.ModelVersion),
		slog.Int("responseLength", len(text)),
	)

	return text, nil
}
