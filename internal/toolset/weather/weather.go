package weather

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

const wttrURL = "https://wttr.in/%s?format=j1"

// Tool implements the weather forecast tool using wttr.in API.
type Tool struct {
	httpClient *http.Client
}

// NewTool creates a new weather tool with the specified timeout.
func NewTool(timeout time.Duration) *Tool {
	return &Tool{
		httpClient: &http.Client{
			Timeout: timeout,
		},
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
	if !ok || location == "" {
		return map[string]any{"error": "location is required"}, nil
	}

	url := fmt.Sprintf(wttrURL, location)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to create request: %s", err.Error())}, nil
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("API request failed: %s", err.Error())}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return map[string]any{"error": fmt.Sprintf("API error: status %d", resp.StatusCode)}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to read response: %s", err.Error())}, nil
	}

	var wttrResp wttrResponse
	if err := json.Unmarshal(body, &wttrResp); err != nil {
		return map[string]any{"error": fmt.Sprintf("failed to parse response: %s", err.Error())}, nil
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
