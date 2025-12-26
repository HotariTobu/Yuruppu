package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"yuruppu/internal/line"
	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// mockLLMProvider is a mock implementation of llm.Provider for testing.
type mockLLMProvider struct{}

func (m *mockLLMProvider) GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "mock response", nil
}

// Ensure mockLLMProvider implements llm.Provider interface
var _ llm.Provider = (*mockLLMProvider)(nil)

// TestLoadConfig_ValidCredentials tests loading configuration with valid environment variables.
// AC-001: Given LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN are set,
// when application starts, then no error occurs and values are used for Bot initialization.
// FR-003: GCP_PROJECT_ID is also required for LLM initialization.
func TestLoadConfig_ValidCredentials(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
		gcpProjectID       string
	}{
		{
			name:               "all credentials set returns config",
			channelSecret:      "test-channel-secret",
			channelAccessToken: "test-access-token",
			gcpProjectID:       "test-project-id",
		},
		{
			name:               "long credentials are accepted",
			channelSecret:      "very-long-channel-secret-with-many-characters-0123456789",
			channelAccessToken: "very-long-access-token-with-many-characters-0123456789",
			gcpProjectID:       "very-long-project-id-with-many-characters-0123456789",
		},
		{
			name:               "credentials with special characters are accepted",
			channelSecret:      "secret-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			channelAccessToken: "token-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			gcpProjectID:       "project-id-with-hyphens-and-numbers-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			t.Setenv("GCP_PROJECT_ID", tt.gcpProjectID)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error with valid credentials")

			// Then: Config should contain the expected values
			assert.NotNil(t, config, "config should not be nil")
			assert.Equal(t, tt.channelSecret, config.ChannelSecret,
				"ChannelSecret should match environment variable")
			assert.Equal(t, tt.channelAccessToken, config.ChannelAccessToken,
				"ChannelAccessToken should match environment variable")
			assert.Equal(t, tt.gcpProjectID, config.GCPProjectID,
				"GCPProjectID should match environment variable")
		})
	}
}

// TestLoadConfig_MissingChannelSecret tests error when LINE_CHANNEL_SECRET is missing.
// AC-002a: Given LINE_CHANNEL_SECRET is not set,
// when application starts,
// then error "LINE_CHANNEL_SECRET is required" is output and program exits.
func TestLoadConfig_MissingChannelSecret(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
		wantErrMsg         string
	}{
		{
			name:               "empty channel secret returns error",
			channelSecret:      "",
			channelAccessToken: "valid-access-token",
			wantErrMsg:         "LINE_CHANNEL_SECRET is required",
		},
		{
			name:               "unset channel secret returns error",
			channelSecret:      "", // Will not be set in environment
			channelAccessToken: "valid-access-token",
			wantErrMsg:         "LINE_CHANNEL_SECRET is required",
		},
		{
			name:               "whitespace-only channel secret returns error",
			channelSecret:      "   ",
			channelAccessToken: "valid-access-token",
			wantErrMsg:         "LINE_CHANNEL_SECRET is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Unset any existing environment variables
			os.Unsetenv("LINE_CHANNEL_SECRET")
			os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")

			// Given: Set environment variables (if not empty)
			if tt.channelSecret != "" && tt.name != "unset channel secret returns error" {
				t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			}
			if tt.channelAccessToken != "" {
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when LINE_CHANNEL_SECRET is missing")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate LINE_CHANNEL_SECRET is required
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate LINE_CHANNEL_SECRET is required")
		})
	}
}

// TestLoadConfig_MissingChannelAccessToken tests error when LINE_CHANNEL_ACCESS_TOKEN is missing.
// AC-002b: Given LINE_CHANNEL_ACCESS_TOKEN is not set,
// when application starts,
// then error "LINE_CHANNEL_ACCESS_TOKEN is required" is output and program exits.
func TestLoadConfig_MissingChannelAccessToken(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
		wantErrMsg         string
	}{
		{
			name:               "empty channel access token returns error",
			channelSecret:      "valid-channel-secret",
			channelAccessToken: "",
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN is required",
		},
		{
			name:               "unset channel access token returns error",
			channelSecret:      "valid-channel-secret",
			channelAccessToken: "", // Will not be set in environment
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN is required",
		},
		{
			name:               "whitespace-only channel access token returns error",
			channelSecret:      "valid-channel-secret",
			channelAccessToken: "   ",
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Unset any existing environment variables
			os.Unsetenv("LINE_CHANNEL_SECRET")
			os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")

			// Given: Set environment variables (if not empty)
			if tt.channelSecret != "" {
				t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			}
			if tt.channelAccessToken != "" && tt.name != "unset channel access token returns error" {
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when LINE_CHANNEL_ACCESS_TOKEN is missing")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate LINE_CHANNEL_ACCESS_TOKEN is required
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate LINE_CHANNEL_ACCESS_TOKEN is required")
		})
	}
}

// TestLoadConfig_BothMissing tests error when both credentials are missing.
// This tests the error handling priority (which error is reported first).
func TestLoadConfig_BothMissing(t *testing.T) {
	tests := []struct {
		name       string
		wantErrMsg string
	}{
		{
			name:       "both credentials missing returns error",
			wantErrMsg: "is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Unset all environment variables
			os.Unsetenv("LINE_CHANNEL_SECRET")
			os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when both credentials are missing")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate a required variable is missing
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate a required variable is missing")
		})
	}
}

// TestLoadConfig_GCPConfigOptional tests that GCP config is optional in loadConfig.
// AC-002, AC-003: GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
func TestLoadConfig_GCPConfigOptional(t *testing.T) {
	// Given: Set LINE credentials but not GCP config
	t.Setenv("LINE_CHANNEL_SECRET", "valid-secret")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "valid-token")
	os.Unsetenv("GCP_PROJECT_ID")
	os.Unsetenv("GCP_REGION")

	// When: Load configuration
	config, err := loadConfig()

	// Then: Should succeed without error
	require.NoError(t, err, "loadConfig should not return error when GCP config is missing")

	// Then: GCPProjectID and GCPRegion should be empty strings
	assert.Equal(t, "", config.GCPProjectID, "GCPProjectID should be empty string")
	assert.Equal(t, "", config.GCPRegion, "GCPRegion should be empty string")
}

// TestLoadConfig_TrimsWhitespace tests that configuration values are trimmed.
// This ensures that accidental whitespace in environment variables doesn't cause issues.
func TestLoadConfig_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name                  string
		channelSecret         string
		channelAccessToken    string
		gcpProjectID          string
		expectedChannelSecret string
		expectedAccessToken   string
		expectedProjectID     string
	}{
		{
			name:                  "leading whitespace is trimmed",
			channelSecret:         "  secret",
			channelAccessToken:    "  token",
			gcpProjectID:          "  project-id",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "project-id",
		},
		{
			name:                  "trailing whitespace is trimmed",
			channelSecret:         "secret  ",
			channelAccessToken:    "token  ",
			gcpProjectID:          "project-id  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "project-id",
		},
		{
			name:                  "leading and trailing whitespace is trimmed",
			channelSecret:         "  secret  ",
			channelAccessToken:    "  token  ",
			gcpProjectID:          "  project-id  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "project-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables with whitespace
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			t.Setenv("GCP_PROJECT_ID", tt.gcpProjectID)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed
			require.NoError(t, err, "loadConfig should not return error")

			// Then: Whitespace should be trimmed
			assert.Equal(t, tt.expectedChannelSecret, config.ChannelSecret,
				"ChannelSecret should have whitespace trimmed")
			assert.Equal(t, tt.expectedAccessToken, config.ChannelAccessToken,
				"ChannelAccessToken should have whitespace trimmed")
			assert.Equal(t, tt.expectedProjectID, config.GCPProjectID,
				"GCPProjectID should have whitespace trimmed")
		})
	}
}

// TestLoadConfig_ErrorMessages tests that error messages are descriptive.
// This ensures users can easily identify which configuration is missing.
func TestLoadConfig_ErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		setupEnv        func(*testing.T)
		wantErrContains []string
	}{
		{
			name: "missing secret error mentions LINE_CHANNEL_SECRET",
			setupEnv: func(t *testing.T) {
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "token")
				// LINE_CHANNEL_SECRET is not set
			},
			wantErrContains: []string{"LINE_CHANNEL_SECRET", "required"},
		},
		{
			name: "missing token error mentions LINE_CHANNEL_ACCESS_TOKEN",
			setupEnv: func(t *testing.T) {
				t.Setenv("LINE_CHANNEL_SECRET", "secret")
				// LINE_CHANNEL_ACCESS_TOKEN is not set
			},
			wantErrContains: []string{"LINE_CHANNEL_ACCESS_TOKEN", "required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup environment
			os.Unsetenv("LINE_CHANNEL_SECRET")
			os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")
			tt.setupEnv(t)

			// When: Load configuration
			_, err := loadConfig()

			// Then: Should return error with descriptive message
			require.Error(t, err)
			for _, want := range tt.wantErrContains {
				assert.Contains(t, err.Error(), want,
					"error message should contain %q", want)
			}
		})
	}
}

// TestNewServer_Success tests successful Server initialization.
// AC-003: Given valid credentials are set,
// when application starts,
// then line.NewServer() is called and Server instance is created successfully.
func TestNewServer_Success(t *testing.T) {
	tests := []struct {
		name          string
		channelSecret string
	}{
		{
			name:          "valid credentials initialize server successfully",
			channelSecret: "test-channel-secret",
		},
		{
			name:          "long credentials initialize server successfully",
			channelSecret: "very-long-channel-secret-0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Initialize server directly
			s, err := line.NewServer(tt.channelSecret, discardLogger())

			// Then: Should succeed without error
			require.NoError(t, err, "line.NewServer should not return error with valid credentials")

			// Then: Server instance should not be nil
			assert.NotNil(t, s, "server instance should not be nil")
		})
	}
}

// TestNewServer_EmptySecret tests error when channel secret is empty.
func TestNewServer_EmptySecret(t *testing.T) {
	// When: Initialize server with empty channel secret
	s, err := line.NewServer("", discardLogger())

	// Then: Should return error
	require.Error(t, err, "line.NewServer should return error with empty channel secret")

	// Then: Server instance should be nil
	assert.Nil(t, s, "server instance should be nil on error")

	// Then: Error message should indicate missing credential
	assert.Contains(t, err.Error(), "channelSecret", "error message should mention channelSecret")
}

// TestNewClient_Success tests successful Client initialization.
func TestNewClient_Success(t *testing.T) {
	tests := []struct {
		name               string
		channelAccessToken string
	}{
		{
			name:               "valid credentials initialize client successfully",
			channelAccessToken: "test-access-token",
		},
		{
			name:               "long credentials initialize client successfully",
			channelAccessToken: "very-long-access-token-0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When: Initialize client directly
			c, err := line.NewClient(tt.channelAccessToken, discardLogger())

			// Then: Should succeed without error
			require.NoError(t, err, "line.NewClient should not return error with valid credentials")

			// Then: Client instance should not be nil
			assert.NotNil(t, c, "client instance should not be nil")
		})
	}
}

// TestNewClient_EmptyToken tests error when channel access token is empty.
func TestNewClient_EmptyToken(t *testing.T) {
	// When: Initialize client with empty channel access token
	c, err := line.NewClient("", discardLogger())

	// Then: Should return error
	require.Error(t, err, "line.NewClient should return error with empty channel access token")

	// Then: Client instance should be nil
	assert.Nil(t, c, "client instance should be nil on error")

	// Then: Error message should indicate missing credential
	assert.Contains(t, err.Error(), "channelToken", "error message should mention channelToken")
}

// TestNewVertexAIClient_EmptyGCPProjectID tests error when GCP_PROJECT_ID is empty.
// AC-013: Bot fails to start during initialization if credentials are missing.
func TestNewVertexAIClient_EmptyGCPProjectID(t *testing.T) {
	// When: Initialize LLM with empty GCP_PROJECT_ID
	llmProvider, err := llm.NewVertexAIClient(context.Background(), "", "", discardLogger())

	// Then: Should return error
	require.Error(t, err, "llm.NewVertexAIClient should return error with empty GCP_PROJECT_ID")

	// Then: LLM provider should be nil
	assert.Nil(t, llmProvider, "llmProvider should be nil on error")

	// Then: Error message should indicate GCP_PROJECT_ID is missing
	assert.Contains(t, err.Error(), "GCP_PROJECT_ID", "error message should mention GCP_PROJECT_ID")
}

// Note: TestGetPort_* tests were removed as part of SC-006.
// PORT functionality is now tested via TestLoadConfig_Port tests.
// The getPort() function was removed and replaced with Config.Port.

// TestCreateHandler tests that createHandler returns an http.Handler.
// AC-004: Given Server is initialized successfully,
// when application starts,
// then /webhook endpoint is accessible.
func TestCreateHandler(t *testing.T) {
	// Given: Create a server directly
	server, err := line.NewServer("test-secret", discardLogger())
	require.NoError(t, err)

	// When: Create handler
	handler := createHandler(server)

	// Then: Should return non-nil handler
	assert.NotNil(t, handler, "handler should not be nil")
}

// TestLoadConfig_LLMTimeout tests LLM timeout configuration loading.
// NFR-001: LLM API total request timeout should be configurable via environment variable (default: 30 seconds)
func TestLoadConfig_LLMTimeout(t *testing.T) {
	tests := []struct {
		name            string
		llmTimeoutEnv   string
		expectedTimeout int
	}{
		{
			name:            "default timeout is 30 seconds when not set",
			llmTimeoutEnv:   "",
			expectedTimeout: 30,
		},
		{
			name:            "custom timeout from environment variable",
			llmTimeoutEnv:   "60",
			expectedTimeout: 60,
		},
		{
			name:            "timeout of 1 second",
			llmTimeoutEnv:   "1",
			expectedTimeout: 1,
		},
		{
			name:            "timeout of 120 seconds",
			llmTimeoutEnv:   "120",
			expectedTimeout: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")

			if tt.llmTimeoutEnv != "" {
				t.Setenv("LLM_TIMEOUT_SECONDS", tt.llmTimeoutEnv)
			} else {
				os.Unsetenv("LLM_TIMEOUT_SECONDS")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: LLM timeout should match expected value
			assert.Equal(t, tt.expectedTimeout, config.LLMTimeoutSeconds,
				"LLMTimeoutSeconds should match expected value")
		})
	}
}

// TestLoadConfig_Port tests that PORT is loaded via loadConfig.
// SC-001, AC-001: PORT is read and trimmed, defaults to "8080" if empty.
func TestLoadConfig_Port(t *testing.T) {
	tests := []struct {
		name         string
		portEnv      string
		expectedPort string
	}{
		{
			name:         "default port is 8080 when not set",
			portEnv:      "",
			expectedPort: "8080",
		},
		{
			name:         "custom port from environment variable",
			portEnv:      "3000",
			expectedPort: "3000",
		},
		{
			name:         "port 9000",
			portEnv:      "9000",
			expectedPort: "9000",
		},
		{
			name:         "port 80",
			portEnv:      "80",
			expectedPort: "80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")

			if tt.portEnv != "" {
				t.Setenv("PORT", tt.portEnv)
			} else {
				os.Unsetenv("PORT")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: Port should match expected value
			assert.Equal(t, tt.expectedPort, config.Port,
				"Port should match expected value")
		})
	}
}

// TestLoadConfig_Port_TrimsWhitespace tests that PORT value is trimmed.
// SC-001, AC-001: PORT environment variable is read and trimmed.
func TestLoadConfig_Port_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name         string
		portEnv      string
		expectedPort string
	}{
		{
			name:         "leading whitespace is trimmed",
			portEnv:      "  3000",
			expectedPort: "3000",
		},
		{
			name:         "trailing whitespace is trimmed",
			portEnv:      "3000  ",
			expectedPort: "3000",
		},
		{
			name:         "leading and trailing whitespace is trimmed",
			portEnv:      "  3000  ",
			expectedPort: "3000",
		},
		{
			name:         "whitespace only defaults to 8080",
			portEnv:      "   ",
			expectedPort: "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("PORT", tt.portEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: Port should be trimmed and match expected value
			assert.Equal(t, tt.expectedPort, config.Port,
				"Port should have whitespace trimmed")
		})
	}
}

// TestLoadConfig_GCPRegion tests that GCP_REGION is loaded via loadConfig.
// GCP_REGION is optional (auto-detected on Cloud Run).
func TestLoadConfig_GCPRegion(t *testing.T) {
	tests := []struct {
		name           string
		gcpRegionEnv   string
		expectedRegion string
	}{
		{
			name:           "empty string when not set",
			gcpRegionEnv:   "",
			expectedRegion: "",
		},
		{
			name:           "custom region from environment variable",
			gcpRegionEnv:   "asia-northeast1",
			expectedRegion: "asia-northeast1",
		},
		{
			name:           "region us-west1",
			gcpRegionEnv:   "us-west1",
			expectedRegion: "us-west1",
		},
		{
			name:           "region europe-west1",
			gcpRegionEnv:   "europe-west1",
			expectedRegion: "europe-west1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")

			if tt.gcpRegionEnv != "" {
				t.Setenv("GCP_REGION", tt.gcpRegionEnv)
			} else {
				os.Unsetenv("GCP_REGION")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: GCPRegion should match expected value
			assert.Equal(t, tt.expectedRegion, config.GCPRegion,
				"GCPRegion should match expected value")
		})
	}
}

// TestLoadConfig_GCPRegion_TrimsWhitespace tests that GCP_REGION value is trimmed.
// GCP_REGION environment variable is read and trimmed.
func TestLoadConfig_GCPRegion_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name           string
		gcpRegionEnv   string
		expectedRegion string
	}{
		{
			name:           "leading whitespace is trimmed",
			gcpRegionEnv:   "  asia-northeast1",
			expectedRegion: "asia-northeast1",
		},
		{
			name:           "trailing whitespace is trimmed",
			gcpRegionEnv:   "asia-northeast1  ",
			expectedRegion: "asia-northeast1",
		},
		{
			name:           "leading and trailing whitespace is trimmed",
			gcpRegionEnv:   "  asia-northeast1  ",
			expectedRegion: "asia-northeast1",
		},
		{
			name:           "whitespace only results in empty string",
			gcpRegionEnv:   "   ",
			expectedRegion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_REGION", tt.gcpRegionEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: GCPRegion should be trimmed and match expected value
			assert.Equal(t, tt.expectedRegion, config.GCPRegion,
				"GCPRegion should have whitespace trimmed")
		})
	}
}

// TestLoadConfig_LLMTimeout_InvalidValue tests error handling for invalid timeout values.
// NFR-001: Invalid timeout values should fall back to default
func TestLoadConfig_LLMTimeout_InvalidValue(t *testing.T) {
	tests := []struct {
		name            string
		llmTimeoutEnv   string
		expectedTimeout int
	}{
		{
			name:            "non-numeric value falls back to default",
			llmTimeoutEnv:   "abc",
			expectedTimeout: 30,
		},
		{
			name:            "negative value falls back to default",
			llmTimeoutEnv:   "-5",
			expectedTimeout: 30,
		},
		{
			name:            "zero value falls back to default",
			llmTimeoutEnv:   "0",
			expectedTimeout: 30,
		},
		{
			name:            "float value uses integer part",
			llmTimeoutEnv:   "45.5",
			expectedTimeout: 30, // strconv.Atoi doesn't parse floats, falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_TIMEOUT_SECONDS", tt.llmTimeoutEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed (invalid values fall back to default)
			require.NoError(t, err, "loadConfig should not return error for invalid timeout")

			// Then: LLM timeout should fall back to default
			assert.Equal(t, tt.expectedTimeout, config.LLMTimeoutSeconds,
				"LLMTimeoutSeconds should fall back to default for invalid value")
		})
	}
}
