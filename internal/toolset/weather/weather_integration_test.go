//go:build integration

package weather_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yuruppu/internal/toolset/weather"
)

func TestTool_Integration_Callback_Tokyo(t *testing.T) {
	tool := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}
	assert.Equal(t, "Tokyo", result["location"])
	assert.NotEmpty(t, result["current_temp_c"])
	assert.NotEmpty(t, result["condition"])
}

func TestTool_Integration_Callback_LocationWithSpace(t *testing.T) {
	tool := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "New York"})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}
	assert.Equal(t, "New York", result["location"])
	assert.NotEmpty(t, result["current_temp_c"])
	assert.NotEmpty(t, result["condition"])
}

func TestTool_Integration_Callback_Timeout(t *testing.T) {
	tool := weather.NewTool(&http.Client{Timeout: 1 * time.Nanosecond}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Contains(t, result["error"], "API request failed")
}
