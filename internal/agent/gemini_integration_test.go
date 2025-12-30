//go:build integration

package agent_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yuruppu/internal/agent"
)

func requireGCPCredentials(t *testing.T) (projectID, region, model string) {
	t.Helper()
	projectID = os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Fatal("GCP_PROJECT_ID environment variable is not set")
	}
	region = os.Getenv("GCP_REGION")
	if region == "" {
		t.Fatal("GCP_REGION environment variable is not set")
	}
	model = os.Getenv("LLM_MODEL")
	if model == "" {
		t.Fatal("LLM_MODEL environment variable is not set")
	}
	return projectID, region, model
}

func TestGeminiAgent_Integration_Generate(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	cfg := agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            model,
		CacheTTL:         5 * time.Minute,
		CacheDisplayName: "test-cache",
		SystemPrompt:     "You are a helpful assistant. Respond briefly.",
	}
	logger := slog.New(slog.DiscardHandler)
	a, err := agent.NewGeminiAgent(ctx, cfg, logger)
	require.NoError(t, err)
	defer a.Close(ctx)

	userMessage := &agent.UserMessage{Parts: []agent.UserPart{&agent.UserTextPart{Text: "Say hello"}}}
	response, err := a.Generate(ctx, nil, userMessage)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Parts)
}

func TestGeminiAgent_Integration_GenerateWithHistory(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	cfg := agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            model,
		CacheTTL:         5 * time.Minute,
		CacheDisplayName: "test-cache-history",
		SystemPrompt:     "You are a helpful assistant. Respond briefly.",
	}
	logger := slog.New(slog.DiscardHandler)
	a, err := agent.NewGeminiAgent(ctx, cfg, logger)
	require.NoError(t, err)
	defer a.Close(ctx)

	history := []agent.Message{
		&agent.UserMessage{Parts: []agent.UserPart{&agent.UserTextPart{Text: "My name is Taro"}}},
		&agent.AssistantMessage{Parts: []agent.AssistantPart{&agent.AssistantTextPart{Text: "Nice to meet you, Taro!"}}},
	}
	userMessage := &agent.UserMessage{Parts: []agent.UserPart{&agent.UserTextPart{Text: "What is my name?"}}}

	response, err := a.Generate(ctx, history, userMessage)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Parts)

	// Check response contains "Taro"
	var responseText string
	for _, part := range response.Parts {
		if textPart, ok := part.(*agent.AssistantTextPart); ok {
			responseText += textPart.Text
		}
	}
	assert.Contains(t, responseText, "Taro")
}

func TestGeminiAgent_Integration_GenerateWithCache(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	// Create a system prompt with 1024+ tokens to trigger caching
	basePrompt := "You are a helpful assistant. "
	systemPrompt := basePrompt + strings.Repeat("This is additional context to increase the token count. ", 200)

	// Capture logs to verify cache creation
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := agent.GeminiConfig{
		ProjectID:        projectID,
		Region:           region,
		Model:            model,
		CacheTTL:         5 * time.Minute,
		CacheDisplayName: "test-cache-large",
		SystemPrompt:     systemPrompt,
	}
	a, err := agent.NewGeminiAgent(ctx, cfg, logger)
	require.NoError(t, err)
	defer a.Close(ctx)

	// Wait for async cache creation
	time.Sleep(2 * time.Second)

	// Verify cache was created
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "cache created")

	userMessage := &agent.UserMessage{Parts: []agent.UserPart{&agent.UserTextPart{Text: "Say hello"}}}
	response, err := a.Generate(ctx, nil, userMessage)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Parts)
}
