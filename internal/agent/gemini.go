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

// minCacheTokens is the minimum token count required for Gemini context caching.
const minCacheTokens = 1024

// GeminiConfig holds configuration for GeminiAgent.
type GeminiConfig struct {
	ProjectID        string
	Region           string
	Model            string
	SystemPrompt     string
	CacheDisplayName string
	CacheTTL         time.Duration
}

// GeminiAgent is an implementation of Agent using Google Gemini via Vertex AI.
type GeminiAgent struct {
	client                    *genai.Client
	model                     string
	contentConfigWithCache    *genai.GenerateContentConfig
	contentConfigWithoutCache *genai.GenerateContentConfig
	logger                    *slog.Logger

	closedMu    sync.RWMutex
	closed      bool
	stopRefresh chan struct{}
}

// NewGeminiAgent creates a new GeminiAgent with Vertex AI backend.
// ctx: Context for initialization.
// cfg: Configuration for the agent.
// logger: Structured logger (required, returns error if nil).
func NewGeminiAgent(ctx context.Context, cfg GeminiConfig, logger *slog.Logger) (Agent, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate logger
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	// Normalize and validate inputs
	projectID := strings.TrimSpace(cfg.ProjectID)
	region := strings.TrimSpace(cfg.Region)
	model := strings.TrimSpace(cfg.Model)
	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	cacheDisplayName := strings.TrimSpace(cfg.CacheDisplayName)

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
	if cacheDisplayName == "" {
		return nil, errors.New("cacheDisplayName is required")
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

	// Count tokens in system prompt
	systemInstruction := genai.NewContentFromText(systemPrompt, genai.RoleUser)
	tokenResp, err := client.Models.CountTokens(
		ctx,
		model,
		genai.Text(""),
		&genai.CountTokensConfig{
			SystemInstruction: systemInstruction,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count tokens: %w", err)
	}

	tokenCount := tokenResp.TotalTokens
	logger.Debug("system prompt token count",
		slog.String("model", model),
		slog.Int("tokenCount", int(tokenCount)),
		slog.Int("minCacheTokens", minCacheTokens),
	)

	budget := disabledThinkingBudget
	thinkingConfig := &genai.ThinkingConfig{
		ThinkingBudget: &budget,
	}

	agent := &GeminiAgent{
		client: client,
		model:  model,
		logger: logger,
		contentConfigWithCache: &genai.GenerateContentConfig{
			CachedContent:  "",
			ThinkingConfig: thinkingConfig,
		},
		contentConfigWithoutCache: &genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			ThinkingConfig:    thinkingConfig,
		},
		stopRefresh: make(chan struct{}),
	}

	if tokenCount < minCacheTokens {
		logger.Debug("cache skipped: token count below minimum")
	} else {
		cachedContentConfig := &genai.CreateCachedContentConfig{
			DisplayName:       cacheDisplayName,
			TTL:               cfg.CacheTTL,
			SystemInstruction: systemInstruction,
		}
		go agent.refreshCache(ctx, cachedContentConfig)
	}

	return agent, nil
}

// GenerateText generates a text response for the conversation history.
// The last message in history must be the user message to respond to.
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

	g.logger.Debug("generating text",
		slog.String("model", g.model),
		slog.Int("historyLength", len(history)),
	)

	contents := g.buildContentsFromHistory(history)

	config := g.contentConfigWithCache
	if config.CachedContent == "" {
		config = g.contentConfigWithoutCache
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, config)
	if err != nil {
		return "", mapAPIError(err)
	}

	return g.extractTextFromResponse(resp)
}

// Close releases any resources held by the agent.
func (g *GeminiAgent) Close(ctx context.Context) error {
	g.closedMu.Lock()
	if g.closed {
		g.closedMu.Unlock()
		return nil
	}
	g.closed = true
	close(g.stopRefresh)
	g.closedMu.Unlock()

	g.logger.Debug("agent closed", slog.String("model", g.model))
	return nil
}

// refreshCache periodically refreshes the cache TTL.
func (g *GeminiAgent) refreshCache(ctx context.Context, cfg *genai.CreateCachedContentConfig) {
	ticker := time.NewTicker(cfg.TTL / 2)
	defer ticker.Stop()

	createCache := func() {
		cache, err := g.client.Caches.Create(ctx, g.model, cfg)
		if err == nil {
			g.contentConfigWithCache.CachedContent = cache.Name
			g.logger.Debug("cache created", slog.String("cacheName", cache.Name))
		} else {
			g.logger.Warn("cache creation failed", slog.Any("error", err))
		}
	}

	updateCache := func(cacheName string) {
		_, err := g.client.Caches.Update(ctx, cacheName, &genai.UpdateCachedContentConfig{
			TTL: cfg.TTL,
		})
		if err == nil {
			g.logger.Debug("cache refreshed")
		} else {
			g.contentConfigWithCache.CachedContent = ""
			g.logger.Warn("cache refresh failed", slog.Any("error", err))
		}
	}

	for {
		cacheName := g.contentConfigWithCache.CachedContent
		if cacheName == "" {
			createCache()
		} else {
			updateCache(cacheName)
		}

		select {
		case <-g.stopRefresh:
			return
		case <-ticker.C:
		}
	}
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

	content := resp.Candidates[0].Content

	if content == nil || len(content.Parts) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "no content parts in response"),
		)
		return "", &ResponseError{Message: "no content parts in response"}
	}

	// Concatenate all parts
	var textBuilder strings.Builder
	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		textBuilder.WriteString(part.Text)
	}

	text := textBuilder.String()
	if text == "" {
		g.logger.Error("LLM response error",
			slog.String("reason", "response has no text"),
		)
		return "", &ResponseError{Message: "response has no text"}
	}

	g.logger.Info("text generated successfully",
		slog.String("model", g.model),
		slog.String("modelVersion", resp.ModelVersion),
		slog.Int("responseLength", len(text)),
	)

	return text, nil
}
