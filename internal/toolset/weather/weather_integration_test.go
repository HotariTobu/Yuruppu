//go:build integration

package weather_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yuruppu/internal/toolset/weather"
)

func TestTool_Integration_Callback_Tokyo(t *testing.T) {
	tool := weather.NewTool(30 * time.Second)
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
	tool := weather.NewTool(30 * time.Second)
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
	tool := weather.NewTool(1 * time.Nanosecond)
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Contains(t, result["error"], "API request failed")
}
