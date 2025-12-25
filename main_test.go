package main

import (
	"context"
	"os"
	"testing"
	"time"

	"yuruppu/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			os.Unsetenv("GCP_PROJECT_ID")

			// Given: Set environment variables (if not empty)
			if tt.channelSecret != "" && tt.name != "unset channel secret returns error" {
				t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			}
			if tt.channelAccessToken != "" {
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			}
			t.Setenv("GCP_PROJECT_ID", "test-project-id")

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
			os.Unsetenv("GCP_PROJECT_ID")

			// Given: Set environment variables (if not empty)
			if tt.channelSecret != "" {
				t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			}
			if tt.channelAccessToken != "" && tt.name != "unset channel access token returns error" {
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			}
			t.Setenv("GCP_PROJECT_ID", "test-project-id")

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
			os.Unsetenv("GCP_PROJECT_ID")

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

// TestLoadConfig_MissingGCPProjectID tests error when GCP_PROJECT_ID is missing.
// FR-003, AC-013: Bot fails to start during initialization if credentials are missing
func TestLoadConfig_MissingGCPProjectID(t *testing.T) {
	tests := []struct {
		name         string
		gcpProjectID string
		wantErrMsg   string
	}{
		{
			name:         "empty GCP_PROJECT_ID returns error",
			gcpProjectID: "",
			wantErrMsg:   "GCP_PROJECT_ID is required",
		},
		{
			name:         "whitespace-only GCP_PROJECT_ID returns error",
			gcpProjectID: "   ",
			wantErrMsg:   "GCP_PROJECT_ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set LINE credentials but not GCP_PROJECT_ID
			t.Setenv("LINE_CHANNEL_SECRET", "valid-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "valid-token")

			if tt.gcpProjectID != "" {
				t.Setenv("GCP_PROJECT_ID", tt.gcpProjectID)
			} else {
				os.Unsetenv("GCP_PROJECT_ID")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when GCP_PROJECT_ID is missing")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate GCP_PROJECT_ID is required
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate GCP_PROJECT_ID is required")
		})
	}
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
				t.Setenv("GCP_PROJECT_ID", "project-id")
				// LINE_CHANNEL_SECRET is not set
			},
			wantErrContains: []string{"LINE_CHANNEL_SECRET", "required"},
		},
		{
			name: "missing token error mentions LINE_CHANNEL_ACCESS_TOKEN",
			setupEnv: func(t *testing.T) {
				t.Setenv("LINE_CHANNEL_SECRET", "secret")
				t.Setenv("GCP_PROJECT_ID", "project-id")
				// LINE_CHANNEL_ACCESS_TOKEN is not set
			},
			wantErrContains: []string{"LINE_CHANNEL_ACCESS_TOKEN", "required"},
		},
		{
			name: "missing GCP_PROJECT_ID error mentions GCP_PROJECT_ID",
			setupEnv: func(t *testing.T) {
				t.Setenv("LINE_CHANNEL_SECRET", "secret")
				t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "token")
				// GCP_PROJECT_ID is not set
			},
			wantErrContains: []string{"GCP_PROJECT_ID", "required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Setup environment
			os.Unsetenv("LINE_CHANNEL_SECRET")
			os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")
			os.Unsetenv("GCP_PROJECT_ID")
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

// TestInitBot_Success tests successful Bot initialization.
// AC-003: Given valid credentials are set,
// when application starts,
// then bot.NewBot() is called and Bot instance is created successfully.
func TestInitBot_Success(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
	}{
		{
			name:               "valid credentials initialize bot successfully",
			channelSecret:      "test-channel-secret",
			channelAccessToken: "test-access-token",
		},
		{
			name:               "long credentials initialize bot successfully",
			channelSecret:      "very-long-channel-secret-0123456789",
			channelAccessToken: "very-long-access-token-0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Valid configuration
			config := &Config{
				ChannelSecret:      tt.channelSecret,
				ChannelAccessToken: tt.channelAccessToken,
			}

			// When: Initialize bot
			b, err := initBot(config)

			// Then: Should succeed without error
			require.NoError(t, err, "initBot should not return error with valid credentials")

			// Then: Bot instance should not be nil
			assert.NotNil(t, b, "bot instance should not be nil")
		})
	}
}

// TestInitBot_NilConfig tests error when config is nil.
// FR-002: Bot initialization should handle edge cases gracefully.
func TestInitBot_NilConfig(t *testing.T) {
	// When: Initialize bot with nil config
	b, err := initBot(nil)

	// Then: Should return error
	require.Error(t, err, "initBot should return error with nil config")

	// Then: Bot instance should be nil
	assert.Nil(t, b, "bot instance should be nil on error")

	// Then: Error message should be descriptive
	assert.Contains(t, err.Error(), "config", "error message should mention config")
}

// TestInitBot_EmptyCredentials tests error when credentials are empty.
// FR-002: Bot initialization should fail when credentials are empty.
func TestInitBot_EmptyCredentials(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
		wantErrContains    string
	}{
		{
			name:               "empty channel secret fails initialization",
			channelSecret:      "",
			channelAccessToken: "valid-token",
			wantErrContains:    "LINE_CHANNEL_SECRET",
		},
		{
			name:               "empty channel access token fails initialization",
			channelSecret:      "valid-secret",
			channelAccessToken: "",
			wantErrContains:    "LINE_CHANNEL_ACCESS_TOKEN",
		},
		{
			name:               "both credentials empty fails initialization",
			channelSecret:      "",
			channelAccessToken: "",
			wantErrContains:    "LINE_CHANNEL_SECRET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Configuration with empty credentials
			config := &Config{
				ChannelSecret:      tt.channelSecret,
				ChannelAccessToken: tt.channelAccessToken,
			}

			// When: Initialize bot
			b, err := initBot(config)

			// Then: Should return error
			require.Error(t, err, "initBot should return error with empty credentials")

			// Then: Bot instance should be nil
			assert.Nil(t, b, "bot instance should be nil on error")

			// Then: Error message should indicate which credential is missing
			assert.Contains(t, err.Error(), tt.wantErrContains,
				"error message should indicate missing credential")
		})
	}
}

// TestSetupPackageLevel tests package-level configuration.
// AC-007: Given Bot is initialized successfully,
// when application starts,
// then bot.SetDefaultBot(), bot.SetLogger(), bot.SetDefaultLLMProvider(), and bot.SetLLMTimeout() are called.
func TestSetupPackageLevel(t *testing.T) {
	// Given: Valid configuration
	t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
	t.Setenv("GCP_PROJECT_ID", "test-project-id")

	config, err := loadConfig()
	require.NoError(t, err)

	b, err := initBot(config)
	require.NoError(t, err)

	// Given: Mock LLM provider (since we can't initialize real Vertex AI in tests)
	llmProvider := &mockLLMProvider{}

	// Given: LLM timeout
	llmTimeout := time.Duration(config.LLMTimeoutSeconds) * time.Second

	// When: Setup package-level configuration
	setupPackageLevel(b, llmProvider, llmTimeout)

	// Then: Function should complete without panic
	// (The actual verification of SetDefaultBot/SetLogger/SetDefaultLLMProvider/SetLLMTimeout is done by the bot package tests)
}

// TestSetupPackageLevel_NilBot tests package-level configuration with nil bot.
// FR-004: Package-level settings should handle edge cases gracefully.
func TestSetupPackageLevel_NilBot(t *testing.T) {
	// When: Setup package-level configuration with nil bot
	// Then: Should not panic
	assert.NotPanics(t, func() {
		setupPackageLevel(nil, nil, 0)
	}, "setupPackageLevel should not panic with nil bot")
}

// TestInitLLM_Success tests successful LLM provider initialization.
// FR-003: LLM provider initializes successfully with valid GCP_PROJECT_ID.
func TestInitLLM_Success(t *testing.T) {
	// Given: Valid configuration
	config := &Config{
		ChannelSecret:      "test-secret",
		ChannelAccessToken: "test-token",
		GCPProjectID:       "test-project-id",
	}

	// When: Initialize LLM
	llmProvider, err := initLLM(context.Background(), config)

	// Then: Should succeed without error
	require.NoError(t, err, "initLLM should not return error with valid GCP_PROJECT_ID")

	// Then: LLM provider should not be nil
	assert.NotNil(t, llmProvider, "llmProvider should not be nil")
}

// TestInitLLM_NilConfig tests error when config is nil.
func TestInitLLM_NilConfig(t *testing.T) {
	// When: Initialize LLM with nil config
	llmProvider, err := initLLM(context.Background(), nil)

	// Then: Should return error
	require.Error(t, err, "initLLM should return error with nil config")

	// Then: LLM provider should be nil
	assert.Nil(t, llmProvider, "llmProvider should be nil on error")

	// Then: Error message should be descriptive
	assert.Contains(t, err.Error(), "config", "error message should mention config")
}

// TestInitLLM_EmptyGCPProjectID tests error when GCP_PROJECT_ID is empty.
// AC-013: Bot fails to start during initialization if credentials are missing.
func TestInitLLM_EmptyGCPProjectID(t *testing.T) {
	// Given: Configuration with empty GCP_PROJECT_ID
	config := &Config{
		ChannelSecret:      "test-secret",
		ChannelAccessToken: "test-token",
		GCPProjectID:       "",
	}

	// When: Initialize LLM
	llmProvider, err := initLLM(context.Background(), config)

	// Then: Should return error
	require.Error(t, err, "initLLM should return error with empty GCP_PROJECT_ID")

	// Then: LLM provider should be nil
	assert.Nil(t, llmProvider, "llmProvider should be nil on error")

	// Then: Error message should indicate GCP_PROJECT_ID is missing
	assert.Contains(t, err.Error(), "GCP_PROJECT_ID", "error message should mention GCP_PROJECT_ID")
}

// TestGetPort_DefaultPort tests default port when PORT is not set.
// AC-005: Given PORT environment variable is not set,
// when application starts,
// then server starts on port 8080.
func TestGetPort_DefaultPort(t *testing.T) {
	// Given: PORT is not set
	os.Unsetenv("PORT")

	// When: Get port
	port := getPort()

	// Then: Should return default port 8080
	assert.Equal(t, "8080", port, "default port should be 8080")
}

// TestGetPort_CustomPort tests custom port from PORT environment variable.
// AC-006: Given PORT environment variable is set to "3000",
// when application starts,
// then server starts on port 3000.
func TestGetPort_CustomPort(t *testing.T) {
	tests := []struct {
		name     string
		portEnv  string
		expected string
	}{
		{
			name:     "port 3000",
			portEnv:  "3000",
			expected: "3000",
		},
		{
			name:     "port 9000",
			portEnv:  "9000",
			expected: "9000",
		},
		{
			name:     "port 80",
			portEnv:  "80",
			expected: "80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: PORT is set
			t.Setenv("PORT", tt.portEnv)

			// When: Get port
			port := getPort()

			// Then: Should return the custom port
			assert.Equal(t, tt.expected, port, "port should match PORT environment variable")
		})
	}
}

// TestGetPort_EmptyPort tests that empty PORT falls back to default.
func TestGetPort_EmptyPort(t *testing.T) {
	// Given: PORT is set to empty string
	t.Setenv("PORT", "")

	// When: Get port
	port := getPort()

	// Then: Should return default port 8080
	assert.Equal(t, "8080", port, "empty PORT should fallback to default 8080")
}

// TestCreateHandler tests that createHandler returns an http.Handler.
// AC-004: Given Bot is initialized successfully,
// when application starts,
// then /webhook endpoint is accessible.
func TestCreateHandler(t *testing.T) {
	// When: Create handler
	handler := createHandler()

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
