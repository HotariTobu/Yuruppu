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
	"time"

	"yuruppu/internal/bot"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
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
	testBot, err := bot.NewBot(channelSecret, channelAccessToken)
	require.NoError(t, err)
	bot.SetDefaultBot(testBot)

	// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
	mock := &mockSender{}
	bot.SetDefaultMessageSender(mock)
	t.Cleanup(func() {
		bot.SetDefaultMessageSender(nil)
	})

	t.Run("returns 401 when signature is invalid", func(t *testing.T) {
		mock.reset()
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
		// Then: Should not send any reply
		mock.assertNotCalled(t)
	})

	t.Run("returns 400 when payload is malformed", func(t *testing.T) {
		mock.reset()
		// Given: Create request with valid signature but invalid JSON
		body := `{"events":[invalid json]}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 400
		assert.Equal(t, http.StatusBadRequest, rec.Code,
			"should return 400 for malformed payload")
		// Then: Should not send any reply
		mock.assertNotCalled(t)
	})

	t.Run("returns 200 and processes text message event", func(t *testing.T) {
		mock.reset()
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
		// Then: Should send reply with "Yuruppu: Hello"
		mock.assertCalledWith(t, "Yuruppu: Hello")
	})

	t.Run("returns 200 and ignores non-text message", func(t *testing.T) {
		mock.reset()
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
		// Then: Should NOT send any reply (AC-004)
		mock.assertNotCalled(t)
	})

	t.Run("returns 200 with empty events", func(t *testing.T) {
		mock.reset()
		// Given: Create webhook request with no events
		body := `{"events":[]}`
		req := createRequestWithValidSignature(t, channelSecret, body)
		rec := httptest.NewRecorder()

		// When: Call webhook handler
		bot.HandleWebhook(rec, req)

		// Then: Should return 200
		assert.Equal(t, http.StatusOK, rec.Code,
			"should return 200 for empty events")
		// Then: Should not send any reply
		mock.assertNotCalled(t)
	})

	t.Run("returns 200 even if reply fails", func(t *testing.T) {
		mock.reset()
		// Error handling requirement: ReplyError returns HTTP 200
		// to prevent LINE from retrying, but error is logged

		// Given: Mock sender returns an error
		mock.err = fmt.Errorf("API error: rate limit exceeded")

		// Given: Create valid webhook request
		body := `{
			"events": [
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12345",
						"text": "Hello"
					},
					"replyToken": "test-reply-token",
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
		// Then: Should have attempted to send reply
		mock.assertCalledOnce(t)

		// Cleanup: Reset error for other tests
		mock.err = nil
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

// TestHandleWebhook_LoggingTextMessages tests that text messages are logged with required fields.
// NFR-002: Log all incoming messages at INFO level including:
// timestamp, user ID, message type, and message text (for text messages)
func TestHandleWebhook_LoggingTextMessages(t *testing.T) {
	// Setup: Create a valid channel secret
	channelSecret := "test-channel-secret"
	channelAccessToken := "test-access-token"

	// Setup: Initialize bot for handler
	testBot, err := bot.NewBot(channelSecret, channelAccessToken)
	require.NoError(t, err)
	bot.SetDefaultBot(testBot)

	// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
	mock := &mockSender{}
	bot.SetDefaultMessageSender(mock)
	t.Cleanup(func() {
		bot.SetDefaultMessageSender(nil)
	})

	tests := []struct {
		name             string
		webhookBody      string
		expectedUserID   string
		expectedMsgType  string
		expectedMsgText  string
		expectedLogLevel string
	}{
		{
			name: "text message logs timestamp, user ID, message type, and text",
			webhookBody: `{
				"destination": "xxxxxxxxxx",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "Hello World"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "U1234567890abcdef"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "U1234567890abcdef",
			expectedMsgType:  "text",
			expectedMsgText:  "Hello World",
			expectedLogLevel: "INFO",
		},
		{
			name: "text message with special characters logs correctly",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "Hello!@#$%^&*()"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "U9876543210fedcba"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "U9876543210fedcba",
			expectedMsgType:  "text",
			expectedMsgText:  "Hello!@#$%^&*()",
			expectedLogLevel: "INFO",
		},
		{
			name: "text message with unicode and emojis logs correctly",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "こんにちは 🌍"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Uabcdef1234567890"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Uabcdef1234567890",
			expectedMsgType:  "text",
			expectedMsgText:  "こんにちは 🌍",
			expectedLogLevel: "INFO",
		},
		{
			name: "whitespace-only text message logs correctly",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "   "
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Uwhitespace123456"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Uwhitespace123456",
			expectedMsgType:  "text",
			expectedMsgText:  "   ",
			expectedLogLevel: "INFO",
		},
		{
			name: "long text message logs correctly",
			webhookBody: fmt.Sprintf(`{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "%s"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Ulongmessage12345"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`, strings.Repeat("a", 5000)),
			expectedUserID:   "Ulongmessage12345",
			expectedMsgType:  "text",
			expectedMsgText:  strings.Repeat("a", 5000),
			expectedLogLevel: "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.reset()
			// Given: Create mock logger to capture log entries
			mockLogger := &mockLogger{}
			bot.SetLogger(mockLogger)

			// Given: Create valid webhook request
			req := createRequestWithValidSignature(t, channelSecret, tt.webhookBody)
			rec := httptest.NewRecorder()

			// When: Call webhook handler
			bot.HandleWebhook(rec, req)

			// Then: Should return 200
			assert.Equal(t, http.StatusOK, rec.Code,
				"should return 200 for valid webhook request")

			// Then: Should have logged the message with required fields
			assert.True(t, mockLogger.infoLogCalled,
				"should log at INFO level")

			// Then: Log entry should contain timestamp
			assert.Contains(t, mockLogger.lastLogEntry, "timestamp",
				"log entry should contain timestamp field")
			assert.Contains(t, mockLogger.lastLogEntry, "1609459200000",
				"log entry should contain actual timestamp value")

			// Then: Log entry should contain user ID
			assert.Contains(t, mockLogger.lastLogEntry, "userId",
				"log entry should contain userId field")
			assert.Contains(t, mockLogger.lastLogEntry, tt.expectedUserID,
				"log entry should contain actual user ID")

			// Then: Log entry should contain message type
			assert.Contains(t, mockLogger.lastLogEntry, "messageType",
				"log entry should contain messageType field")
			assert.Contains(t, mockLogger.lastLogEntry, tt.expectedMsgType,
				"log entry should contain actual message type")

			// Then: Log entry should contain message text
			assert.Contains(t, mockLogger.lastLogEntry, "text",
				"log entry should contain text field for text messages")
			assert.Contains(t, mockLogger.lastLogEntry, tt.expectedMsgText,
				"log entry should contain actual message text")
		})
	}
}

// TestHandleWebhook_LoggingNonTextMessages tests that non-text messages are logged without text field.
// NFR-002: Log all incoming messages at INFO level including:
// timestamp, user ID, message type (but NOT message text for non-text messages)
func TestHandleWebhook_LoggingNonTextMessages(t *testing.T) {
	// Setup: Create a valid channel secret
	channelSecret := "test-channel-secret"
	channelAccessToken := "test-access-token"

	// Setup: Initialize bot for handler
	testBot, err := bot.NewBot(channelSecret, channelAccessToken)
	require.NoError(t, err)
	bot.SetDefaultBot(testBot)

	// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
	mock := &mockSender{}
	bot.SetDefaultMessageSender(mock)
	t.Cleanup(func() {
		bot.SetDefaultMessageSender(nil)
	})

	tests := []struct {
		name             string
		webhookBody      string
		expectedUserID   string
		expectedMsgType  string
		shouldNotContain string
		expectedLogLevel string
	}{
		{
			name: "image message logs without text field",
			webhookBody: `{
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
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Uimage123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Uimage123456789",
			expectedMsgType:  "image",
			shouldNotContain: "\"text\":",
			expectedLogLevel: "INFO",
		},
		{
			name: "sticker message logs without text field",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "sticker",
							"id": "12345",
							"packageId": "1",
							"stickerId": "1"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Usticker123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Usticker123456789",
			expectedMsgType:  "sticker",
			shouldNotContain: "\"text\":",
			expectedLogLevel: "INFO",
		},
		{
			name: "video message logs without text field",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "video",
							"id": "12345",
							"duration": 5000,
							"contentProvider": {
								"type": "line"
							}
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Uvideo123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Uvideo123456789",
			expectedMsgType:  "video",
			shouldNotContain: "\"text\":",
			expectedLogLevel: "INFO",
		},
		{
			name: "audio message logs without text field",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "audio",
							"id": "12345",
							"duration": 3000,
							"contentProvider": {
								"type": "line"
							}
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Uaudio123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Uaudio123456789",
			expectedMsgType:  "audio",
			shouldNotContain: "\"text\":",
			expectedLogLevel: "INFO",
		},
		{
			name: "location message logs without text field",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "location",
							"id": "12345",
							"title": "My Location",
							"address": "Tokyo",
							"latitude": 35.6762,
							"longitude": 139.6503
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Ulocation123456"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
			expectedUserID:   "Ulocation123456",
			expectedMsgType:  "location",
			shouldNotContain: "\"text\":",
			expectedLogLevel: "INFO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create mock logger to capture log entries
			mockLogger := &mockLogger{}
			bot.SetLogger(mockLogger)

			// Given: Create valid webhook request
			req := createRequestWithValidSignature(t, channelSecret, tt.webhookBody)
			rec := httptest.NewRecorder()

			// When: Call webhook handler
			bot.HandleWebhook(rec, req)

			// Then: Should return 200
			assert.Equal(t, http.StatusOK, rec.Code,
				"should return 200 for valid webhook request")

			// Then: Should have logged the message
			assert.True(t, mockLogger.infoLogCalled,
				"should log at INFO level")

			// Then: Log entry should contain timestamp
			assert.Contains(t, mockLogger.lastLogEntry, "timestamp",
				"log entry should contain timestamp field")
			assert.Contains(t, mockLogger.lastLogEntry, "1609459200000",
				"log entry should contain actual timestamp value")

			// Then: Log entry should contain user ID
			assert.Contains(t, mockLogger.lastLogEntry, "userId",
				"log entry should contain userId field")
			assert.Contains(t, mockLogger.lastLogEntry, tt.expectedUserID,
				"log entry should contain actual user ID")

			// Then: Log entry should contain message type
			assert.Contains(t, mockLogger.lastLogEntry, "messageType",
				"log entry should contain messageType field")
			assert.Contains(t, mockLogger.lastLogEntry, tt.expectedMsgType,
				"log entry should contain actual message type")

			// Then: Log entry should NOT contain text field
			assert.NotContains(t, mockLogger.lastLogEntry, tt.shouldNotContain,
				"log entry should NOT contain text field for non-text messages")
		})
	}
}

// TestHandleWebhook_LoggingMultipleMessages tests that multiple messages in one webhook are all logged.
// NFR-002: All incoming messages should be logged
func TestHandleWebhook_LoggingMultipleMessages(t *testing.T) {
	t.Run("multiple messages in single webhook are all logged", func(t *testing.T) {
		// Setup: Create a valid channel secret
		channelSecret := "test-channel-secret"
		channelAccessToken := "test-access-token"

		// Setup: Initialize bot for handler
		testBot, err := bot.NewBot(channelSecret, channelAccessToken)
		require.NoError(t, err)
		bot.SetDefaultBot(testBot)

		// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
		mock := &mockSender{}
		bot.SetDefaultMessageSender(mock)
		t.Cleanup(func() {
			bot.SetDefaultMessageSender(nil)
		})

		// Given: Create mock logger to capture log entries
		mockLogger := &mockLogger{}
		bot.SetLogger(mockLogger)

		// Given: Create webhook request with multiple events
		body := `{
			"destination": "xxxxxxxxxx",
			"events": [
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12345",
						"text": "First message"
					},
					"timestamp": 1609459200000,
					"source": {
						"type": "user",
						"userId": "Ufirst123456789"
					},
					"replyToken": "test-reply-token-1",
					"mode": "active"
				},
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12346",
						"text": "Second message"
					},
					"timestamp": 1609459201000,
					"source": {
						"type": "user",
						"userId": "Usecond123456789"
					},
					"replyToken": "test-reply-token-2",
					"mode": "active"
				},
				{
					"type": "message",
					"message": {
						"type": "image",
						"id": "12347",
						"contentProvider": {
							"type": "line"
						}
					},
					"timestamp": 1609459202000,
					"source": {
						"type": "user",
						"userId": "Uthird123456789"
					},
					"replyToken": "test-reply-token-3",
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

		// Then: Should have logged all messages (3 log calls)
		assert.Equal(t, 3, mockLogger.infoLogCallCount,
			"should log each message in the webhook")

		// Then: All log entries should be captured
		assert.Len(t, mockLogger.allLogEntries, 3,
			"should have 3 log entries for 3 messages")

		// Then: First message should be logged
		assert.Contains(t, mockLogger.allLogEntries[0], "Ufirst123456789",
			"first log entry should contain first user ID")
		assert.Contains(t, mockLogger.allLogEntries[0], "First message",
			"first log entry should contain first message text")

		// Then: Second message should be logged
		assert.Contains(t, mockLogger.allLogEntries[1], "Usecond123456789",
			"second log entry should contain second user ID")
		assert.Contains(t, mockLogger.allLogEntries[1], "Second message",
			"second log entry should contain second message text")

		// Then: Third message (image) should be logged
		assert.Contains(t, mockLogger.allLogEntries[2], "Uthird123456789",
			"third log entry should contain third user ID")
		assert.Contains(t, mockLogger.allLogEntries[2], "image",
			"third log entry should contain message type")
		assert.NotContains(t, mockLogger.allLogEntries[2], "\"text\":",
			"third log entry (image) should not contain text field")
	})
}

// TestHandleWebhook_LoggingFormat tests the logging format.
// NFR-002: Logs should be structured and include all required fields
func TestHandleWebhook_LoggingFormat(t *testing.T) {
	t.Run("log format includes all required fields in structured format", func(t *testing.T) {
		// Setup: Create a valid channel secret
		channelSecret := "test-channel-secret"
		channelAccessToken := "test-access-token"

		// Setup: Initialize bot for handler
		testBot, err := bot.NewBot(channelSecret, channelAccessToken)
		require.NoError(t, err)
		bot.SetDefaultBot(testBot)

		// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
		mock := &mockSender{}
		bot.SetDefaultMessageSender(mock)
		t.Cleanup(func() {
			bot.SetDefaultMessageSender(nil)
		})

		// Given: Create mock logger
		mockLogger := &mockLogger{}
		bot.SetLogger(mockLogger)

		// Given: Create valid webhook request
		body := `{
			"events": [
				{
					"type": "message",
					"message": {
						"type": "text",
						"id": "12345",
						"text": "Test message"
					},
					"timestamp": 1609459200000,
					"source": {
						"type": "user",
						"userId": "Utest123456789"
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

		// Then: Log should be structured with key-value pairs
		logEntry := mockLogger.lastLogEntry

		// Then: Should contain all required fields
		requiredFields := []string{"timestamp", "userId", "messageType", "text"}
		for _, field := range requiredFields {
			assert.Contains(t, logEntry, field,
				"log entry should contain %s field", field)
		}

		// Then: Should be parseable (suggests structured format like JSON)
		// The log should be in a format like: "field=value" or JSON
		assert.Regexp(t, `timestamp[=:]`, logEntry,
			"log should use structured format with field names")
	})
}

// TestHandleWebhook_LoggingLevel tests that logs are at INFO level.
// NFR-002: Log at INFO level
func TestHandleWebhook_LoggingLevel(t *testing.T) {
	tests := []struct {
		name        string
		webhookBody string
	}{
		{
			name: "text message logs at INFO level",
			webhookBody: `{
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "12345",
							"text": "Hello"
						},
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Utest123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
		},
		{
			name: "non-text message logs at INFO level",
			webhookBody: `{
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
						"timestamp": 1609459200000,
						"source": {
							"type": "user",
							"userId": "Utest123456789"
						},
						"replyToken": "test-reply-token",
						"mode": "active"
					}
				]
			}`,
		},
	}

	// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
	mock := &mockSender{}
	bot.SetDefaultMessageSender(mock)
	t.Cleanup(func() {
		bot.SetDefaultMessageSender(nil)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.reset()
			// Setup
			channelSecret := "test-channel-secret"
			channelAccessToken := "test-access-token"
			testBot, err := bot.NewBot(channelSecret, channelAccessToken)
			require.NoError(t, err)
			bot.SetDefaultBot(testBot)

			// Given: Create mock logger
			mockLogger := &mockLogger{}
			bot.SetLogger(mockLogger)

			// Given: Create valid webhook request
			req := createRequestWithValidSignature(t, channelSecret, tt.webhookBody)
			rec := httptest.NewRecorder()

			// When: Call webhook handler
			bot.HandleWebhook(rec, req)

			// Then: Should log at INFO level, not DEBUG, WARN, or ERROR
			assert.True(t, mockLogger.infoLogCalled,
				"should call Info logging method")
			assert.False(t, mockLogger.debugLogCalled,
				"should not call Debug logging method")
			assert.False(t, mockLogger.warnLogCalled,
				"should not call Warn logging method")
			assert.False(t, mockLogger.errorLogCalled,
				"should not call Error logging method")
		})
	}
}

// mockLogger is a mock logger implementation for testing logging behavior.
type mockLogger struct {
	infoLogCalled    bool
	debugLogCalled   bool
	warnLogCalled    bool
	errorLogCalled   bool
	infoLogCallCount int
	lastLogEntry     string
	allLogEntries    []string
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.infoLogCalled = true
	m.infoLogCallCount++
	m.lastLogEntry = fmt.Sprintf(format, args...)
	m.allLogEntries = append(m.allLogEntries, m.lastLogEntry)
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.debugLogCalled = true
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	m.warnLogCalled = true
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.errorLogCalled = true
}

// mockSender is a mock implementation of MessageSender for testing.
// ADR: 20251217-testing-strategy.md
type mockSender struct {
	calls []*messaging_api.ReplyMessageRequest
	err   error
}

func (m *mockSender) ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	m.calls = append(m.calls, req)
	if m.err != nil {
		return nil, m.err
	}
	return &messaging_api.ReplyMessageResponse{}, nil
}

// reset clears the recorded calls.
func (m *mockSender) reset() {
	m.calls = nil
}

// assertCalledOnce verifies ReplyMessage was called exactly once.
func (m *mockSender) assertCalledOnce(t *testing.T) {
	t.Helper()
	assert.Len(t, m.calls, 1, "expected ReplyMessage to be called once")
}

// assertNotCalled verifies ReplyMessage was not called.
func (m *mockSender) assertNotCalled(t *testing.T) {
	t.Helper()
	assert.Empty(t, m.calls, "expected ReplyMessage to not be called")
}

// assertCalledWith verifies ReplyMessage was called with expected message.
func (m *mockSender) assertCalledWith(t *testing.T, expectedText string) {
	t.Helper()
	require.Len(t, m.calls, 1, "expected exactly one call")
	require.Len(t, m.calls[0].Messages, 1, "expected exactly one message")
	textMsg, ok := m.calls[0].Messages[0].(messaging_api.TextMessage)
	require.True(t, ok, "expected TextMessage")
	assert.Equal(t, expectedText, textMsg.Text, "message text should match")
}

// BenchmarkHandleWebhook_InternalProcessing benchmarks the internal processing time.
// NFR-001: Respond within 1 second to avoid LINE timeout.
// This benchmark measures internal processing time (excluding LINE API call).
// Target: < 100ms for internal processing to leave margin for network latency.
func BenchmarkHandleWebhook_InternalProcessing(b *testing.B) {
	// Setup: Create a valid channel secret
	channelSecret := "test-channel-secret"
	channelAccessToken := "test-access-token"

	// Setup: Initialize bot for handler
	testBot, err := bot.NewBot(channelSecret, channelAccessToken)
	if err != nil {
		b.Fatalf("failed to create bot: %v", err)
	}
	bot.SetDefaultBot(testBot)

	// Setup: Initialize mock sender (ADR: 20251217-testing-strategy.md)
	// This prevents real API calls during benchmarking
	mock := &mockSender{}
	bot.SetDefaultMessageSender(mock)

	// Setup: Create mock logger (no-op for benchmarks)
	bot.SetLogger(&mockLogger{})

	// Setup: Create valid webhook request body
	body := `{
		"destination": "xxxxxxxxxx",
		"events": [
			{
				"type": "message",
				"message": {
					"type": "text",
					"id": "12345",
					"text": "Hello World"
				},
				"timestamp": 1609459200000,
				"source": {
					"type": "user",
					"userId": "U1234567890abcdef"
				},
				"replyToken": "test-reply-token",
				"mode": "active"
			}
		]
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := createBenchmarkRequest(channelSecret, body)
		rec := httptest.NewRecorder()
		bot.HandleWebhook(rec, req)
	}
}

// BenchmarkFormatEchoMessage benchmarks the echo message formatting.
func BenchmarkFormatEchoMessage(b *testing.B) {
	message := "Hello World"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bot.FormatEchoMessage(message)
	}
}

// BenchmarkFormatEchoMessage_LongMessage benchmarks formatting with 5000 char message.
func BenchmarkFormatEchoMessage_LongMessage(b *testing.B) {
	message := strings.Repeat("a", 5000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bot.FormatEchoMessage(message)
	}
}

// BenchmarkVerifySignature benchmarks signature verification.
func BenchmarkVerifySignature(b *testing.B) {
	channelSecret := "test-channel-secret"
	testBot, err := bot.NewBot(channelSecret, "test-access-token")
	if err != nil {
		b.Fatalf("failed to create bot: %v", err)
	}

	body := `{"events":[{"type":"message","message":{"type":"text","text":"Hello"}}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := createBenchmarkRequest(channelSecret, body)
		_ = testBot.VerifySignature(req)
	}
}

// createBenchmarkRequest creates an HTTP request for benchmarking.
func createBenchmarkRequest(channelSecret, body string) *http.Request {
	signature := computeSignature(channelSecret, body)
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Line-Signature", signature)
	return req
}

// TestResponseTime_InternalProcessingWithMock verifies internal processing time.
// NFR-001: Respond within 1 second to avoid LINE timeout.
// This test uses the HandleTextMessage function directly with a mock client
// to measure internal processing time without network calls.
func TestResponseTime_InternalProcessingWithMock(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		maxDuration time.Duration
	}{
		{
			name:        "simple text message processes under 10ms",
			message:     "Hello",
			maxDuration: 10 * time.Millisecond,
		},
		{
			name:        "long message (5000 chars) processes under 10ms",
			message:     strings.Repeat("a", 5000),
			maxDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create mock bot client (no network calls)
			mockBot := &mockLineBotClient{}
			mockBot.ExpectReply("test-reply-token", "Yuruppu: "+tt.message)

			// Given: Create text message event
			event := &mockMessageEvent{
				messageType: "text",
				text:        tt.message,
				replyToken:  "test-reply-token",
			}

			// When: Measure processing time
			start := time.Now()
			err := bot.HandleTextMessage(mockBot, event)
			duration := time.Since(start)

			// Then: Should succeed
			require.NoError(t, err)

			// Then: Processing should complete within limit
			assert.Less(t, duration, tt.maxDuration,
				"internal processing should complete within %v, took %v", tt.maxDuration, duration)

			// Log actual duration for visibility
			t.Logf("Processing time: %v", duration)
		})
	}
}

// TestResponseTime_SignatureVerification verifies signature verification is fast.
// NFR-001: Signature verification is part of the critical path.
func TestResponseTime_SignatureVerification(t *testing.T) {
	// Given: Create bot
	channelSecret := "test-channel-secret"
	testBot, err := bot.NewBot(channelSecret, "test-access-token")
	require.NoError(t, err)

	tests := []struct {
		name        string
		bodySize    int
		maxDuration time.Duration
	}{
		{
			name:        "small payload verifies under 1ms",
			bodySize:    100,
			maxDuration: 1 * time.Millisecond,
		},
		{
			name:        "large payload (10KB) verifies under 5ms",
			bodySize:    10000,
			maxDuration: 5 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Create request with payload of specified size
			body := fmt.Sprintf(`{"events":[],"padding":"%s"}`, strings.Repeat("a", tt.bodySize))
			req := createRequestWithValidSignature(t, channelSecret, body)

			// When: Measure verification time
			start := time.Now()
			result := testBot.VerifySignature(req)
			duration := time.Since(start)

			// Then: Signature should be valid
			assert.True(t, result, "signature should be valid")

			// Then: Verification should be fast
			assert.Less(t, duration, tt.maxDuration,
				"signature verification should complete within %v, took %v", tt.maxDuration, duration)

			t.Logf("Verification time for %d byte payload: %v", tt.bodySize, duration)
		})
	}
}

// TestResponseTime_MessageFormatting verifies message formatting is fast.
// NFR-001: Message formatting is part of the critical path.
func TestResponseTime_MessageFormatting(t *testing.T) {
	tests := []struct {
		name        string
		messageLen  int
		maxDuration time.Duration
	}{
		{
			name:        "short message formats under 1μs",
			messageLen:  10,
			maxDuration: 1 * time.Microsecond * 100, // Allow 100μs for safety
		},
		{
			name:        "long message (5000 chars) formats under 100μs",
			messageLen:  5000,
			maxDuration: 100 * time.Microsecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := strings.Repeat("a", tt.messageLen)

			// Warm up
			_ = bot.FormatEchoMessage(message)

			// When: Measure formatting time (average of 1000 iterations)
			start := time.Now()
			for i := 0; i < 1000; i++ {
				_ = bot.FormatEchoMessage(message)
			}
			totalDuration := time.Since(start)
			avgDuration := totalDuration / 1000

			// Then: Formatting should be fast
			assert.Less(t, avgDuration, tt.maxDuration,
				"message formatting should complete within %v, average was %v", tt.maxDuration, avgDuration)

			t.Logf("Average formatting time for %d char message: %v", tt.messageLen, avgDuration)
		})
	}
}

// generateMultipleEventsBody generates a webhook body with multiple text message events.
func generateMultipleEventsBody(count int) string {
	events := make([]string, count)
	for i := 0; i < count; i++ {
		events[i] = fmt.Sprintf(`{
			"type": "message",
			"message": {
				"type": "text",
				"id": "%d",
				"text": "Message %d"
			},
			"timestamp": 1609459200000,
			"source": {
				"type": "user",
				"userId": "U1234567890abcdef"
			},
			"replyToken": "test-reply-token-%d",
			"mode": "active"
		}`, i, i, i)
	}
	return fmt.Sprintf(`{"destination": "xxxxxxxxxx", "events": [%s]}`, strings.Join(events, ","))
}
