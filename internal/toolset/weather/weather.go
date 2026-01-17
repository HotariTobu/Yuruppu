package weather

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
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
func NewTool(httpClient HTTPClient, logger *slog.Logger) (*Tool, error) {
	if httpClient == nil {
		return nil, errors.New("httpClient cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "get_weather"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Get weather forecast for a location. Supports current weather and up to 3-day forecasts with configurable detail levels."
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
		return nil, errors.New("invalid location")
	}

	dates := []string{"today"}
	if d, ok := args["date"].([]any); ok {
		dates = make([]string, 0, len(d))
		for _, v := range d {
			if s, ok := v.(string); ok {
				dates = append(dates, s)
			}
		}
	}

	detail := "basic"
	if d, ok := args["detail"].(string); ok {
		detail = d
	}

	hourly := false
	if h, ok := args["hourly"].(bool); ok {
		hourly = h
	}

	wttrResp, err := t.fetchWeather(ctx, location)
	if err != nil {
		return nil, err
	}

	forecasts, err := t.buildForecasts(wttrResp, dates, detail, hourly)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"location":  location,
		"forecasts": forecasts,
	}, nil
}

func (t *Tool) fetchWeather(ctx context.Context, location string) (*wttrResponse, error) {
	encodedLocation := url.PathEscape(location)
	requestURL := fmt.Sprintf(wttrURL, encodedLocation)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		t.logger.Error("failed to create request", slog.Any("error", err))
		return nil, errors.New("failed to create request")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error("API request failed", slog.Any("error", err), slog.String("location", location))
		return nil, errors.New("API request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error("API returned error status", slog.Int("status", resp.StatusCode), slog.String("location", location))
		return nil, errors.New("API returned error status")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		t.logger.Error("failed to read response", slog.Any("error", err))
		return nil, errors.New("failed to read response")
	}

	var wttrResp wttrResponse
	if err := json.Unmarshal(body, &wttrResp); err != nil {
		t.logger.Error("failed to parse response", slog.Any("error", err))
		return nil, errors.New("failed to parse response")
	}

	return &wttrResp, nil
}

func (t *Tool) buildForecasts(resp *wttrResponse, dates []string, detail string, hourly bool) ([]any, error) {
	if len(resp.Weather) == 0 {
		return nil, errors.New("no weather data available")
	}

	dateIndexMap := map[string]int{
		"today":              0,
		"tomorrow":           1,
		"day_after_tomorrow": 2,
	}

	forecasts := make([]any, 0, len(dates))
	for _, dateKey := range dates {
		idx, ok := dateIndexMap[dateKey]
		if !ok || idx >= len(resp.Weather) {
			continue
		}

		weather := resp.Weather[idx]
		forecast := t.buildForecast(resp, weather, idx, detail)

		if hourly {
			forecast["hourly"] = t.buildHourly(weather, detail)
		}

		forecasts = append(forecasts, forecast)
	}

	if len(forecasts) == 0 {
		return nil, errors.New("no forecast data for requested dates")
	}

	return forecasts, nil
}

func (t *Tool) buildForecast(resp *wttrResponse, weather wttrWeather, idx int, detail string) map[string]any {
	condition := "unknown"
	if len(weather.Hourly) > 0 && len(weather.Hourly[0].WeatherDesc) > 0 {
		condition = weather.Hourly[0].WeatherDesc[0].Value
	}

	tempC := weather.AvgTempC
	if idx == 0 && len(resp.CurrentCondition) > 0 {
		tempC = resp.CurrentCondition[0].TempC
		if len(resp.CurrentCondition[0].WeatherDesc) > 0 {
			condition = resp.CurrentCondition[0].WeatherDesc[0].Value
		}
	}

	forecast := map[string]any{
		"date":       weather.Date,
		"temp_c":     tempC,
		"condition":  condition,
		"max_temp_c": weather.MaxTempC,
		"min_temp_c": weather.MinTempC,
	}

	if detail == "detailed" || detail == "full" {
		if idx == 0 && len(resp.CurrentCondition) > 0 {
			cur := resp.CurrentCondition[0]
			forecast["humidity"] = cur.Humidity
			forecast["wind_speed_kmph"] = cur.WindspeedKmph
			forecast["wind_direction"] = cur.Winddir16Point
			forecast["feels_like_c"] = cur.FeelsLikeC
			if len(weather.Hourly) > 0 {
				forecast["rain_chance"] = weather.Hourly[0].ChanceOfRain
			}
		} else if len(weather.Hourly) > 0 {
			h := weather.Hourly[0]
			forecast["humidity"] = h.Humidity
			forecast["wind_speed_kmph"] = h.WindspeedKmph
			forecast["wind_direction"] = h.Winddir16Point
			forecast["feels_like_c"] = h.FeelsLikeC
			forecast["rain_chance"] = h.ChanceOfRain
		}
	}

	if detail == "full" {
		if idx == 0 && len(resp.CurrentCondition) > 0 {
			cur := resp.CurrentCondition[0]
			forecast["uv_index"] = cur.UVIndex
			forecast["pressure"] = cur.Pressure
			forecast["visibility"] = cur.Visibility
			forecast["cloud_cover"] = cur.CloudCover
		} else if len(weather.Hourly) > 0 {
			h := weather.Hourly[0]
			forecast["uv_index"] = h.UVIndex
			forecast["pressure"] = h.Pressure
			forecast["visibility"] = h.Visibility
			forecast["cloud_cover"] = h.CloudCover
		}
		if len(weather.Astronomy) > 0 {
			forecast["sunrise"] = weather.Astronomy[0].Sunrise
			forecast["sunset"] = weather.Astronomy[0].Sunset
		}
	}

	return forecast
}

func (t *Tool) buildHourly(weather wttrWeather, detail string) []any {
	hourlyData := make([]any, 0, len(weather.Hourly))
	for _, h := range weather.Hourly {
		condition := "unknown"
		if len(h.WeatherDesc) > 0 {
			condition = h.WeatherDesc[0].Value
		}

		entry := map[string]any{
			"time":      h.Time,
			"temp_c":    h.TempC,
			"condition": condition,
		}

		if detail == "detailed" || detail == "full" {
			entry["humidity"] = h.Humidity
			entry["wind_speed_kmph"] = h.WindspeedKmph
			entry["wind_direction"] = h.Winddir16Point
			entry["feels_like_c"] = h.FeelsLikeC
			entry["rain_chance"] = h.ChanceOfRain
		}

		if detail == "full" {
			entry["uv_index"] = h.UVIndex
			entry["pressure"] = h.Pressure
			entry["visibility"] = h.Visibility
			entry["cloud_cover"] = h.CloudCover
		}

		hourlyData = append(hourlyData, entry)
	}
	return hourlyData
}

// wttrResponse represents the wttr.in API response structure.
type wttrResponse struct {
	CurrentCondition []wttrCurrentCondition `json:"current_condition"`
	Weather          []wttrWeather          `json:"weather"`
}

type wttrCurrentCondition struct {
	TempC          string `json:"temp_C"`
	FeelsLikeC     string `json:"FeelsLikeC"`
	Humidity       string `json:"humidity"`
	WindspeedKmph  string `json:"windspeedKmph"`
	Winddir16Point string `json:"winddir16Point"`
	UVIndex        string `json:"uvIndex"`
	Pressure       string `json:"pressure"`
	Visibility     string `json:"visibility"`
	CloudCover     string `json:"cloudcover"`
	WeatherDesc    []struct {
		Value string `json:"value"`
	} `json:"weatherDesc"`
}

type wttrWeather struct {
	Date      string `json:"date"`
	MaxTempC  string `json:"maxtempC"`
	MinTempC  string `json:"mintempC"`
	AvgTempC  string `json:"avgtempC"`
	Astronomy []struct {
		Sunrise string `json:"sunrise"`
		Sunset  string `json:"sunset"`
	} `json:"astronomy"`
	Hourly []wttrHourly `json:"hourly"`
}

type wttrHourly struct {
	Time           string `json:"time"`
	TempC          string `json:"tempC"`
	FeelsLikeC     string `json:"FeelsLikeC"`
	Humidity       string `json:"humidity"`
	WindspeedKmph  string `json:"windspeedKmph"`
	Winddir16Point string `json:"winddir16Point"`
	ChanceOfRain   string `json:"chanceofrain"`
	UVIndex        string `json:"uvIndex"`
	Pressure       string `json:"pressure"`
	Visibility     string `json:"visibility"`
	CloudCover     string `json:"cloudcover"`
	WeatherDesc    []struct {
		Value string `json:"value"`
	} `json:"weatherDesc"`
}
