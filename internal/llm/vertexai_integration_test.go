//go:build integration

package llm_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"yuruppu/internal/gcp"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resolveGCPCredentials resolves project ID, region, and model using MetadataClient with env var fallback.
// This mirrors the production code pattern in main.go.
func resolveGCPCredentials(t *testing.T) (projectID, region, model string) {
	t.Helper()

	// Use MetadataClient to resolve values (same pattern as main.go)
	httpClient := &http.Client{Timeout: 2 * time.Second}
	metadataClient := gcp.New(gcp.DefaultMetadataServerURL, httpClient, slog.Default())

	projectID = metadataClient.GetProjectID(os.Getenv("GCP_PROJECT_ID"))
	if projectID == "" {
		t.Fatal("GCP_PROJECT_ID could not be resolved from metadata or environment")
	}

	region = metadataClient.GetRegion(os.Getenv("GCP_REGION"))
	if region == "" {
		t.Fatal("GCP_REGION could not be resolved from metadata or environment")
	}

	model = os.Getenv("LLM_MODEL")
	if model == "" {
		t.Fatal("LLM_MODEL environment variable is required")
	}

	return projectID, region, model
}

// TestVertexAI_Integration_NewClient tests that NewVertexAIClient creates a client successfully.
func TestVertexAI_Integration_NewClient(t *testing.T) {
	projectID, region, model := resolveGCPCredentials(t)

	ctx := context.Background()

	client, err := llm.New(ctx, projectID, region, model, slog.Default())

	require.NoError(t, err, "NewVertexAIClient should succeed with valid credentials")
	assert.NotNil(t, client, "client should not be nil")
}

// TestVertexAI_Integration_GenerateText tests that GenerateText returns a response from Vertex AI.
func TestVertexAI_Integration_GenerateText(t *testing.T) {
	projectID, region, model := resolveGCPCredentials(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := llm.New(ctx, projectID, region, model, slog.Default())
	require.NoError(t, err, "NewVertexAIClient should succeed")

	response, err := client.GenerateText(ctx, "You are a helpful assistant.", "Say hello in one word.")

	require.NoError(t, err, "GenerateText should succeed")
	assert.NotEmpty(t, response, "response should not be empty")
}
