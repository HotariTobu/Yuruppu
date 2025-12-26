//go:build integration

package llm_test

import (
	"context"
	"os"
	"testing"
	"time"

	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfMissingCredentials skips the test if required GCP credentials are not available.
// AC-003: Integration tests skip without credentials with descriptive message.
func skipIfMissingCredentials(t *testing.T) {
	t.Helper()
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("Skipping integration test: GCP_PROJECT_ID environment variable is not set")
	}
}

// TestVertexAI_Integration_NewClient tests that NewVertexAIClient creates a client successfully.
// AC-002: NewVertexAIClient() creates client successfully with valid credentials.
func TestVertexAI_Integration_NewClient(t *testing.T) {
	skipIfMissingCredentials(t)

	projectID := os.Getenv("GCP_PROJECT_ID")
	region := os.Getenv("GCP_REGION")
	if region == "" {
		region = "us-central1"
	}

	ctx := context.Background()

	client, err := llm.NewVertexAIClient(ctx, projectID, region)

	require.NoError(t, err, "NewVertexAIClient should succeed with valid credentials")
	assert.NotNil(t, client, "client should not be nil")
}

// TestVertexAI_Integration_GenerateText tests that GenerateText returns a response from Vertex AI.
// AC-002: GenerateText() returns response from Vertex AI.
func TestVertexAI_Integration_GenerateText(t *testing.T) {
	skipIfMissingCredentials(t)

	projectID := os.Getenv("GCP_PROJECT_ID")
	region := os.Getenv("GCP_REGION")
	if region == "" {
		region = "us-central1"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := llm.NewVertexAIClient(ctx, projectID, region)
	require.NoError(t, err, "NewVertexAIClient should succeed")

	response, err := client.GenerateText(ctx, "You are a helpful assistant.", "Say hello in one word.")

	require.NoError(t, err, "GenerateText should succeed")
	assert.NotEmpty(t, response, "response should not be empty")
}
