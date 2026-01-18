package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Handler handles incoming LINE messages by type.
// Each method receives a context with timeout and LINE-specific values
// (reply token, source ID, user ID) accessible via context accessor functions.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type Handler interface {
	HandleFollow(ctx context.Context) error
	HandleJoin(ctx context.Context) error
	HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
	HandleMemberLeft(ctx context.Context, leftUserIDs []string) error
	HandleText(ctx context.Context, text string) error
	HandleImage(ctx context.Context, messageID string) error
	HandleSticker(ctx context.Context, packageID, stickerID string) error
	HandleVideo(ctx context.Context, messageID string) error
	HandleAudio(ctx context.Context, messageID string) error
	HandleLocation(ctx context.Context, latitude, longitude float64) error
	HandleUnknown(ctx context.Context) error
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
		case webhook.FollowEvent:
			s.dispatchFollow(e)
		case webhook.JoinEvent:
			s.dispatchJoin(e)
		case webhook.MemberJoinedEvent:
			s.dispatchMemberJoined(e)
		case webhook.MemberLeftEvent:
			s.dispatchMemberLeft(e)
		case webhook.MessageEvent:
			s.dispatchMessage(e)
		}
	}
}

// extractSourceInfo returns (chatType, sourceID, userID).
func extractSourceInfo(source webhook.SourceInterface) (line.ChatType, string, string) {
	if source == nil {
		return "", "", ""
	}
	switch s := source.(type) {
	case webhook.UserSource:
		return line.ChatTypeOneOnOne, s.UserId, s.UserId
	case webhook.GroupSource:
		return line.ChatTypeGroup, s.GroupId, s.UserId
	case webhook.RoomSource:
		return line.ChatTypeGroup, s.RoomId, s.UserId
	}
	return "", "", ""
}
