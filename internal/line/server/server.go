package server

import (
	// Standard library
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"yuruppu/internal/line"

	// Third-party packages
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	// Internal packages
)

// Handler handles incoming LINE messages by type.
// Each method receives a context with timeout and message-specific parameters.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type Handler interface {
	HandleText(ctx context.Context, replyToken, userID, text string) error
	HandleImage(ctx context.Context, replyToken, userID, messageID string) error
	HandleSticker(ctx context.Context, replyToken, userID, packageID, stickerID string) error
	HandleVideo(ctx context.Context, replyToken, userID, messageID string) error
	HandleAudio(ctx context.Context, replyToken, userID, messageID string) error
	HandleLocation(ctx context.Context, replyToken, userID string, latitude, longitude float64) error
	HandleUnknown(ctx context.Context, replyToken, userID string) error
}

// Server handles incoming LINE webhook requests and dispatches to handlers.
type Server struct {
	channelSecret  string
	handlers       []Handler
	handlerTimeout time.Duration
	logger         *slog.Logger
}

// New creates a new LINE webhook server.
// channelSecret is the LINE channel secret for signature verification.
// timeout is the timeout for handler execution (must be positive).
// logger is the structured logger for the server.
// Returns an error if channelSecret is empty or timeout is not positive.
func New(channelSecret string, timeout time.Duration, logger *slog.Logger) (*Server, error) {
	channelSecret = strings.TrimSpace(channelSecret)
	if channelSecret == "" {
		return nil, &line.ConfigError{Variable: "channelSecret"}
	}

	if timeout <= 0 {
		return nil, &line.ConfigError{Variable: "timeout"}
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
func (s *Server) RegisterHandler(handler Handler) {
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

// extractSourceID extracts the source ID from a webhook source.
// For 1:1 chats (UserSource), it returns the user ID.
// For group chats (GroupSource), it returns the group ID.
// For room chats (RoomSource), it returns the room ID.
func extractSourceID(source webhook.SourceInterface) string {
	if source == nil {
		return ""
	}
	switch s := source.(type) {
	case webhook.UserSource:
		return s.UserId
	case webhook.GroupSource:
		return s.GroupId
	case webhook.RoomSource:
		return s.RoomId
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
	sourceID := extractSourceID(msgEvent.Source)

	for _, handler := range s.handlers {
		go s.invokeHandler(handler, msgEvent, replyToken, sourceID)
	}
}

// invokeHandler invokes a single handler with panic recovery.
// sourceID identifies the conversation source (user ID for 1:1, group ID for groups, room ID for rooms).
func (s *Server) invokeHandler(handler Handler, msgEvent webhook.MessageEvent, replyToken, sourceID string) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("handler panicked",
				slog.String("replyToken", replyToken),
				slog.String("sourceID", sourceID),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	var err error
	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		err = handler.HandleText(ctx, replyToken, sourceID, msg.Text)
	case webhook.ImageMessageContent:
		err = handler.HandleImage(ctx, replyToken, sourceID, msg.Id)
	case webhook.StickerMessageContent:
		err = handler.HandleSticker(ctx, replyToken, sourceID, msg.PackageId, msg.StickerId)
	case webhook.VideoMessageContent:
		err = handler.HandleVideo(ctx, replyToken, sourceID, msg.Id)
	case webhook.AudioMessageContent:
		err = handler.HandleAudio(ctx, replyToken, sourceID, msg.Id)
	case webhook.LocationMessageContent:
		err = handler.HandleLocation(ctx, replyToken, sourceID, msg.Latitude, msg.Longitude)
	default:
		err = handler.HandleUnknown(ctx, replyToken, sourceID)
	}

	if err != nil {
		s.logger.Error("handler failed",
			slog.String("replyToken", replyToken),
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
	}
}
