//go:build integration

package llm_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireGCPCredentials fails the test if required GCP credentials are not available.
// AC-002 (fix-integration-test-issues): Fail instead of skip when credentials are missing.
func requireGCPCredentials(t *testing.T) (projectID, region string) {
	t.Helper()
	projectID = os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Fatal("GCP_PROJECT_ID environment variable is not set")
	}
	region = os.Getenv("GCP_REGION")
	if region == "" {
		t.Fatal("GCP_REGION environment variable is not set")
	}
	return projectID, region
}

// TestVertexAI_Integration_NewClient tests that NewVertexAIClient creates a client successfully.
func TestVertexAI_Integration_NewClient(t *testing.T) {
	projectID, region := requireGCPCredentials(t)

	ctx := context.Background()

	client, err := llm.NewVertexAIClient(ctx, projectID, region, slog.Default())

	require.NoError(t, err, "NewVertexAIClient should succeed with valid credentials")
	assert.NotNil(t, client, "client should not be nil")
}

// TestVertexAI_Integration_GenerateText tests that GenerateText returns a response from Vertex AI.
func TestVertexAI_Integration_GenerateText(t *testing.T) {
	projectID, region := requireGCPCredentials(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := llm.NewVertexAIClient(ctx, projectID, region, slog.Default())
	require.NoError(t, err, "NewVertexAIClient should succeed")

	response, err := client.GenerateText(ctx, "You are a helpful assistant.", "Say hello in one word.")

	require.NoError(t, err, "GenerateText should succeed")
	assert.NotEmpty(t, response, "response should not be empty")
}
