package main

import (
	"log/slog"
	"os"
	"testing"
	"time"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// =============================================================================
// LINE Credentials Tests
// =============================================================================

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
			channelSecret:      "test-long-secret",
			channelAccessToken: "test-long-token",
			gcpProjectID:       "test-long-project-id",
		},
		{
			name:               "credentials with special characters are accepted",
			channelSecret:      "secret-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			channelAccessToken: "token-with-special!@#$%^&*()_+-=[]{}|;:,.<>?",
			gcpProjectID:       "test-project-id-hyphenated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			t.Setenv("GCP_PROJECT_ID", tt.gcpProjectID)
			t.Setenv("LLM_MODEL", "test-model")

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
			channelAccessToken: "test-access-token",
			wantErrMsg:         "LINE_CHANNEL_SECRET is required",
		},
		{
			name:               "unset channel secret returns error",
			channelSecret:      "", // Will not be set in environment
			channelAccessToken: "test-access-token",
			wantErrMsg:         "LINE_CHANNEL_SECRET is required",
		},
		{
			name:               "whitespace-only channel secret returns error",
			channelSecret:      "   ",
			channelAccessToken: "test-access-token",
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
			channelSecret:      "test-channel-secret",
			channelAccessToken: "",
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN is required",
		},
		{
			name:               "unset channel access token returns error",
			channelSecret:      "test-channel-secret",
			channelAccessToken: "", // Will not be set in environment
			wantErrMsg:         "LINE_CHANNEL_ACCESS_TOKEN is required",
		},
		{
			name:               "whitespace-only channel access token returns error",
			channelSecret:      "test-channel-secret",
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
			gcpProjectID:          "  test-project-id",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "test-project-id",
		},
		{
			name:                  "trailing whitespace is trimmed",
			channelSecret:         "secret  ",
			channelAccessToken:    "token  ",
			gcpProjectID:          "test-project-id  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "test-project-id",
		},
		{
			name:                  "leading and trailing whitespace is trimmed",
			channelSecret:         "  secret  ",
			channelAccessToken:    "  token  ",
			gcpProjectID:          "  test-project-id  ",
			expectedChannelSecret: "secret",
			expectedAccessToken:   "token",
			expectedProjectID:     "test-project-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set environment variables with whitespace
			t.Setenv("LINE_CHANNEL_SECRET", tt.channelSecret)
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", tt.channelAccessToken)
			t.Setenv("GCP_PROJECT_ID", tt.gcpProjectID)
			t.Setenv("LLM_MODEL", "test-model")

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

// =============================================================================
// GCP Configuration Tests
// =============================================================================

// TestLoadConfig_GCPConfigOptional tests that GCP config is optional in loadConfig.
// AC-002, AC-003: GCP_PROJECT_ID and GCP_REGION are optional (auto-detected on Cloud Run).
func TestLoadConfig_GCPConfigOptional(t *testing.T) {
	// Given: Set LINE credentials and LLM_MODEL but not GCP config
	t.Setenv("LINE_CHANNEL_SECRET", "test-valid-secret")
	t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-valid-token")
	t.Setenv("LLM_MODEL", "test-model")
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
			gcpRegionEnv:   "test-region",
			expectedRegion: "test-region",
		},
		{
			name:           "region test-region",
			gcpRegionEnv:   "test-region",
			expectedRegion: "test-region",
		},
		{
			name:           "region test-region-eu",
			gcpRegionEnv:   "test-region-eu",
			expectedRegion: "test-region-eu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("LLM_MODEL", "test-model")

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
			gcpRegionEnv:   "  test-region",
			expectedRegion: "test-region",
		},
		{
			name:           "trailing whitespace is trimmed",
			gcpRegionEnv:   "test-region  ",
			expectedRegion: "test-region",
		},
		{
			name:           "leading and trailing whitespace is trimmed",
			gcpRegionEnv:   "  test-region  ",
			expectedRegion: "test-region",
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
			t.Setenv("LLM_MODEL", "test-model")
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

// =============================================================================
// Port Configuration Tests
// =============================================================================

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
			t.Setenv("LLM_MODEL", "test-model")

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
			t.Setenv("LLM_MODEL", "test-model")
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

// =============================================================================
// LLM Model Configuration Tests (FX-001, FX-002, FX-005, AC-001)
// =============================================================================

// TestLoadConfig_LLMModel_Valid tests loading configuration with valid LLM_MODEL.
// FX-001: Config struct has LLMModel field
// FX-002: loadConfig loads LLM_MODEL as required env var
func TestLoadConfig_LLMModel_Valid(t *testing.T) {
	tests := []struct {
		name          string
		llmModel      string
		expectedModel string
	}{
		{
			name:          "valid model name is loaded",
			llmModel:      "test-model",
			expectedModel: "test-model",
		},
		{
			name:          "different model name is loaded",
			llmModel:      "another-model",
			expectedModel: "another-model",
		},
		{
			name:          "model name with special characters is accepted",
			llmModel:      "model-with-special_chars.v2",
			expectedModel: "model-with-special_chars.v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("LLM_MODEL", tt.llmModel)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error with valid LLM_MODEL")

			// Then: LLMModel should match expected value
			assert.Equal(t, tt.expectedModel, config.LLMModel,
				"LLMModel should match environment variable")
		})
	}
}

// TestLoadConfig_LLMModel_Missing tests error when LLM_MODEL is missing.
// AC-001: Given LLM_MODEL is not set, when application starts, then error with LLM_MODEL
func TestLoadConfig_LLMModel_Missing(t *testing.T) {
	tests := []struct {
		name       string
		llmModel   string
		setEnv     bool
		wantErrMsg string
	}{
		{
			name:       "unset LLM_MODEL returns error",
			llmModel:   "",
			setEnv:     false,
			wantErrMsg: "LLM_MODEL is required",
		},
		{
			name:       "empty LLM_MODEL returns error",
			llmModel:   "",
			setEnv:     true,
			wantErrMsg: "LLM_MODEL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set LINE credentials
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")

			// Given: Handle LLM_MODEL based on test case
			os.Unsetenv("LLM_MODEL")
			if tt.setEnv {
				t.Setenv("LLM_MODEL", tt.llmModel)
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when LLM_MODEL is missing")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate LLM_MODEL is required
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate LLM_MODEL is required")
		})
	}
}

// TestLoadConfig_LLMModel_WhitespaceOnly tests error when LLM_MODEL is whitespace-only.
// FX-005: LLM_MODEL that is empty or whitespace-only returns error
func TestLoadConfig_LLMModel_WhitespaceOnly(t *testing.T) {
	tests := []struct {
		name       string
		llmModel   string
		wantErrMsg string
	}{
		{
			name:       "whitespace-only LLM_MODEL returns error",
			llmModel:   "   ",
			wantErrMsg: "LLM_MODEL is required",
		},
		{
			name:       "tabs-only LLM_MODEL returns error",
			llmModel:   "\t\t",
			wantErrMsg: "LLM_MODEL is required",
		},
		{
			name:       "mixed whitespace LLM_MODEL returns error",
			llmModel:   " \t \n ",
			wantErrMsg: "LLM_MODEL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set LINE credentials
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("LLM_MODEL", tt.llmModel)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error
			require.Error(t, err, "loadConfig should return error when LLM_MODEL is whitespace-only")

			// Then: Config should be nil
			assert.Nil(t, config, "config should be nil on error")

			// Then: Error message should indicate LLM_MODEL is required
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate LLM_MODEL is required")
		})
	}
}

// TestLoadConfig_LLMModel_TrimsWhitespace tests that LLM_MODEL value is trimmed.
func TestLoadConfig_LLMModel_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		name          string
		llmModel      string
		expectedModel string
	}{
		{
			name:          "leading whitespace is trimmed",
			llmModel:      "  test-model",
			expectedModel: "test-model",
		},
		{
			name:          "trailing whitespace is trimmed",
			llmModel:      "test-model  ",
			expectedModel: "test-model",
		},
		{
			name:          "leading and trailing whitespace is trimmed",
			llmModel:      "  test-model  ",
			expectedModel: "test-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("LLM_MODEL", tt.llmModel)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: LLMModel should have whitespace trimmed
			assert.Equal(t, tt.expectedModel, config.LLMModel,
				"LLMModel should have whitespace trimmed")
		})
	}
}

// =============================================================================
// LLM Timeout Configuration Tests
// =============================================================================

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
			t.Setenv("LLM_MODEL", "test-model")

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

// TestLoadConfig_LLMTimeout_InvalidValue tests error handling for invalid timeout values.
// NFR-001: Invalid timeout values should return error
func TestLoadConfig_LLMTimeout_InvalidValue(t *testing.T) {
	tests := []struct {
		name          string
		llmTimeoutEnv string
		wantErrMsg    string
	}{
		{
			name:          "non-numeric value returns error",
			llmTimeoutEnv: "abc",
			wantErrMsg:    "LLM_TIMEOUT_SECONDS must be a positive integer",
		},
		{
			name:          "negative value returns error",
			llmTimeoutEnv: "-5",
			wantErrMsg:    "LLM_TIMEOUT_SECONDS must be a positive integer",
		},
		{
			name:          "zero value returns error",
			llmTimeoutEnv: "0",
			wantErrMsg:    "LLM_TIMEOUT_SECONDS must be a positive integer",
		},
		{
			name:          "float value returns error",
			llmTimeoutEnv: "45.5",
			wantErrMsg:    "LLM_TIMEOUT_SECONDS must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_MODEL", "test-model")
			t.Setenv("LLM_TIMEOUT_SECONDS", tt.llmTimeoutEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error for invalid values
			require.Error(t, err, "loadConfig should return error for invalid timeout")
			assert.Nil(t, config, "config should be nil on error")
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate invalid timeout value")
		})
	}
}

// TestLoadConfig_GCPMetadataTimeout tests GCP metadata timeout configuration loading.
func TestLoadConfig_GCPMetadataTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeoutEnv      string
		expectedTimeout int
	}{
		{
			name:            "default timeout is 2 seconds when not set",
			timeoutEnv:      "",
			expectedTimeout: 2,
		},
		{
			name:            "custom timeout from environment variable",
			timeoutEnv:      "5",
			expectedTimeout: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_MODEL", "test-model")

			if tt.timeoutEnv != "" {
				t.Setenv("GCP_METADATA_TIMEOUT_SECONDS", tt.timeoutEnv)
			} else {
				os.Unsetenv("GCP_METADATA_TIMEOUT_SECONDS")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: GCP metadata timeout should match expected value
			assert.Equal(t, tt.expectedTimeout, config.GCPMetadataTimeoutSeconds,
				"GCPMetadataTimeoutSeconds should match expected value")
		})
	}
}

// TestLoadConfig_GCPMetadataTimeout_InvalidValue tests error handling for invalid timeout values.
func TestLoadConfig_GCPMetadataTimeout_InvalidValue(t *testing.T) {
	tests := []struct {
		name       string
		timeoutEnv string
		wantErrMsg string
	}{
		{
			name:       "non-numeric value returns error",
			timeoutEnv: "abc",
			wantErrMsg: "GCP_METADATA_TIMEOUT_SECONDS must be a positive integer",
		},
		{
			name:       "negative value returns error",
			timeoutEnv: "-5",
			wantErrMsg: "GCP_METADATA_TIMEOUT_SECONDS must be a positive integer",
		},
		{
			name:       "zero value returns error",
			timeoutEnv: "0",
			wantErrMsg: "GCP_METADATA_TIMEOUT_SECONDS must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_MODEL", "test-model")
			t.Setenv("GCP_METADATA_TIMEOUT_SECONDS", tt.timeoutEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error for invalid values
			require.Error(t, err, "loadConfig should return error for invalid timeout")
			assert.Nil(t, config, "config should be nil on error")
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate invalid timeout value")
		})
	}
}

// =============================================================================
// LLM Cache TTL Configuration Tests
// =============================================================================

// TestLoadConfig_LLMCacheTTL tests LLM cache TTL configuration loading.
func TestLoadConfig_LLMCacheTTL(t *testing.T) {
	tests := []struct {
		name            string
		llmCacheTTLEnv  string
		expectedCacheTTL int
	}{
		{
			name:             "default TTL is 60 minutes when not set",
			llmCacheTTLEnv:   "",
			expectedCacheTTL: 60,
		},
		{
			name:             "custom TTL from environment variable",
			llmCacheTTLEnv:   "120",
			expectedCacheTTL: 120,
		},
		{
			name:             "TTL of 1 minute",
			llmCacheTTLEnv:   "1",
			expectedCacheTTL: 1,
		},
		{
			name:             "TTL of 1440 minutes (24 hours)",
			llmCacheTTLEnv:   "1440",
			expectedCacheTTL: 1440,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_MODEL", "test-model")

			if tt.llmCacheTTLEnv != "" {
				t.Setenv("LLM_CACHE_TTL_MINUTES", tt.llmCacheTTLEnv)
			} else {
				os.Unsetenv("LLM_CACHE_TTL_MINUTES")
			}

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should succeed without error
			require.NoError(t, err, "loadConfig should not return error")

			// Then: LLM cache TTL should match expected value
			assert.Equal(t, tt.expectedCacheTTL, config.LLMCacheTTLMinutes,
				"LLMCacheTTLMinutes should match expected value")
		})
	}
}

// TestLoadConfig_LLMCacheTTL_InvalidValue tests error handling for invalid TTL values.
func TestLoadConfig_LLMCacheTTL_InvalidValue(t *testing.T) {
	tests := []struct {
		name           string
		llmCacheTTLEnv string
		wantErrMsg     string
	}{
		{
			name:           "non-numeric value returns error",
			llmCacheTTLEnv: "abc",
			wantErrMsg:     "LLM_CACHE_TTL_MINUTES must be a positive integer",
		},
		{
			name:           "negative value returns error",
			llmCacheTTLEnv: "-5",
			wantErrMsg:     "LLM_CACHE_TTL_MINUTES must be a positive integer",
		},
		{
			name:           "zero value returns error",
			llmCacheTTLEnv: "0",
			wantErrMsg:     "LLM_CACHE_TTL_MINUTES must be a positive integer",
		},
		{
			name:           "float value returns error",
			llmCacheTTLEnv: "60.5",
			wantErrMsg:     "LLM_CACHE_TTL_MINUTES must be a positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set required environment variables
			t.Setenv("LINE_CHANNEL_SECRET", "test-secret")
			t.Setenv("LINE_CHANNEL_ACCESS_TOKEN", "test-token")
			t.Setenv("GCP_PROJECT_ID", "test-project-id")
			t.Setenv("LLM_MODEL", "test-model")
			t.Setenv("LLM_CACHE_TTL_MINUTES", tt.llmCacheTTLEnv)

			// When: Load configuration
			config, err := loadConfig()

			// Then: Should return error for invalid values
			require.Error(t, err, "loadConfig should return error for invalid TTL")
			assert.Nil(t, config, "config should be nil on error")
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should indicate invalid TTL value")
		})
	}
}

// =============================================================================
// Handler Tests
// =============================================================================

// TestCreateHandler tests that createHandler returns an http.Handler.
// AC-004: Given Server is initialized successfully,
// when application starts,
// then /webhook endpoint is accessible.
func TestCreateHandler(t *testing.T) {
	// Given: Create a server directly
	server, err := line.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	// When: Create handler
	handler := createHandler(server)

	// Then: Should return non-nil handler
	assert.NotNil(t, handler, "handler should not be nil")
}
