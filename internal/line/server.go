package line

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// defaultCallbackTimeout is the default timeout for callback execution.
// This matches the LLM timeout from the spec (30 seconds).
const defaultCallbackTimeout = 30 * time.Second

// ConfigError represents an error related to missing or invalid configuration.
type ConfigError struct {
	Variable string
}

func (e *ConfigError) Error() string {
	return "Missing required configuration: " + e.Variable
}

// Server handles incoming LINE webhook requests and dispatches callbacks.
type Server struct {
	channelSecret   string
	callback        MessageHandler
	callbackTimeout time.Duration
	logger          *slog.Logger
}

// NewServer creates a new LINE webhook server.
// channelSecret is the LINE channel secret for signature verification.
// Returns an error if channelSecret is empty.
func NewServer(channelSecret string) (*Server, error) {
	channelSecret = strings.TrimSpace(channelSecret)
	if channelSecret == "" {
		return nil, &ConfigError{Variable: "channelSecret"}
	}

	return &Server{
		channelSecret:   channelSecret,
		callbackTimeout: defaultCallbackTimeout,
	}, nil
}

// SetLogger sets the logger for the server.
// If nil, no logging is performed.
func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// SetCallbackTimeout sets the timeout for callback execution.
// The callback context will have this timeout set.
func (s *Server) SetCallbackTimeout(timeout time.Duration) {
	s.callbackTimeout = timeout
}

// OnMessage registers a callback to be invoked for each incoming message.
// The callback is invoked asynchronously in a goroutine after HTTP 200 is returned.
func (s *Server) OnMessage(callback MessageHandler) {
	s.callback = callback
}

// HandleWebhook processes incoming LINE webhook requests.
// Signature is verified synchronously.
// Events are parsed synchronously.
// HTTP 200 is returned synchronously.
// Callbacks are invoked asynchronously in goroutines.
func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse webhook request using LINE SDK (includes signature verification)
	cb, err := webhook.ParseRequest(s.channelSecret, r)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("webhook parsing failed",
				slog.Any("error", err),
			)
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Return HTTP 200 OK immediately (before callback execution)
	w.WriteHeader(http.StatusOK)

	// Process each event asynchronously
	for _, event := range cb.Events {
		// Handle message events only
		var msgEvent *webhook.MessageEvent
		if me, ok := event.(*webhook.MessageEvent); ok {
			msgEvent = me
		} else if me, ok := event.(webhook.MessageEvent); ok {
			msgEvent = &me
		}

		if msgEvent != nil {
			msg := extractMessage(msgEvent)
			s.invokeCallback(msg)
		}
	}
}

// extractMessage converts a LINE webhook MessageEvent to a line.Message.
func extractMessage(msgEvent *webhook.MessageEvent) Message {
	msg := Message{
		ReplyToken: msgEvent.ReplyToken,
	}

	// Extract user ID from source
	if msgEvent.Source != nil {
		if userSource, ok := msgEvent.Source.(*webhook.UserSource); ok {
			msg.UserID = userSource.UserId
		} else if userSource, ok := msgEvent.Source.(webhook.UserSource); ok {
			msg.UserID = userSource.UserId
		}
	}

	// Extract message type and text
	switch content := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		msg.Type = "text"
		msg.Text = content.Text
	case *webhook.TextMessageContent:
		msg.Type = "text"
		msg.Text = content.Text
	case webhook.ImageMessageContent, *webhook.ImageMessageContent:
		msg.Type = "image"
		msg.Text = "[User sent an image]"
	case webhook.StickerMessageContent, *webhook.StickerMessageContent:
		msg.Type = "sticker"
		msg.Text = "[User sent a sticker]"
	case webhook.VideoMessageContent, *webhook.VideoMessageContent:
		msg.Type = "video"
		msg.Text = "[User sent a video]"
	case webhook.AudioMessageContent, *webhook.AudioMessageContent:
		msg.Type = "audio"
		msg.Text = "[User sent an audio]"
	case webhook.LocationMessageContent, *webhook.LocationMessageContent:
		msg.Type = "location"
		msg.Text = "[User sent a location]"
	default:
		msg.Type = "unknown"
		msg.Text = "[User sent a message]"
	}

	return msg
}

// invokeCallback invokes the registered callback asynchronously.
// Each callback runs in its own goroutine with panic recovery.
func (s *Server) invokeCallback(msg Message) {
	if s.callback == nil {
		return
	}

	go func() {
		// Panic recovery (AC-008)
		defer func() {
			if r := recover(); r != nil {
				if s.logger != nil {
					s.logger.Error("callback panicked",
						slog.String("replyToken", msg.ReplyToken),
						slog.String("userID", msg.UserID),
						slog.Any("panic", r),
					)
				}
			}
		}()

		// Create context with timeout (not from HTTP request context)
		ctx, cancel := context.WithTimeout(context.Background(), s.callbackTimeout)
		defer cancel()

		// Invoke callback
		if err := s.callback(ctx, msg); err != nil {
			if s.logger != nil {
				s.logger.Error("callback failed",
					slog.String("replyToken", msg.ReplyToken),
					slog.String("userID", msg.UserID),
					slog.Any("error", err),
				)
			}
		}
	}()
}
