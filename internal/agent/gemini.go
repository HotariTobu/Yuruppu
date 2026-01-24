package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/genai"
)

// minCacheTokens is the minimum token count required for Gemini context caching.
const minCacheTokens = 1024

// GeminiConfig holds configuration for GeminiAgent.
type GeminiConfig struct {
	ProjectID        string
	Region           string
	Model            string
	SystemPrompt     string
	Tools            []Tool
	FunctionCallOnly bool
	CacheDisplayName string
	CacheTTL         time.Duration
}

// GeminiAgent is an implementation of Agent using Google Gemini via Vertex AI.
type GeminiAgent struct {
	client                    *genai.Client
	model                     string
	contentConfigWithCache    *genai.GenerateContentConfig
	contentConfigWithoutCache *genai.GenerateContentConfig
	toolMap                   map[string]tool
	logger                    *slog.Logger

	closed             atomic.Bool
	cancelCacheRefresh context.CancelFunc
	cacheName          atomic.Value // string
}

// NewGeminiAgent creates a new GeminiAgent with Vertex AI backend.
// ctx: Context for initialization.
// cfg: Configuration for the agent.
// logger: Structured logger (required, returns error if nil).
func NewGeminiAgent(ctx context.Context, cfg GeminiConfig, logger *slog.Logger) (*GeminiAgent, error) {
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

	var genaiTools []*genai.Tool
	var toolConfig *genai.ToolConfig
	var toolMap map[string]tool
	if len(cfg.Tools) > 0 {
		genaiTools = []*genai.Tool{toGenaiTool(cfg.Tools)}
		toolMap = make(map[string]tool, len(cfg.Tools))
		for _, t := range cfg.Tools {
			wrapped, err := newTool(t)
			if err != nil {
				return nil, fmt.Errorf("failed to create tool %s: %w", t.Name(), err)
			}
			toolMap[t.Name()] = wrapped
		}
		if cfg.FunctionCallOnly {
			toolConfig = &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode: genai.FunctionCallingConfigModeAny,
				},
			}
		}
	}

	agent := &GeminiAgent{
		client: client,
		model:  model,
		// Do not duplicate fields already set in cachedContentConfig.
		// Duplicating them will cause an error.
		contentConfigWithCache: &genai.GenerateContentConfig{},
		contentConfigWithoutCache: &genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
			Tools:             genaiTools,
			ToolConfig:        toolConfig,
		},
		toolMap: toolMap,
		logger:  logger,
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
			Tools:             genaiTools,
			ToolConfig:        toolConfig,
		}
		go agent.refreshCache(refreshCtx, cachedContentConfig)
	}

	return agent, nil
}

// Generate generates a response for the conversation history.
// The last message in history must be the user message to respond to.
func (g *GeminiAgent) Generate(ctx context.Context, history []Message) (*AssistantMessage, error) {
	if g.closed.Load() {
		return nil, errors.New("agent is closed")
	}

	g.logger.Debug("generating text",
		slog.String("model", g.model),
		slog.Int("historyLength", len(history)),
	)

	contents := g.buildContents(history)

	var config *genai.GenerateContentConfig
	cacheName, _ := g.cacheName.Load().(string)
	if cacheName == "" {
		config = g.contentConfigWithoutCache
	} else {
		configCopy := *g.contentConfigWithCache
		configCopy.CachedContent = cacheName
		config = &configCopy
	}

	addedContents, err := g.generateWithToolLoop(ctx, g.model, contents, config)
	if err != nil {
		return nil, err
	}

	parts := g.extractAssistantParts(addedContents)

	g.logger.Info("response generated successfully",
		slog.String("model", g.model),
		slog.Int("partsCount", len(parts)),
	)

	return &AssistantMessage{
		Parts: parts,
	}, nil
}

// generateWithToolLoop handles multi-turn conversation with tool calling.
// Returns all contents added after initialContents.
func (g *GeminiAgent) generateWithToolLoop(ctx context.Context, model string, initialContents []*genai.Content, config *genai.GenerateContentConfig) ([]*genai.Content, error) {
	var addedContents []*genai.Content

	for {
		allContents := slices.Concat(initialContents, addedContents)
		resp, err := g.client.Models.GenerateContent(ctx, model, allContents, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate content: %w", err)
		}

		// Append model's response
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			addedContents = append(addedContents, resp.Candidates[0].Content)
		}

		// Check for function calls
		functionCalls := resp.FunctionCalls()
		if len(functionCalls) == 0 {
			return addedContents, nil
		}

		// Execute all function calls in parallel
		toolCtx := WithModelName(ctx, resp.ModelVersion)
		funcResps := make([]*genai.FunctionResponse, len(functionCalls))
		finals := make([]bool, len(functionCalls))
		var wg sync.WaitGroup
		for i, call := range functionCalls {
			wg.Add(1)
			go func(i int, call *genai.FunctionCall) {
				defer wg.Done()
				funcResps[i], finals[i] = g.executeTool(toolCtx, call)
			}(i, call)
		}
		wg.Wait()

		// Combine all function responses into a single Content
		funcRespParts := make([]*genai.Part, len(funcResps))
		for i, funcResp := range funcResps {
			g.logger.Debug("tool executed",
				slog.String("tool", funcResp.Name),
				slog.Any("args", functionCalls[i].Args),
				slog.Any("response", funcResp.Response),
				slog.Bool("final", finals[i]),
			)
			funcRespParts[i] = genai.NewPartFromFunctionResponse(funcResp.Name, funcResp.Response)
		}
		addedContents = append(addedContents, genai.NewContentFromParts(funcRespParts, genai.RoleUser))

		if slices.Contains(finals, true) {
			return addedContents, nil
		}
	}
}

// executeTool executes a tool and returns the function response.
func (g *GeminiAgent) executeTool(ctx context.Context, call *genai.FunctionCall) (*genai.FunctionResponse, bool) {
	resp := &genai.FunctionResponse{
		Name: call.Name,
		ID:   call.ID,
	}

	t, ok := g.toolMap[call.Name]
	if !ok {
		resp.Response = map[string]any{"error": fmt.Sprintf("unknown tool: %s", call.Name)}
		return resp, false
	}

	result, err := t.Use(ctx, call.Args)
	if err != nil {
		resp.Response = map[string]any{"error": err.Error()}
		return resp, false
	}

	resp.Response = result.Response
	return resp, result.Final
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

// buildContents builds the conversation contents from history.
// The last message in history must be the user message to respond to.
func (g *GeminiAgent) buildContents(history []Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history))

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

// extractAssistantParts extracts all model response parts from contents.
func (g *GeminiAgent) extractAssistantParts(contents []*genai.Content) []AssistantPart {
	var parts []AssistantPart

	for _, content := range contents {
		if content == nil || content.Role != "model" {
			continue
		}
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
	}

	return parts
}

// toGenaiTool converts Tool[] to a single *genai.Tool.
// All function declarations are combined into one genai.Tool because
// Gemini only allows multiple tools when they are all search tools.
func toGenaiTool(tools []Tool) *genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	funcDecls := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, t := range tools {
		var paramsSchema any
		if err := json.Unmarshal(t.ParametersJsonSchema(), &paramsSchema); err != nil {
			continue
		}

		var respSchema any
		if err := json.Unmarshal(t.ResponseJsonSchema(), &respSchema); err != nil {
			continue
		}

		funcDecls = append(funcDecls, &genai.FunctionDeclaration{
			Name:                 t.Name(),
			Description:          t.Description(),
			ParametersJsonSchema: paramsSchema,
			ResponseJsonSchema:   respSchema,
		})
	}

	if len(funcDecls) == 0 {
		return nil
	}

	return &genai.Tool{FunctionDeclarations: funcDecls}
}
