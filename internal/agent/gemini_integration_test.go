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

func TestGeminiAgent_Integration_GenerateText(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	systemPrompt := "You are a helpful assistant. Respond briefly."
	a, err := agent.NewGeminiAgent(ctx, projectID, region, model, 5*time.Minute, systemPrompt, nil)
	require.NoError(t, err)
	defer a.Close(ctx)

	response, err := a.GenerateText(ctx, []agent.Message{{Role: "user", Content: "Say hello"}})
	require.NoError(t, err)
	assert.NotEmpty(t, response)
}

func TestGeminiAgent_Integration_GenerateTextWithHistory(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	systemPrompt := "You are a helpful assistant. Respond briefly."
	a, err := agent.NewGeminiAgent(ctx, projectID, region, model, 5*time.Minute, systemPrompt, nil)
	require.NoError(t, err)
	defer a.Close(ctx)

	history := []agent.Message{
		{Role: "user", Content: "My name is Taro"},
		{Role: "assistant", Content: "Nice to meet you, Taro!"},
		{Role: "user", Content: "What is my name?"},
	}

	response, err := a.GenerateText(ctx, history)
	require.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "Taro")
}

func TestGeminiAgent_Integration_GenerateTextWithCache(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	// Create a system prompt with 1024+ tokens to trigger caching
	basePrompt := "You are a helpful assistant. "
	systemPrompt := basePrompt + strings.Repeat("This is additional context to increase the token count. ", 200)

	// Capture logs to verify cache creation
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	a, err := agent.NewGeminiAgent(ctx, projectID, region, model, 5*time.Minute, systemPrompt, logger)
	require.NoError(t, err)
	defer a.Close(ctx)

	// Verify cache was created
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "cache created successfully")

	response, err := a.GenerateText(ctx, []agent.Message{{Role: "user", Content: "Say hello"}})
	require.NoError(t, err)
	assert.NotEmpty(t, response)
}
