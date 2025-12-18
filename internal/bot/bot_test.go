package bot_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestBot_VerifySignature_ValidSignature tests signature verification with valid signatures.
// AC-003: Given LINE bot webhook endpoint is exposed,
// when request with valid signature is received,
// then request is accepted with HTTP 200, message is processed.
func TestBot_VerifySignature_ValidSignature(t *testing.T) {
	tests := []struct {
		name          string
		channelSecret string
		requestBody   string
		wantAccepted  bool
	}{
		{
			name:          "valid signature returns true",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[{"type":"message","message":{"type":"text","text":"Hello"}}]}`,
			wantAccepted:  true,
		},
		{
			name:          "valid signature with empty events returns true",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[]}`,
			wantAccepted:  true,
		},
		{
			name:          "valid signature with complex body returns true",
			channelSecret: "another-secret-123",
			requestBody:   `{"events":[{"type":"message","message":{"type":"text","text":"こんにちは 世界 🌍"},"replyToken":"test-token"}]}`,
			wantAccepted:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create bot with channel secret
			b, err := bot.NewBot(tt.channelSecret, "test-access-token")
			require.NoError(t, err)

			// Given: Create HTTP request with valid signature
			req := createRequestWithValidSignature(t, tt.channelSecret, tt.requestBody)

			// When: Verify signature
			accepted := b.VerifySignature(req)

			// Then: Signature should be accepted
			assert.Equal(t, tt.wantAccepted, accepted,
				"valid signature should be accepted")
		})
	}
}

// TestBot_VerifySignature_InvalidSignature tests signature verification with invalid signatures.
// AC-002: Given LINE bot webhook endpoint is exposed,
// when request with invalid signature is received,
// then request is rejected with HTTP 401, no reply is sent.
func TestBot_VerifySignature_InvalidSignature(t *testing.T) {
	tests := []struct {
		name          string
		channelSecret string
		requestBody   string
		signature     string
		wantAccepted  bool
	}{
		{
			name:          "completely wrong signature returns false",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[{"type":"message"}]}`,
			signature:     "invalid-signature-abc123",
			wantAccepted:  false,
		},
		{
			name:          "signature for different secret returns false",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[{"type":"message"}]}`,
			signature:     computeSignature("wrong-secret", `{"events":[{"type":"message"}]}`),
			wantAccepted:  false,
		},
		{
			name:          "signature for different body returns false",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[{"type":"message","text":"Hello"}]}`,
			signature:     computeSignature("test-channel-secret", `{"events":[]}`),
			wantAccepted:  false,
		},
		{
			name:          "empty signature returns false",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[]}`,
			signature:     "",
			wantAccepted:  false,
		},
		{
			name:          "malformed base64 signature returns false",
			channelSecret: "test-channel-secret",
			requestBody:   `{"events":[]}`,
			signature:     "not-valid-base64!@#$%",
			wantAccepted:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create bot with channel secret
			b, err := bot.NewBot(tt.channelSecret, "test-access-token")
			require.NoError(t, err)

			// Given: Create HTTP request with invalid signature
			req := createRequestWithSignature(t, tt.requestBody, tt.signature)

			// When: Verify signature
			accepted := b.VerifySignature(req)

			// Then: Signature should be rejected
			assert.Equal(t, tt.wantAccepted, accepted,
				"invalid signature should be rejected")
		})
	}
}

// TestBot_VerifySignature_MissingSignature tests signature verification when signature header is missing.
// AC-002: Request without signature header should be rejected.
func TestBot_VerifySignature_MissingSignature(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  string
		wantAccepted bool
	}{
		{
			name:         "missing signature header returns false",
			requestBody:  `{"events":[{"type":"message"}]}`,
			wantAccepted: false,
		},
		{
			name:         "missing signature with empty body returns false",
			requestBody:  `{}`,
			wantAccepted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create bot
			b, err := bot.NewBot("test-channel-secret", "test-access-token")
			require.NoError(t, err)

			// Given: Create HTTP request without signature header
			req := createRequestWithoutSignature(t, tt.requestBody)

			// When: Verify signature
			accepted := b.VerifySignature(req)

			// Then: Request should be rejected
			assert.Equal(t, tt.wantAccepted, accepted,
				"request without signature should be rejected")
		})
	}
}

// TestBot_VerifySignature_EmptyBody tests signature verification with empty request body.
func TestBot_VerifySignature_EmptyBody(t *testing.T) {
	tests := []struct {
		name          string
		channelSecret string
		requestBody   string
		wantAccepted  bool
	}{
		{
			name:          "empty body with valid signature returns true",
			channelSecret: "test-channel-secret",
			requestBody:   "",
			wantAccepted:  true,
		},
		{
			name:          "whitespace-only body with valid signature returns true",
			channelSecret: "test-channel-secret",
			requestBody:   "   ",
			wantAccepted:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create bot with channel secret
			b, err := bot.NewBot(tt.channelSecret, "test-access-token")
			require.NoError(t, err)

			// Given: Create HTTP request with valid signature for empty/whitespace body
			req := createRequestWithValidSignature(t, tt.channelSecret, tt.requestBody)

			// When: Verify signature
			accepted := b.VerifySignature(req)

			// Then: Should handle empty body correctly
			assert.Equal(t, tt.wantAccepted, accepted,
				"empty body should be handled correctly")
		})
	}
}

// TestBot_VerifySignature_BodyConsumed tests that signature verification can be called after reading body.
func TestBot_VerifySignature_BodyConsumed(t *testing.T) {
	t.Run("can verify signature after body is read", func(t *testing.T) {
		// Given: Create bot
		b, err := bot.NewBot("test-channel-secret", "test-access-token")
		require.NoError(t, err)

		requestBody := `{"events":[{"type":"message"}]}`
		req := createRequestWithValidSignature(t, "test-channel-secret", requestBody)

		// Given: Read the body (simulating middleware or previous handler)
		_ = req.Body
		// Body is consumed here in real scenarios

		// When: Verify signature
		accepted := b.VerifySignature(req)

		// Then: Should still be able to verify
		// Note: Implementation should handle this case (e.g., by caching body)
		assert.True(t, accepted,
			"should be able to verify signature even after body is consumed")
	})
}

// Helper functions for creating test HTTP requests

// createRequestWithValidSignature creates an HTTP request with a valid LINE webhook signature.
func createRequestWithValidSignature(t *testing.T, channelSecret, body string) *http.Request {
	t.Helper()
	signature := computeSignature(channelSecret, body)
	return createRequestWithSignature(t, body, signature)
}

// createRequestWithSignature creates an HTTP request with the specified signature.
func createRequestWithSignature(t *testing.T, body, signature string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Line-Signature", signature)
	return req
}

// createRequestWithoutSignature creates an HTTP request without a signature header.
func createRequestWithoutSignature(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// computeSignature computes the HMAC-SHA256 signature for LINE webhook verification.
// The signature is base64-encoded HMAC-SHA256 of the request body using the channel secret.
func computeSignature(channelSecret, body string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
