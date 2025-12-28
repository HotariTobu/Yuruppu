package line

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// ConfigError represents an error related to missing or invalid configuration.
type ConfigError struct {
	Variable string
}

func (e *ConfigError) Error() string {
	return "Missing required configuration: " + e.Variable
}

// Server handles incoming LINE webhook requests and dispatches to handlers.
type Server struct {
	channelSecret  string
	handlers       []MessageHandler
	handlerTimeout time.Duration
	logger         *slog.Logger
}

// NewServer creates a new LINE webhook server.
// channelSecret is the LINE channel secret for signature verification.
// timeout is the timeout for handler execution (must be positive).
// logger is the structured logger for the server.
// Returns an error if channelSecret is empty or timeout is not positive.
func NewServer(channelSecret string, timeout time.Duration, logger *slog.Logger) (*Server, error) {
	channelSecret = strings.TrimSpace(channelSecret)
	if channelSecret == "" {
		return nil, &ConfigError{Variable: "channelSecret"}
	}

	if timeout <= 0 {
		return nil, &ConfigError{Variable: "timeout"}
	}

	return &Server{
		channelSecret:  channelSecret,
		handlerTimeout: timeout,
		logger:         logger,
	}, nil
}

// RegisterHandler registers a message handler.
// Multiple handlers can be registered and all will be invoked for each message.
// Handler methods are invoked asynchronously in goroutines after HTTP 200 is returned.
func (s *Server) RegisterHandler(handler MessageHandler) {
	s.handlers = append(s.handlers, handler)
}

// HandleWebhook processes incoming LINE webhook requests.
// Signature is verified synchronously.
// Events are parsed synchronously.
// HTTP 200 is returned synchronously.
// Handler methods are invoked asynchronously in goroutines.
func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse webhook request using LINE SDK (includes signature verification)
	cb, err := webhook.ParseRequest(s.channelSecret, r)
	if err != nil {
		s.logger.Error("webhook parsing failed",
			slog.Any("error", err),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Return HTTP 200 OK immediately (before handler execution)
	w.WriteHeader(http.StatusOK)

	// Process each event asynchronously
	for _, event := range cb.Events {
		if msgEvent, ok := event.(webhook.MessageEvent); ok {
			s.dispatchMessage(msgEvent)
		}
	}
}

// extractUserID extracts the user ID from a webhook source.
func extractUserID(source webhook.SourceInterface) string {
	if source == nil {
		return ""
	}
	if s, ok := source.(webhook.UserSource); ok {
		return s.UserId
	}
	return ""
}

// dispatchMessage dispatches the message event to all registered handlers.
// Each handler runs asynchronously in its own goroutine with panic recovery.
func (s *Server) dispatchMessage(msgEvent webhook.MessageEvent) {
	if len(s.handlers) == 0 {
		return
	}

	replyToken := msgEvent.ReplyToken
	userID := extractUserID(msgEvent.Source)

	for _, handler := range s.handlers {
		go s.invokeHandler(handler, msgEvent, replyToken, userID)
	}
}

// invokeHandler invokes a single handler with panic recovery.
func (s *Server) invokeHandler(handler MessageHandler, msgEvent webhook.MessageEvent, replyToken, userID string) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("handler panicked",
				slog.String("replyToken", replyToken),
				slog.String("userID", userID),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	var err error
	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		err = handler.HandleText(ctx, replyToken, userID, msg.Text)
	case webhook.ImageMessageContent:
		err = handler.HandleImage(ctx, replyToken, userID, msg.Id)
	case webhook.StickerMessageContent:
		err = handler.HandleSticker(ctx, replyToken, userID, msg.PackageId, msg.StickerId)
	case webhook.VideoMessageContent:
		err = handler.HandleVideo(ctx, replyToken, userID, msg.Id)
	case webhook.AudioMessageContent:
		err = handler.HandleAudio(ctx, replyToken, userID, msg.Id)
	case webhook.LocationMessageContent:
		err = handler.HandleLocation(ctx, replyToken, userID, msg.Latitude, msg.Longitude)
	default:
		err = handler.HandleUnknown(ctx, replyToken, userID)
	}

	if err != nil {
		s.logger.Error("handler failed",
			slog.String("replyToken", replyToken),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}
