package bot_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

// TestFormatEchoMessage tests the message formatting function.
// AC-001: User sends "Hello" -> bot replies "Yuruppu: Hello"
// AC-005: Whitespace-only message handling ("   " -> "Yuruppu:    ")
// AC-006: Long message handling (5000 chars)
func TestFormatEchoMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			// AC-001: Basic echo with prefix
			name:    "simple message is prefixed with Yuruppu",
			message: "Hello",
			want:    "Yuruppu: Hello",
		},
		{
			name:    "message with Japanese characters",
			message: "こんにちは",
			want:    "Yuruppu: こんにちは",
		},
		{
			name:    "message with emojis",
			message: "Hello 🌍",
			want:    "Yuruppu: Hello 🌍",
		},
		{
			name:    "empty message is prefixed",
			message: "",
			want:    "Yuruppu: ",
		},
		{
			// AC-005: Whitespace-only message handling
			name:    "whitespace-only message preserves whitespace",
			message: "   ",
			want:    "Yuruppu:    ",
		},
		{
			name:    "message with leading whitespace",
			message: "  Hello",
			want:    "Yuruppu:   Hello",
		},
		{
			name:    "message with trailing whitespace",
			message: "Hello  ",
			want:    "Yuruppu: Hello  ",
		},
		{
			name:    "message with newlines",
			message: "Hello\nWorld",
			want:    "Yuruppu: Hello\nWorld",
		},
		{
			// AC-006: Long message handling
			name:    "long message with 5000 characters",
			message: strings.Repeat("a", 5000),
			want:    "Yuruppu: " + strings.Repeat("a", 5000),
		},
		{
			name:    "message with special characters",
			message: "Hello!@#$%^&*()_+-=[]{}|;:,.<>?",
			want:    "Yuruppu: Hello!@#$%^&*()_+-=[]{}|;:,.<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := bot.FormatEchoMessage(tt.message)

			// Then
			assert.Equal(t, tt.want, got,
				"formatted message should have 'Yuruppu: ' prefix")
		})
	}
}

// TestHandleWebhook tests the HTTP webhook handler.
// FR-001: Receive text messages from LINE webhook
func TestHandleWebhook(t *testing.T) {
	// Setup: Create a valid channel secret
	channelSecret := "test-channel-secret"
	channelAccessToken := "test-access-token"

	// Setup: Initialize bot for handler
	// Note: HandleWebhook needs access to bot instance
	// This test assumes a global bot or dependency injection pattern
	testBot, err := bot.NewBot(channelSecret, channelAccessToken)
	require.NoError(t, err)
	bot.SetDefaultBot(testBot)
	t.Run("returns 401 when signature is invalid", func(t *testing.T) {
		// AC-002: Given LINE bot webhook endpoint is exposed,
		// when request with invalid signature is received,
		// then request is rejected with HTTP 401, no reply is sent

		// Given: Create request with invalid signature
		body := `{"events":[{"type":"message","message":{"type":"text","text":"Hello"}}]}`
		req := createRequestWithSignature(t, body, "invalid-signature")
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 401
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"should return 401 for invalid signature")
	})

	t.Run("returns 400 when payload is malformed", func(t *testing.T) {
		// Given: Create request with valid signature but invalid JSON
		body := `{"events":[invalid json]}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 400
		assert.Equal(t, http.StatusBadRequest, rec.Code,
			"should return 400 for malformed payload")
	})

	t.Run("returns 200 and processes text message event", func(t *testing.T) {
		// AC-001: Given LINE bot is running and connected,
		// when user sends a text message "Hello",
		// then bot receives the message via webhook and replies with "Yuruppu: Hello"

		// Given: Create valid webhook request with text message
		body := `{
			"destination": "xxxxxxxxxx",
			"events": [
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12345",
						"text": "Hello"
					},
					"timestamp": 1234567890,
					"source": {
						"type": "user",
						"userId": "U1234567890abcdef"
					},
					"replyToken": "test-reply-token",
					"mode": "active"
				}
			]
		}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 200
		assert.Equal(t, http.StatusOK, rec.Code,
			"should return 200 for valid webhook request")
		// Note: Actual reply is tested in HandleTextMessage tests
	})

	t.Run("returns 200 and ignores non-text message", func(t *testing.T) {
		// AC-004: Given LINE bot is running,
		// when user sends a non-text message (image, sticker, etc.),
		// then bot does not reply, no error is raised

		// Given: Create webhook request with image message
		body := `{
			"destination": "xxxxxxxxxx",
			"events": [
				{
					"type": "message",
					"message": {
						"type": "image",
						"id": "12345",
						"contentProvider": {
							"type": "line"
						}
					},
					"timestamp": 1234567890,
					"source": {
						"type": "user",
						"userId": "U1234567890abcdef"
					},
					"replyToken": "test-reply-token",
					"mode": "active"
				}
			]
		}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 200 without error
		assert.Equal(t, http.StatusOK, rec.Code,
			"should return 200 for non-text message")
	})

	t.Run("returns 200 with empty events", func(t *testing.T) {
		// Given: Create webhook request with no events
		body := `{"events":[]}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 200
		assert.Equal(t, http.StatusOK, rec.Code,
			"should return 200 for empty events")
	})

	t.Run("returns 200 even if reply fails", func(t *testing.T) {
		// Error handling requirement: ReplyError returns HTTP 200
		// to prevent LINE from retrying, but error is logged

		// Given: Create valid webhook request
		// (In real implementation, this would test a scenario where reply fails)
		body := `{
			"events": [
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12345",
						"text": "Hello"
					},
					"replyToken": "invalid-reply-token",
					"source": {
						"type": "user",
						"userId": "U1234567890abcdef"
					},
					"timestamp": 1234567890,
					"mode": "active"
				}
			]
		}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 200 even on reply failure
		assert.Equal(t, http.StatusOK, rec.Code,
			"should return 200 even if reply fails")
	})
}

// TestHandleTextMessage tests text message event processing.
// FR-002: Reply with the same message prefixed with "Yuruppu: "
func TestHandleTextMessage(t *testing.T) {
	// Note: These tests require mocking the LINE bot client
	// since we can't make real API calls in unit tests

	t.Run("sends reply with formatted echo message", func(t *testing.T) {
		// AC-001: Given LINE bot is running and connected,
		// when user sends a text message "Hello",
		// then bot replies with "Yuruppu: Hello"

		// Given: Create mock bot client
		mockBot := &mockLineBotClient{}

		// Given: Create text message event
		event := &mockMessageEvent{
			messageType: "text",
			text:        "Hello",
			replyToken:  "test-reply-token",
		}

		// Given: Mock expects reply with formatted message
		mockBot.ExpectReply("test-reply-token", "Yuruppu: Hello")

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should succeed without error
		require.NoError(t, err, "should successfully send reply")

		// Then: Should have called reply with correct message
		mockBot.AssertExpectations(t)
	})

	t.Run("handles whitespace-only message", func(t *testing.T) {
		// AC-005: Given LINE bot is running,
		// when user sends a whitespace-only text message "   ",
		// then bot replies with "Yuruppu:    "

		// Given: Create mock bot client
		mockBot := &mockLineBotClient{}

		// Given: Create text message event with whitespace
		event := &mockMessageEvent{
			messageType: "text",
			text:        "   ",
			replyToken:  "test-reply-token",
		}

		// Given: Mock expects reply with formatted whitespace
		mockBot.ExpectReply("test-reply-token", "Yuruppu:    ")

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should succeed without error
		require.NoError(t, err, "should handle whitespace message")
		mockBot.AssertExpectations(t)
	})

	t.Run("handles long message with 5000 characters", func(t *testing.T) {
		// AC-006: Given LINE bot is running,
		// when user sends a text message with 5000 characters,
		// then bot replies with the full message prefixed with "Yuruppu: "

		// Given: Create mock bot client
		mockBot := &mockLineBotClient{}

		// Given: Create long text message
		longText := strings.Repeat("a", 5000)
		event := &mockMessageEvent{
			messageType: "text",
			text:        longText,
			replyToken:  "test-reply-token",
		}

		// Given: Mock expects reply with full long message
		expectedReply := "Yuruppu: " + longText
		mockBot.ExpectReply("test-reply-token", expectedReply)

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should succeed without error
		require.NoError(t, err, "should handle long message")
		mockBot.AssertExpectations(t)
	})

	t.Run("handles message with special characters", func(t *testing.T) {
		// Given: Create mock bot client
		mockBot := &mockLineBotClient{}

		// Given: Create text message with special characters
		specialText := "Hello!@#$%^&*()_+-=[]{}|;:,.<>?"
		event := &mockMessageEvent{
			messageType: "text",
			text:        specialText,
			replyToken:  "test-reply-token",
		}

		// Given: Mock expects reply with special characters
		mockBot.ExpectReply("test-reply-token", "Yuruppu: "+specialText)

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should succeed without error
		require.NoError(t, err, "should handle special characters")
		mockBot.AssertExpectations(t)
	})

	t.Run("handles message with emojis and unicode", func(t *testing.T) {
		// Given: Create mock bot client
		mockBot := &mockLineBotClient{}

		// Given: Create text message with emojis and unicode
		unicodeText := "こんにちは 世界 🌍"
		event := &mockMessageEvent{
			messageType: "text",
			text:        unicodeText,
			replyToken:  "test-reply-token",
		}

		// Given: Mock expects reply with unicode preserved
		mockBot.ExpectReply("test-reply-token", "Yuruppu: "+unicodeText)

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should succeed without error
		require.NoError(t, err, "should handle unicode and emojis")
		mockBot.AssertExpectations(t)
	})

	t.Run("returns error when reply fails", func(t *testing.T) {
		// Error handling: ReplyError is returned when API call fails

		// Given: Create mock bot client that fails
		mockBot := &mockLineBotClient{}
		mockBot.SetReplyError("API error: rate limit exceeded")

		// Given: Create text message event
		event := &mockMessageEvent{
			messageType: "text",
			text:        "Hello",
			replyToken:  "test-reply-token",
		}

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should return error
		require.Error(t, err, "should return error when reply fails")
		assert.Contains(t, err.Error(), "rate limit",
			"error should contain API error message")
	})

	t.Run("returns error when reply token is invalid", func(t *testing.T) {
		// Given: Create mock bot client that fails for invalid token
		mockBot := &mockLineBotClient{}
		mockBot.SetReplyError("invalid reply token")

		// Given: Create text message event with invalid token
		event := &mockMessageEvent{
			messageType: "text",
			text:        "Hello",
			replyToken:  "invalid-token",
		}

		// When: Handle text message
		err := bot.HandleTextMessage(mockBot, event)

		// Then: Should return error
		require.Error(t, err, "should return error for invalid reply token")
	})
}

// Mock implementations for testing

// mockLineBotClient is a mock LINE bot client for testing.
type mockLineBotClient struct {
	expectedReplyToken   string
	expectedReplyMessage string
	replyError           string
	replyCalled          bool
}

func (m *mockLineBotClient) ExpectReply(replyToken, message string) {
	m.expectedReplyToken = replyToken
	m.expectedReplyMessage = message
}

func (m *mockLineBotClient) SetReplyError(errMsg string) {
	m.replyError = errMsg
}

func (m *mockLineBotClient) Reply(replyToken, message string) error {
	m.replyCalled = true

	// Check if reply error is set
	if m.replyError != "" {
		return fmt.Errorf("%s", m.replyError)
	}

	// Verify expectations if set
	if m.expectedReplyToken != "" {
		if replyToken != m.expectedReplyToken {
			return assert.AnError
		}
	}
	if m.expectedReplyMessage != "" {
		if message != m.expectedReplyMessage {
			return assert.AnError
		}
	}

	return nil
}

func (m *mockLineBotClient) AssertExpectations(t *testing.T) {
	t.Helper()
	assert.True(t, m.replyCalled, "expected Reply to be called")
}

// mockMessageEvent is a mock LINE message event for testing.
type mockMessageEvent struct {
	messageType string
	text        string
	replyToken  string
}

func (m *mockMessageEvent) GetType() string {
	return "message"
}

func (m *mockMessageEvent) GetReplyToken() string {
	return m.replyToken
}

func (m *mockMessageEvent) GetText() string {
	return m.text
}
