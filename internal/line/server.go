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

// Handler handles incoming LINE messages by type.
// Each method receives a context with timeout and LINE-specific values
// (reply token, source ID, user ID) accessible via context accessor functions.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type Handler interface {
	HandleText(ctx context.Context, text string) error
	HandleImage(ctx context.Context, messageID string) error
	HandleSticker(ctx context.Context, packageID, stickerID string) error
	HandleVideo(ctx context.Context, messageID string) error
	HandleAudio(ctx context.Context, messageID string) error
	HandleLocation(ctx context.Context, latitude, longitude float64) error
	HandleUnknown(ctx context.Context) error
	HandleFollow(ctx context.Context) error
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
		switch e := event.(type) {
		case webhook.MessageEvent:
			s.dispatchMessage(e)
		case webhook.FollowEvent:
			s.dispatchFollow(e)
		}
	}
}

// dispatchMessage dispatches the message event to all registered handlers.
// Each handler runs asynchronously in its own goroutine with panic recovery.
func (s *Server) dispatchMessage(msgEvent webhook.MessageEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeHandler(handler, msgEvent)
	}
}

// invokeHandler invokes a single handler with panic recovery.
func (s *Server) invokeHandler(handler Handler, msgEvent webhook.MessageEvent) {
	sourceID, userID := extractSourceIDs(msgEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("handler panicked",
				slog.String("sourceID", sourceID),
				slog.String("userID", userID),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	// Set LINE-specific values in context
	ctx = WithReplyToken(ctx, msgEvent.ReplyToken)
	ctx = WithSourceID(ctx, sourceID)
	ctx = WithUserID(ctx, userID)

	var err error
	switch msg := msgEvent.Message.(type) {
	case webhook.TextMessageContent:
		err = handler.HandleText(ctx, msg.Text)
	case webhook.ImageMessageContent:
		err = handler.HandleImage(ctx, msg.Id)
	case webhook.StickerMessageContent:
		err = handler.HandleSticker(ctx, msg.PackageId, msg.StickerId)
	case webhook.VideoMessageContent:
		err = handler.HandleVideo(ctx, msg.Id)
	case webhook.AudioMessageContent:
		err = handler.HandleAudio(ctx, msg.Id)
	case webhook.LocationMessageContent:
		err = handler.HandleLocation(ctx, msg.Latitude, msg.Longitude)
	default:
		err = handler.HandleUnknown(ctx)
	}

	if err != nil {
		s.logger.Error("handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}

// dispatchFollow dispatches the follow event to all registered handlers.
func (s *Server) dispatchFollow(followEvent webhook.FollowEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeFollowHandler(handler, followEvent)
	}
}

// invokeFollowHandler invokes a single handler for follow event with panic recovery.
func (s *Server) invokeFollowHandler(handler Handler, followEvent webhook.FollowEvent) {
	sourceID, userID := extractSourceIDs(followEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("follow handler panicked",
				slog.String("sourceID", sourceID),
				slog.String("userID", userID),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	// Set LINE-specific values in context
	ctx = WithSourceID(ctx, sourceID)
	ctx = WithUserID(ctx, userID)

	err := handler.HandleFollow(ctx)
	if err != nil {
		s.logger.Error("follow handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
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
