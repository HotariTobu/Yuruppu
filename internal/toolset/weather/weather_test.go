package weather_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"yuruppu/internal/toolset/weather"

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

func TestCallback(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		responseBody   string
		responseStatus int
		httpErr        error
		wantErr        string
		validate       func(t *testing.T, result map[string]any)
	}{
		{
			name: "success with default parameters",
			args: map[string]any{"location": "Tokyo"},
			responseBody: `{
				"current_condition":[{"temp_C":"15","weatherDesc":[{"value":"Sunny"}],"humidity":"50","windspeedKmph":"10","winddir16Point":"N","FeelsLikeC":"13"}],
				"weather":[{"date":"2026-01-02","maxtempC":"18","mintempC":"10","avgtempC":"14","hourly":[{"time":"0","tempC":"12","weatherDesc":[{"value":"Clear"}]}]}]
			}`,
			responseStatus: http.StatusOK,
			validate: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "Tokyo", result["location"])
				forecasts := result["forecasts"].([]map[string]any)
				require.Len(t, forecasts, 1)
				assert.Equal(t, "2026-01-02", forecasts[0]["date"])
				assert.Equal(t, "15", forecasts[0]["temp_c"])
				assert.Equal(t, "Sunny", forecasts[0]["condition"])
				assert.Equal(t, "18", forecasts[0]["max_temp_c"])
				assert.Equal(t, "10", forecasts[0]["min_temp_c"])
			},
		},
		{
			name: "multiple dates",
			args: map[string]any{"location": "Tokyo", "date": []any{"today", "tomorrow"}},
			responseBody: `{
				"current_condition":[{"temp_C":"15","weatherDesc":[{"value":"Sunny"}]}],
				"weather":[
					{"date":"2026-01-02","maxtempC":"18","mintempC":"10","avgtempC":"14","hourly":[{"time":"0","tempC":"12","weatherDesc":[{"value":"Clear"}]}]},
					{"date":"2026-01-03","maxtempC":"20","mintempC":"12","avgtempC":"16","hourly":[{"time":"0","tempC":"14","weatherDesc":[{"value":"Cloudy"}]}]}
				]
			}`,
			responseStatus: http.StatusOK,
			validate: func(t *testing.T, result map[string]any) {
				forecasts := result["forecasts"].([]map[string]any)
				require.Len(t, forecasts, 2)
				assert.Equal(t, "2026-01-02", forecasts[0]["date"])
				assert.Equal(t, "2026-01-03", forecasts[1]["date"])
			},
		},
		{
			name: "detailed level",
			args: map[string]any{"location": "Tokyo", "detail": "detailed"},
			responseBody: `{
				"current_condition":[{"temp_C":"15","weatherDesc":[{"value":"Sunny"}],"humidity":"50","windspeedKmph":"10","winddir16Point":"N","FeelsLikeC":"13"}],
				"weather":[{"date":"2026-01-02","maxtempC":"18","mintempC":"10","avgtempC":"14","hourly":[{"time":"0","tempC":"12","weatherDesc":[{"value":"Clear"}],"chanceofrain":"20"}]}]
			}`,
			responseStatus: http.StatusOK,
			validate: func(t *testing.T, result map[string]any) {
				forecasts := result["forecasts"].([]map[string]any)
				require.Len(t, forecasts, 1)
				assert.Equal(t, "50", forecasts[0]["humidity"])
				assert.Equal(t, "10", forecasts[0]["wind_speed_kmph"])
				assert.Equal(t, "N", forecasts[0]["wind_direction"])
				assert.Equal(t, "13", forecasts[0]["feels_like_c"])
				assert.Equal(t, "20", forecasts[0]["rain_chance"])
			},
		},
		{
			name: "hourly data",
			args: map[string]any{"location": "Tokyo", "hourly": true},
			responseBody: `{
				"current_condition":[{"temp_C":"15","weatherDesc":[{"value":"Sunny"}]}],
				"weather":[{"date":"2026-01-02","maxtempC":"18","mintempC":"10","avgtempC":"14","hourly":[
					{"time":"0","tempC":"12","weatherDesc":[{"value":"Clear"}]},
					{"time":"300","tempC":"11","weatherDesc":[{"value":"Clear"}]}
				]}]
			}`,
			responseStatus: http.StatusOK,
			validate: func(t *testing.T, result map[string]any) {
				forecasts := result["forecasts"].([]map[string]any)
				require.Len(t, forecasts, 1)
				hourly := forecasts[0]["hourly"].([]map[string]any)
				require.Len(t, hourly, 2)
				assert.Equal(t, "0", hourly[0]["time"])
				assert.Equal(t, "12", hourly[0]["temp_c"])
				assert.Equal(t, "Clear", hourly[0]["condition"])
			},
		},
		{
			name:           "HTTP error",
			args:           map[string]any{"location": "Tokyo"},
			httpErr:        errors.New("connection refused"),
			responseStatus: 0,
			wantErr:        "API request failed",
		},
		{
			name:           "error status",
			args:           map[string]any{"location": "Tokyo"},
			responseBody:   "",
			responseStatus: http.StatusNotFound,
			wantErr:        "API returned error status",
		},
		{
			name:           "invalid JSON",
			args:           map[string]any{"location": "Tokyo"},
			responseBody:   "invalid json",
			responseStatus: http.StatusOK,
			wantErr:        "failed to parse response",
		},
		{
			name:           "empty weather",
			args:           map[string]any{"location": "Tokyo"},
			responseBody:   `{"current_condition":[{"temp_C":"15"}],"weather":[]}`,
			responseStatus: http.StatusOK,
			wantErr:        "no weather data available",
		},
		{
			name:           "invalid location type",
			args:           map[string]any{"location": 123},
			responseStatus: 0,
			wantErr:        "invalid location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *mockHTTPClient
			if tt.httpErr != nil {
				client = &mockHTTPClient{err: tt.httpErr}
			} else {
				client = &mockHTTPClient{
					response: &http.Response{
						StatusCode: tt.responseStatus,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					},
				}
			}

			tool := weather.NewTool(client, slog.Default())
			result, err := tool.Callback(context.Background(), tt.args)

			require.NoError(t, err)

			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, result["error"])
				return
			}

			require.Nil(t, result["error"])
			tt.validate(t, result)
		})
	}
}
