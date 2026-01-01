package weather

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

const (
	wttrURL         = "https://wttr.in/%s?format=j1"
	maxResponseSize = 1 << 20 // 1MB
)

// HTTPClient is an interface for HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Tool implements the weather forecast tool using wttr.in API.
type Tool struct {
	httpClient HTTPClient
	logger     *slog.Logger
}

// NewTool creates a new weather tool with the specified HTTP client and logger.
func NewTool(httpClient HTTPClient, logger *slog.Logger) *Tool {
	return &Tool{
		httpClient: httpClient,
		logger:     logger,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "get_weather"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Get current weather forecast for a location. Supports city names in English or Japanese (e.g., Tokyo, Osaka)."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback fetches weather data for the specified location.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	location, ok := args["location"].(string)
	if !ok {
		return map[string]any{"error": "invalid location"}, nil
	}

	encodedLocation := url.PathEscape(location)
	requestURL := fmt.Sprintf(wttrURL, encodedLocation)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		t.logger.Error("failed to create request", slog.Any("error", err))
		return map[string]any{"error": "failed to create request"}, nil
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error("API request failed", slog.Any("error", err), slog.String("location", location))
		return map[string]any{"error": "API request failed"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error("API returned error status", slog.Int("status", resp.StatusCode), slog.String("location", location))
		return map[string]any{"error": "API returned error status"}, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		t.logger.Error("failed to read response", slog.Any("error", err))
		return map[string]any{"error": "failed to read response"}, nil
	}

	var wttrResp wttrResponse
	if err := json.Unmarshal(body, &wttrResp); err != nil {
		t.logger.Error("failed to parse response", slog.Any("error", err))
		return map[string]any{"error": "failed to parse response"}, nil
	}

	if len(wttrResp.CurrentCondition) == 0 {
		return map[string]any{"error": "no weather data available"}, nil
	}

	current := wttrResp.CurrentCondition[0]
	condition := "unknown"
	if len(current.WeatherDesc) > 0 {
		condition = current.WeatherDesc[0].Value
	}

	return map[string]any{
		"location":       location,
		"current_temp_c": current.TempC,
		"condition":      condition,
	}, nil
}

// wttrResponse represents the wttr.in API response structure.
type wttrResponse struct {
	CurrentCondition []struct {
		TempC       string `json:"temp_C"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`
}
