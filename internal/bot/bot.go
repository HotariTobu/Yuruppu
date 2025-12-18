package bot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

// ConfigError represents an error related to missing or invalid configuration.
type ConfigError struct {
	Variable string
}

func (e *ConfigError) Error() string {
	return "Missing required environment variable: " + e.Variable
}

// Bot represents a LINE bot client.
type Bot struct {
	channelSecret string
	client        *messaging_api.MessagingApiAPI
}

// NewBot creates a new LINE bot client with the given credentials.
// channelSecret is the LINE channel secret for signature verification.
// channelAccessToken is the LINE channel access token for API calls.
// Returns the bot client or an error if initialization fails.
func NewBot(channelSecret, channelAccessToken string) (*Bot, error) {
	// Trim whitespace from credentials
	channelSecret = strings.TrimSpace(channelSecret)
	channelAccessToken = strings.TrimSpace(channelAccessToken)

	// Validate channelSecret
	if channelSecret == "" {
		return nil, &ConfigError{Variable: "LINE_CHANNEL_SECRET"}
	}

	// Validate channelAccessToken
	if channelAccessToken == "" {
		return nil, &ConfigError{Variable: "LINE_CHANNEL_ACCESS_TOKEN"}
	}

	// Create messaging API client
	client, err := messaging_api.NewMessagingApiAPI(channelAccessToken)
	if err != nil {
		return nil, err
	}

	// Create and return Bot instance
	return &Bot{
		channelSecret: channelSecret,
		client:        client,
	}, nil
}

// VerifySignature verifies the LINE webhook signature.
// Returns true if signature is valid, false otherwise.
func (b *Bot) VerifySignature(r *http.Request) bool {
	// Extract signature from header
	signature := r.Header.Get("X-Line-Signature")
	if signature == "" {
		return false
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}

	// Restore body for later use
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(b.channelSecret))
	mac.Write(body)
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Decode provided signature to validate it's valid base64
	_, err = base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Compare signatures using constant-time comparison
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
}
