package bot_test

import (
	"testing"

	"yuruppu/internal/bot"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBot_ValidationErrors tests credential validation in NewBot.
// AC-009: Given env vars are missing, when bot attempts to start,
// then bot fails with ConfigError indicating which variable is missing.
func TestNewBot_ValidationErrors(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
		wantErr            bool
		wantErrMsg         string
	}{
		{
			name:               "empty channel secret returns error",
			channelSecret:      "",
			channelAccessToken: "valid-access-token",
			wantErr:            true,
			wantErrMsg:         "LINE_CHANNEL_SECRET",
		},
		{
			name:               "empty channel access token returns error",
			channelSecret:      "valid-secret",
			channelAccessToken: "",
			wantErr:            true,
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN",
		},
		{
			name:               "both credentials empty returns error",
			channelSecret:      "",
			channelAccessToken: "",
			wantErr:            true,
			wantErrMsg:         "LINE_CHANNEL_SECRET",
		},
		{
			name:               "whitespace-only channel secret returns error",
			channelSecret:      "   ",
			channelAccessToken: "valid-access-token",
			wantErr:            true,
			wantErrMsg:         "LINE_CHANNEL_SECRET",
		},
		{
			name:               "whitespace-only channel access token returns error",
			channelSecret:      "valid-secret",
			channelAccessToken: "   ",
			wantErr:            true,
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			client, err := bot.NewBot(tt.channelSecret, tt.channelAccessToken)

			// Then
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.wantErrMsg,
					"error message should indicate which variable is missing")
				// Verify it's a ConfigError
				var configErr *bot.ConfigError
				assert.ErrorAs(t, err, &configErr,
					"error should be of type ConfigError")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestNewBot_ValidCredentials tests successful bot initialization.
// AC-008: Given env vars are set, when bot starts,
// then bot initializes successfully.
func TestNewBot_ValidCredentials(t *testing.T) {
	tests := []struct {
		name               string
		channelSecret      string
		channelAccessToken string
	}{
		{
			name:               "valid credentials returns bot client",
			channelSecret:      "test-channel-secret",
			channelAccessToken: "test-access-token",
		},
		{
			name:               "long credentials are accepted",
			channelSecret:      "very-long-channel-secret-with-many-characters-0123456789",
			channelAccessToken: "very-long-access-token-with-many-characters-0123456789",
		},
		{
			name:               "special characters in credentials are accepted",
			channelSecret:      "secret-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			channelAccessToken: "token-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			client, err := bot.NewBot(tt.channelSecret, tt.channelAccessToken)

			// Then
			require.NoError(t, err)
			assert.NotNil(t, client, "client should not be nil on successful initialization")
		})
	}
}

// TestConfigError_ErrorMessage tests ConfigError formatting.
// Ensures error messages clearly indicate which variable is missing.
func TestConfigError_ErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		variableName string
		wantContains string
	}{
		{
			name:         "ConfigError contains variable name",
			variableName: "LINE_CHANNEL_SECRET",
			wantContains: "LINE_CHANNEL_SECRET",
		},
		{
			name:         "ConfigError for access token",
			variableName: "LINE_CHANNEL_ACCESS_TOKEN",
			wantContains: "LINE_CHANNEL_ACCESS_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			err := &bot.ConfigError{Variable: tt.variableName}

			// When
			msg := err.Error()

			// Then
			assert.Contains(t, msg, tt.wantContains,
				"error message should contain the variable name")
			assert.Contains(t, msg, "Missing required environment variable",
				"error message should indicate it's a missing environment variable")
		})
	}
}
