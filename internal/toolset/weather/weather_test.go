package weather

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestCallback_Success(t *testing.T) {
	body := `{"current_condition":[{"temp_C":"15","weatherDesc":[{"value":"Sunny"}]}]}`
	client := &mockHTTPClient{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		},
	}
	tool := NewTool(client, slog.Default())

	result, err := tool.Callback(context.Background(), map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Equal(t, "Tokyo", result["location"])
	assert.Equal(t, "15", result["current_temp_c"])
	assert.Equal(t, "Sunny", result["condition"])
}

func TestCallback_HTTPError(t *testing.T) {
	client := &mockHTTPClient{
		err: errors.New("connection refused"),
	}
	tool := NewTool(client, slog.Default())

	result, err := tool.Callback(context.Background(), map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Equal(t, "API request failed", result["error"])
}

func TestCallback_ErrorStatus(t *testing.T) {
	client := &mockHTTPClient{
		response: &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString("")),
		},
	}
	tool := NewTool(client, slog.Default())

	result, err := tool.Callback(context.Background(), map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Equal(t, "API returned error status", result["error"])
}

func TestCallback_InvalidJSON(t *testing.T) {
	client := &mockHTTPClient{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("invalid json")),
		},
	}
	tool := NewTool(client, slog.Default())

	result, err := tool.Callback(context.Background(), map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Equal(t, "failed to parse response", result["error"])
}

func TestCallback_EmptyCurrentCondition(t *testing.T) {
	body := `{"current_condition":[]}`
	client := &mockHTTPClient{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		},
	}
	tool := NewTool(client, slog.Default())

	result, err := tool.Callback(context.Background(), map[string]any{"location": "Tokyo"})

	require.NoError(t, err)
	assert.Equal(t, "no weather data available", result["error"])
}
