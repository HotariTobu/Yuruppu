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
	tool, _ := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}
	assert.Equal(t, "Tokyo", result["location"])

	forecasts, ok := result["forecasts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, forecasts)
	f0 := forecasts[0].(map[string]any)
	assert.NotEmpty(t, f0["date"])
	assert.NotEmpty(t, f0["temp_c"])
	assert.NotEmpty(t, f0["condition"])
}

func TestTool_Integration_Callback_MultipleDates(t *testing.T) {
	tool, _ := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{
		"location": "Tokyo",
		"date":     []any{"today", "tomorrow"},
	})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}

	forecasts, ok := result["forecasts"].([]any)
	require.True(t, ok)
	require.Len(t, forecasts, 2)
}

func TestTool_Integration_Callback_DetailedWithHourly(t *testing.T) {
	tool, _ := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{
		"location": "Tokyo",
		"detail":   "detailed",
		"hourly":   true,
	})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}

	forecasts, ok := result["forecasts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, forecasts)
	f0 := forecasts[0].(map[string]any)

	// Check detailed fields
	assert.NotEmpty(t, f0["humidity"])
	assert.NotEmpty(t, f0["wind_speed_kmph"])
	assert.NotEmpty(t, f0["rain_chance"])

	// Check hourly data
	hourly, ok := f0["hourly"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, hourly)
	h0 := hourly[0].(map[string]any)
	assert.NotEmpty(t, h0["time"])
	assert.NotEmpty(t, h0["temp_c"])
}

func TestTool_Integration_Callback_LocationWithSpace(t *testing.T) {
	tool, _ := weather.NewTool(&http.Client{Timeout: 30 * time.Second}, slog.Default())
	ctx := context.Background()

	result, err := tool.Callback(ctx, map[string]any{"location": "New York"})

	require.NoError(t, err)
	if errMsg, ok := result["error"]; ok {
		t.Fatalf("unexpected error: %v", errMsg)
	}
	assert.Equal(t, "New York", result["location"])

	forecasts, ok := result["forecasts"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, forecasts)
	f0 := forecasts[0].(map[string]any)
	assert.NotEmpty(t, f0["temp_c"])
	assert.NotEmpty(t, f0["condition"])
}

func TestTool_Integration_Callback_Timeout(t *testing.T) {
	tool, _ := weather.NewTool(&http.Client{Timeout: 1 * time.Nanosecond}, slog.Default())
	ctx := context.Background()

	_, err := tool.Callback(ctx, map[string]any{"location": "Tokyo"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed")
}
