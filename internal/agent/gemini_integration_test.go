//go:build integration

package agent_test

import (
	"context"
	"os"
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

	a, err := agent.NewGeminiAgent(ctx, projectID, region, model, 5*time.Minute, nil)
	require.NoError(t, err)
	defer a.Close(ctx)

	err = a.Configure(ctx, "You are a helpful assistant. Respond briefly.")
	require.NoError(t, err)

	response, err := a.GenerateText(ctx, []agent.Message{{Role: "user", Content: "Say hello"}})
	require.NoError(t, err)
	assert.NotEmpty(t, response)
}

func TestGeminiAgent_Integration_GenerateTextWithHistory(t *testing.T) {
	projectID, region, model := requireGCPCredentials(t)
	ctx := context.Background()

	a, err := agent.NewGeminiAgent(ctx, projectID, region, model, 5*time.Minute, nil)
	require.NoError(t, err)
	defer a.Close(ctx)

	err = a.Configure(ctx, "You are a helpful assistant. Respond briefly.")
	require.NoError(t, err)

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
