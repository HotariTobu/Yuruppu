package line

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

type MessageContext struct {
	ReplyToken string
	SourceID   string
	UserID     string
}

// Handler handles incoming LINE messages by type.
// Each method receives a context with timeout and message-specific parameters.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type Handler interface {
	HandleText(ctx context.Context, msgCtx MessageContext, text string) error
	HandleImage(ctx context.Context, msgCtx MessageContext, messageID string) error
	HandleSticker(ctx context.Context, msgCtx MessageContext, packageID, stickerID string) error
	HandleVideo(ctx context.Context, msgCtx MessageContext, messageID string) error
	HandleAudio(ctx context.Context, msgCtx MessageContext, messageID string) error
	HandleLocation(ctx context.Context, msgCtx MessageContext, latitude, longitude float64) error
	HandleUnknown(ctx context.Context, msgCtx MessageContext) error
}

// Server handles incoming LINE webhook requests and dispatches to handlers.
type Server struct {
	channelSecret  string
	handlers       []Handler
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
		return nil, errors.New("missing required configuration: channelSecret")
	}

	if timeout <= 0 {
		return nil, errors.New("missing required configuration: timeout")
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

// dispatchMessage dispatches the message event to all registered handlers.
// Each handler runs asynchronously in its own goroutine with panic recovery.
func (s *Server) dispatchMessage(msgEvent webhook.MessageEvent) {
	if len(s.handlers) == 0 {
		return
	}

	sourceID, userID := extractSourceIDs(msgEvent.Source)
	msgCtx := MessageContext{
		ReplyToken: msgEvent.ReplyToken,
		SourceID:   sourceID,
		UserID:     userID,
	}

	for _, handler := range s.handlers {
		go s.invokeHandler(handler, msgCtx, msgEvent)
	}
}

// extractSourceIDs extracts source ID and user ID from a webhook source.
// Returns (sourceID, userID) where:
//   - sourceID: user ID for 1:1 chats, group ID for groups, room ID for rooms
//   - userID: the user ID for all source types
func extractSourceIDs(source webhook.SourceInterface) (string, string) {
	if source == nil {
		return "", ""
	}
	switch s := source.(type) {
	case webhook.UserSource:
		return s.UserId, s.UserId
	case webhook.GroupSource:
		return s.GroupId, s.UserId
	case webhook.RoomSource:
		return s.RoomId, s.UserId
	}
	return "", ""
}

// invokeHandler invokes a single handler with panic recovery.
func (s *Server) invokeHandler(handler Handler, msgCtx MessageContext, msgEvent webhook.MessageEvent) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("handler panicked",
				slog.Any("msgCtx", msgCtx),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	var err error
	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		err = handler.HandleText(ctx, msgCtx, msg.Text)
	case webhook.ImageMessageContent:
		err = handler.HandleImage(ctx, msgCtx, msg.Id)
	case webhook.StickerMessageContent:
		err = handler.HandleSticker(ctx, msgCtx, msg.PackageId, msg.StickerId)
	case webhook.VideoMessageContent:
		err = handler.HandleVideo(ctx, msgCtx, msg.Id)
	case webhook.AudioMessageContent:
		err = handler.HandleAudio(ctx, msgCtx, msg.Id)
	case webhook.LocationMessageContent:
		err = handler.HandleLocation(ctx, msgCtx, msg.Latitude, msg.Longitude)
	default:
		err = handler.HandleUnknown(ctx, msgCtx)
	}

	if err != nil {
		s.logger.Error("handler failed",
			slog.Any("msgCtx", msgCtx),
			slog.Any("error", err),
		)
	}
}
