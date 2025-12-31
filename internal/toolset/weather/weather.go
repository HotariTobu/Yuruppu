// Package weather provides a weather tool using wttr.in API.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/genai"
)

const (
	wttrAPIURL     = "https://wttr.in/%s?format=j1"
	defaultTimeout = 3 * time.Second // NFR-001: 3 second timeout
)

// Tool implements agent.Tool for weather forecasts using wttr.in.
type Tool struct {
	client *http.Client
}

// New creates a new weather tool.
func New() *Tool {
	return &Tool{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "get_weather"
}

// Declaration returns the tool's function declaration for the LLM.
func (t *Tool) Declaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name: "get_weather",
		//nolint:gosmopolitan // Japanese examples are intentional for this LINE bot
		Description: "Get current weather and forecast for a specified location. Supports Japanese city names like '東京' or 'Tokyo'.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"location": {
					Type: genai.TypeString,
					//nolint:gosmopolitan // Japanese examples are intentional for this LINE bot
					Description: "City name (e.g., 'Tokyo', '東京', 'Osaka', '大阪')",
				},
			},
			Required: []string{"location"},
		},
	}
}

// Execute fetches weather data for the specified location.
// Returns weather info as a string, or error info as a string per NFR-002.
func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
	location, ok := args["location"].(string)
	if !ok || location == "" {
		return "Error: location parameter is required and must be a non-empty string", nil
	}

	url := fmt.Sprintf(wttrAPIURL, location)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Sprintf("Error: failed to create request: %v", err), nil
	}

	resp, err := t.client.Do(req)
	if err != nil {
		// NFR-002: Return error info as string for LLM
		return fmt.Sprintf("Error: failed to fetch weather data: %v", err), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Error: weather API returned status %d", resp.StatusCode), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: failed to read response: %v", err), nil
	}

	var data wttrResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Sprintf("Error: failed to parse weather data: %v", err), nil
	}

	return formatWeatherResponse(location, &data), nil
}

// formatWeatherResponse formats weather data for LLM consumption.
func formatWeatherResponse(location string, data *wttrResponse) string {
	if len(data.CurrentCondition) == 0 {
		return fmt.Sprintf("No weather data available for %s", location)
	}

	current := data.CurrentCondition[0]
	desc := ""
	if len(current.WeatherDesc) > 0 {
		desc = current.WeatherDesc[0].Value
	}

	return fmt.Sprintf(
		"Weather in %s:\n"+
			"Temperature: %s°C (Feels like: %s°C)\n"+
			"Condition: %s\n"+
			"Humidity: %s%%\n"+
			"Wind: %s km/h",
		location,
		current.TempC,
		current.FeelsLikeC,
		desc,
		current.Humidity,
		current.WindspeedKmph,
	)
}

// wttrResponse represents the wttr.in API response.
type wttrResponse struct {
	CurrentCondition []currentCondition `json:"current_condition"`
}

type currentCondition struct {
	TempC         string               `json:"temp_C"`
	FeelsLikeC    string               `json:"FeelsLikeC"`
	Humidity      string               `json:"humidity"`
	WindspeedKmph string               `json:"windspeedKmph"`
	WeatherDesc   []weatherDescription `json:"weatherDesc"`
}

type weatherDescription struct {
	Value string `json:"value"`
}
