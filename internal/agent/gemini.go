package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
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

	closed             atomic.Bool
	cancelCacheRefresh context.CancelFunc
	cacheName          atomic.Value // string
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
	if cfg.CacheTTL <= 0 {
		return nil, errors.New("cacheTTL must be positive")
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
			ThinkingConfig: thinkingConfig,
		},
		contentConfigWithoutCache: &genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			ThinkingConfig:    thinkingConfig,
		},
	}

	if tokenCount < minCacheTokens {
		logger.Debug("cache skipped: token count below minimum")
	} else {
		refreshCtx, cancelRefresh := context.WithCancel(context.Background())
		agent.cancelCacheRefresh = cancelRefresh

		cachedContentConfig := &genai.CreateCachedContentConfig{
			DisplayName:       cacheDisplayName,
			TTL:               cfg.CacheTTL,
			SystemInstruction: systemInstruction,
		}
		go agent.refreshCache(refreshCtx, cachedContentConfig)
	}

	return agent, nil
}

// Generate generates a response for the conversation history and user message.
func (g *GeminiAgent) Generate(ctx context.Context, history []Message, userMessage UserMessage) (AssistantMessage, error) {
	if g.closed.Load() {
		return AssistantMessage{}, &ClosedError{Message: "agent is closed"}
	}

	g.logger.Debug("generating text",
		slog.String("model", g.model),
		slog.Int("historyLength", len(history)),
	)

	contents := g.buildContents(history, userMessage)

	var config *genai.GenerateContentConfig
	cacheName, _ := g.cacheName.Load().(string)
	if cacheName == "" {
		config = g.contentConfigWithoutCache
	} else {
		configCopy := *g.contentConfigWithCache
		configCopy.CachedContent = cacheName
		config = &configCopy
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, config)
	if err != nil {
		return AssistantMessage{}, mapAPIError(err)
	}

	return g.extractResponseToAssistantMessage(resp)
}

// Close releases any resources held by the agent.
func (g *GeminiAgent) Close(ctx context.Context) error {
	if !g.closed.CompareAndSwap(false, true) {
		return nil
	}
	if g.cancelCacheRefresh != nil {
		g.cancelCacheRefresh()
	}

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
			g.cacheName.Store(cache.Name)
			g.logger.Debug("cache created", slog.String("cacheName", cache.Name))
		} else {
			g.logger.Warn("cache creation failed", slog.Any("error", err))
		}
	}

	updateCache := func(name string) {
		_, err := g.client.Caches.Update(ctx, name, &genai.UpdateCachedContentConfig{
			TTL: cfg.TTL,
		})
		if err == nil {
			g.logger.Debug("cache refreshed")
		} else {
			g.cacheName.Store("")
			g.logger.Warn("cache refresh failed", slog.Any("error", err))
		}
	}

	for {
		cacheName, _ := g.cacheName.Load().(string)
		if cacheName == "" {
			createCache()
		} else {
			updateCache(cacheName)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// buildContents builds the conversation contents from history and user message.
func (g *GeminiAgent) buildContents(history []Message, userMessage UserMessage) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history)+1)

	for _, msg := range history {
		switch m := msg.(type) {
		case *UserMessage:
			parts := g.buildUserParts(m.Parts)
			contents = append(contents, &genai.Content{
				Role:  "user",
				Parts: parts,
			})
		case *AssistantMessage:
			parts := g.buildAssistantParts(m.Parts)
			contents = append(contents, &genai.Content{
				Role:  "model",
				Parts: parts,
			})
		}
	}

	// Append current user message
	userParts := g.buildUserParts(userMessage.Parts)
	contents = append(contents, &genai.Content{
		Role:  "user",
		Parts: userParts,
	})

	return contents
}

// buildUserParts converts UserParts to Gemini Parts.
func (g *GeminiAgent) buildUserParts(parts []UserPart) []*genai.Part {
	result := make([]*genai.Part, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case *UserTextPart:
			result = append(result, genai.NewPartFromText(v.Text))
		case *UserFileDataPart:
			part := genai.NewPartFromURI(v.FileURI, v.MIMEType)
			part.FileData.DisplayName = v.DisplayName
			if v.VideoMetadata != nil {
				part.VideoMetadata = &genai.VideoMetadata{
					StartOffset: v.VideoMetadata.StartOffset,
					EndOffset:   v.VideoMetadata.EndOffset,
					FPS:         v.VideoMetadata.FPS,
				}
			}
			result = append(result, part)
		}
	}
	return result
}

// buildAssistantParts converts AssistantParts to Gemini Parts.
func (g *GeminiAgent) buildAssistantParts(parts []AssistantPart) []*genai.Part {
	result := make([]*genai.Part, 0, len(parts))
	for _, p := range parts {
		switch v := p.(type) {
		case *AssistantTextPart:
			part := genai.NewPartFromText(v.Text)
			part.Thought = v.Thought
			if v.ThoughtSignature != "" {
				part.ThoughtSignature = []byte(v.ThoughtSignature)
			}
			result = append(result, part)
		case *AssistantFileDataPart:
			part := genai.NewPartFromURI(v.FileURI, v.MIMEType)
			part.FileData.DisplayName = v.DisplayName
			result = append(result, part)
		}
	}
	return result
}

// extractResponseToAssistantMessage converts LLM response to AssistantMessage.
func (g *GeminiAgent) extractResponseToAssistantMessage(resp *genai.GenerateContentResponse) (AssistantMessage, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "no candidates in response"),
		)
		return AssistantMessage{}, &ResponseError{Message: "no candidates in response"}
	}

	content := resp.Candidates[0].Content

	if content == nil || len(content.Parts) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "no content parts in response"),
		)
		return AssistantMessage{}, &ResponseError{Message: "no content parts in response"}
	}

	// Convert genai.Part to AssistantPart
	parts := make([]AssistantPart, 0, len(content.Parts))
	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			parts = append(parts, &AssistantTextPart{
				Text:             part.Text,
				Thought:          part.Thought,
				ThoughtSignature: string(part.ThoughtSignature),
			})
		} else if part.FileData != nil {
			parts = append(parts, &AssistantFileDataPart{
				FileURI:     part.FileData.FileURI,
				MIMEType:    part.FileData.MIMEType,
				DisplayName: part.FileData.DisplayName,
			})
		}
	}

	if len(parts) == 0 {
		g.logger.Error("LLM response error",
			slog.String("reason", "response has no valid parts"),
		)
		return AssistantMessage{}, &ResponseError{Message: "response has no valid parts"}
	}

	g.logger.Info("response generated successfully",
		slog.String("model", g.model),
		slog.String("modelVersion", resp.ModelVersion),
		slog.Int("partsCount", len(parts)),
	)

	return AssistantMessage{
		ModelName: resp.ModelVersion,
		Parts:     parts,
		LocalTime: time.Now().Format(time.RFC3339),
	}, nil
}
