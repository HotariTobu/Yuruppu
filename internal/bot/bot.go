package bot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"yuruppu/internal/llm"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
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

// Logger is an interface for logging operations.
// This allows injecting different logger implementations (e.g., for testing).
type Logger interface {
	Info(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// Package-level bot instance for HandleWebhook
var (
	defaultBotMu sync.RWMutex
	defaultBot   *Bot
	loggerMu     sync.RWMutex
	logger       Logger
)

// SetLogger sets the package-level logger instance for HandleWebhook.
// This function is safe for concurrent use.
func SetLogger(l Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

// getLogger returns the package-level logger instance.
// This function is safe for concurrent use.
func getLogger() Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return logger
}

// SetDefaultBot sets the package-level bot instance for HandleWebhook.
// This function is safe for concurrent use.
func SetDefaultBot(b *Bot) {
	defaultBotMu.Lock()
	defer defaultBotMu.Unlock()
	defaultBot = b
}

// getDefaultBot returns the package-level bot instance.
// This function is safe for concurrent use.
func getDefaultBot() *Bot {
	defaultBotMu.RLock()
	defer defaultBotMu.RUnlock()
	return defaultBot
}

// LineBotClient is an interface for LINE bot client operations.
// This allows mocking in tests while using the real client in production.
type LineBotClient interface {
	Reply(replyToken, message string) error
}

// MessageSender is an interface for LINE message sending.
// This allows mocking in tests while using the real client in production.
// ADR: 20251217-testing-strategy.md
type MessageSender interface {
	ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error)
}

// Package-level message sender for HandleWebhook
var (
	messageSenderMu sync.RWMutex
	messageSender   MessageSender
)

// LLMProvider is an interface for LLM operations.
// This allows mocking in tests while using the real client in production.
// TR-002: Abstraction layer for LLM providers
type LLMProvider interface {
	GenerateText(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// Package-level LLM provider for HandleWebhook
var (
	llmProviderMu sync.RWMutex
	llmProvider   LLMProvider
)

// SetDefaultLLMProvider sets the package-level LLM provider for HandleWebhook.
// This function is safe for concurrent use.
// FR-003: LLM provider must be set during initialization
func SetDefaultLLMProvider(p LLMProvider) {
	llmProviderMu.Lock()
	defer llmProviderMu.Unlock()
	llmProvider = p
}

// getDefaultLLMProvider returns the package-level LLM provider.
// This function is safe for concurrent use.
func getDefaultLLMProvider() LLMProvider {
	llmProviderMu.RLock()
	defer llmProviderMu.RUnlock()
	return llmProvider
}

// SetDefaultMessageSender sets the package-level message sender for HandleWebhook.
// This function is safe for concurrent use.
func SetDefaultMessageSender(s MessageSender) {
	messageSenderMu.Lock()
	defer messageSenderMu.Unlock()
	messageSender = s
}

// getDefaultMessageSender returns the package-level message sender.
// This function is safe for concurrent use.
func getDefaultMessageSender() MessageSender {
	messageSenderMu.RLock()
	defer messageSenderMu.RUnlock()
	return messageSender
}

// MessageEvent is an interface for LINE message events.
// This allows mocking in tests while using the real event in production.
type MessageEvent interface {
	GetType() string
	GetReplyToken() string
}

// realLineBotClient wraps the real LINE bot client to implement LineBotClient interface.
type realLineBotClient struct {
	bot *Bot
}

func (r *realLineBotClient) Reply(replyToken, message string) error {
	// Create text message
	textMessage := messaging_api.TextMessage{
		Text: message,
	}

	// Create reply request
	request := messaging_api.ReplyMessageRequest{
		ReplyToken: replyToken,
		Messages: []messaging_api.MessageInterface{
			textMessage,
		},
	}

	// Send reply
	_, err := r.bot.client.ReplyMessage(&request)
	return err
}

// realMessageEvent wraps the LINE SDK webhook event to implement MessageEvent interface.
type realMessageEvent struct {
	replyToken string
	text       string
}

func (r *realMessageEvent) GetType() string {
	return "text"
}

func (r *realMessageEvent) GetReplyToken() string {
	return r.replyToken
}

func (r *realMessageEvent) GetText() string {
	return r.text
}

// FormatEchoMessage formats a message with the Yuruppu prefix.
// message is the original user message.
// Returns the formatted echo message.
func FormatEchoMessage(message string) string {
	return "Yuruppu: " + message
}

// TextMessageEvent extends MessageEvent to provide access to text content.
type TextMessageEvent interface {
	MessageEvent
	GetText() string
}

// HandleTextMessage processes a text message event and sends an echo reply.
// client is the LINE bot client (can be mock or real).
// event is the message event from LINE (can be mock or real).
// Returns any error encountered during reply.
func HandleTextMessage(client interface{}, event interface{}) error {
	// Extract message text and reply token from event
	var text string
	var replyToken string

	// Try to get text and reply token from event
	if textEvent, ok := event.(TextMessageEvent); ok {
		text = textEvent.GetText()
		replyToken = textEvent.GetReplyToken()
	} else if msgEvent, ok := event.(MessageEvent); ok {
		// If it's only a MessageEvent, get reply token
		replyToken = msgEvent.GetReplyToken()
		// Try to get text from realMessageEvent
		if realEvent, ok := event.(*realMessageEvent); ok {
			text = realEvent.text
		}
	}

	// Format the echo message
	formattedMessage := FormatEchoMessage(text)

	// Define interface for reply method
	type replyMethod interface {
		Reply(replyToken, message string) error
	}

	// Send reply using the client interface
	if lineBotClient, ok := client.(replyMethod); ok {
		return lineBotClient.Reply(replyToken, formattedMessage)
	}

	return fmt.Errorf("unsupported client type")
}

// logIncomingMessage logs an incoming message event at INFO level.
// NFR-002: Log all incoming messages at INFO level including:
// timestamp, user ID, message type, and message text (for text messages).
func logIncomingMessage(msgEvent *webhook.MessageEvent) {
	logger := getLogger()
	if logger == nil {
		return
	}

	// Extract timestamp from event (already in milliseconds)
	timestamp := msgEvent.Timestamp

	// Extract user ID from source
	userId := ""
	if msgEvent.Source != nil {
		// Try both pointer and non-pointer types
		if userSource, ok := msgEvent.Source.(*webhook.UserSource); ok {
			userId = userSource.UserId
		} else if userSource, ok := msgEvent.Source.(webhook.UserSource); ok {
			userId = userSource.UserId
		}
	}

	// Extract message type and text (if text message)
	var messageType string
	var text string

	switch msg := msgEvent.Message.(type) {
	case *webhook.TextMessageContent:
		messageType = "text"
		text = msg.Text
	case webhook.TextMessageContent:
		messageType = "text"
		text = msg.Text
	case *webhook.ImageMessageContent:
		messageType = "image"
	case webhook.ImageMessageContent:
		messageType = "image"
	case *webhook.StickerMessageContent:
		messageType = "sticker"
	case webhook.StickerMessageContent:
		messageType = "sticker"
	case *webhook.VideoMessageContent:
		messageType = "video"
	case webhook.VideoMessageContent:
		messageType = "video"
	case *webhook.AudioMessageContent:
		messageType = "audio"
	case webhook.AudioMessageContent:
		messageType = "audio"
	case *webhook.LocationMessageContent:
		messageType = "location"
	case webhook.LocationMessageContent:
		messageType = "location"
	default:
		messageType = "unknown"
	}

	// Log with structured format
	if messageType == "text" {
		// For text messages, include the text field
		logger.Info("timestamp=%d userId=%s messageType=%s text=%s",
			timestamp, userId, messageType, text)
	} else {
		// For non-text messages, omit the text field
		logger.Info("timestamp=%d userId=%s messageType=%s",
			timestamp, userId, messageType)
	}
}

// HandleWebhook processes incoming LINE webhook requests.
// w is the HTTP response writer.
// r is the HTTP request containing the webhook payload.
// Returns HTTP 200 on success, 400 on invalid payload, 401 on invalid signature.
func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Use default bot if not set (for testing)
	bot := getDefaultBot()
	if bot == nil {
		// Try to create a bot from environment for production use
		// For now, fail with 401 if no bot is configured
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Verify signature
	if !bot.VerifySignature(r) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook request using LINE SDK
	cb, err := webhook.ParseRequest(bot.channelSecret, r)
	if err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process each event
	for _, event := range cb.Events {
		// Handle message events only
		// Try both pointer and non-pointer types
		var msgEvent *webhook.MessageEvent
		if me, ok := event.(*webhook.MessageEvent); ok {
			msgEvent = me
		} else if me, ok := event.(webhook.MessageEvent); ok {
			msgEvent = &me
		}

		if msgEvent != nil {
			// Log incoming message (NFR-002)
			logIncomingMessage(msgEvent)

			// Determine user message based on message type (FR-001, FR-008)
			// FR-008: For non-text messages, use format "[User sent a {type}]"
			var userMessage string
			var shouldProcess bool

			switch msg := msgEvent.Message.(type) {
			case *webhook.TextMessageContent:
				userMessage = msg.Text
				shouldProcess = true
			case webhook.TextMessageContent:
				userMessage = msg.Text
				shouldProcess = true
			case *webhook.ImageMessageContent:
				userMessage = "[User sent an image]"
				shouldProcess = true
			case webhook.ImageMessageContent:
				userMessage = "[User sent an image]"
				shouldProcess = true
			case *webhook.StickerMessageContent:
				userMessage = "[User sent a sticker]"
				shouldProcess = true
			case webhook.StickerMessageContent:
				userMessage = "[User sent a sticker]"
				shouldProcess = true
			case *webhook.VideoMessageContent:
				userMessage = "[User sent a video]"
				shouldProcess = true
			case webhook.VideoMessageContent:
				userMessage = "[User sent a video]"
				shouldProcess = true
			case *webhook.AudioMessageContent:
				userMessage = "[User sent an audio]"
				shouldProcess = true
			case webhook.AudioMessageContent:
				userMessage = "[User sent an audio]"
				shouldProcess = true
			case *webhook.LocationMessageContent:
				userMessage = "[User sent a location]"
				shouldProcess = true
			case webhook.LocationMessageContent:
				userMessage = "[User sent a location]"
				shouldProcess = true
			default:
				// Unknown message type, skip processing
				shouldProcess = false
			}

			if shouldProcess {
				// Get LLM provider (FR-001, FR-002)
				llmProvider := getDefaultLLMProvider()
				if llmProvider == nil {
					// No LLM provider configured, skip reply
					log.Printf("No LLM provider configured, skipping reply")
					continue
				}

				// NFR-002: Log LLM request at DEBUG level
				if l := getLogger(); l != nil {
					l.Debug("llm_request systemPrompt=%q userMessage=%q", llm.SystemPrompt, userMessage)
				}

				// Call LLM to generate response (FR-001)
				ctx := context.Background()
				response, err := llmProvider.GenerateText(ctx, llm.SystemPrompt, userMessage)
				if err != nil {
					// FR-004: On LLM API error, do not reply to the user and log the error
					log.Printf("LLM API error: %v", err)
					continue
				}

				// NFR-002: Log LLM response at DEBUG level
				if l := getLogger(); l != nil {
					l.Debug("llm_response generatedText=%q", response)
				}

				// Get message sender (use injected mock in tests, real client in production)
				sender := getDefaultMessageSender()
				if sender == nil {
					// Fallback to real client if no sender is set
					sender = bot.client
				}

				// Create reply request with LLM response (FR-002, FR-006: no "Yuruppu: " prefix)
				request := &messaging_api.ReplyMessageRequest{
					ReplyToken: msgEvent.ReplyToken,
					Messages: []messaging_api.MessageInterface{
						messaging_api.TextMessage{
							Text: response,
						},
					},
				}

				// Send reply using MessageSender interface
				if _, err := sender.ReplyMessage(request); err != nil {
					// Log error but return 200 to prevent LINE from retrying
					log.Printf("Failed to send reply: %v", err)
				}
			}
		}
	}

	// Return 200 OK
	w.WriteHeader(http.StatusOK)
}
