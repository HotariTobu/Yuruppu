package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_ValidCredentials tests loading configuration with valid environment variables.
// AC-001: Given LINE_CHANNEL_SECRET and LINE_CHANNEL_ACCESS_TOKEN are set,
// when application starts, then no error occurs and values are used for Bot initialization.
func TestLoadConfig_ValidCredentials(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
	}{
		{
			name:               "both credentials set returns config",
			channelSecret:      "test-channel-secret",
			channelAccessToken: "test-access-token",
		},
		{
			name:               "long credentials are accepted",
			channelSecret:      "very-long-channel-secret-with-many-characters-0123456789",
			channelAccessToken: "very-long-access-token-with-many-characters-0123456789",
		},
		{
			name:               "credentials with special characters are accepted",
			channelSecret:      "secret-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			channelAccessToken: "token-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)

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
			// Given: Unset both environment variables
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

// TestLoadConfig_TrimsWhitespace tests that configuration values are trimmed.
// This ensures that accidental whitespace in environment variables doesn't cause issues.
func TestLoadConfig_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name                  string
		channelSecret         string
		channelAccessToken    string
		expectedChannelSecret string
		expectedAccessToken   string
	}{
		{
			name:                  "leading whitespace is trimmed",
			channelSecret:         "  secret",
			channelAccessToken:    "  token",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
		},
		{
			name:                  "trailing whitespace is trimmed",
			channelSecret:         "secret  ",
			channelAccessToken:    "token  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
		},
		{
			name:                  "leading and trailing whitespace is trimmed",
			channelSecret:         "  secret  ",
			channelAccessToken:    "  token  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables with whitespace
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed
			require.NoError(t, err, "loadConfig should not return error")

			// Then: Whitespace should be trimmed
			assert.Equal(t, tt.expectedChannelSecret, config.ChannelSecret,
				"ChannelSecret should have whitespace trimmed")
			assert.Equal(t, tt.expectedAccessToken, config.ChannelAccessToken,
				"ChannelAccessToken should have whitespace trimmed")
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
