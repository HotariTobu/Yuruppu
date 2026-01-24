package server

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Handler combines all event handler interfaces.
type Handler interface {
	FollowHandler
	JoinHandler
	MessageHandler
	UnsendHandler
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
	if logger == nil {
		return nil, errors.New("missing required configuration: logger")
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

	if len(s.handlers) == 0 {
		return
	}

	// Process each event asynchronously
	for _, event := range cb.Events {
		go s.processEvent(event)
	}
}

func (s *Server) processEvent(event webhook.EventInterface) {
	var invoker func(Handler)
	switch e := event.(type) {
	case webhook.FollowEvent:
		invoker = func(h Handler) { s.invokeFollowHandler(h, e) }
	case webhook.JoinEvent:
		invoker = func(h Handler) { s.invokeJoinHandler(h, e) }
	case webhook.MemberJoinedEvent:
		invoker = func(h Handler) { s.invokeMemberJoinedHandler(h, e) }
	case webhook.MemberLeftEvent:
		invoker = func(h Handler) { s.invokeMemberLeftHandler(h, e) }
	case webhook.MessageEvent:
		invoker = func(h Handler) { s.invokeMessageHandler(h, e) }
	case webhook.UnsendEvent:
		invoker = func(h Handler) { s.invokeUnsendHandler(h, e) }
	default:
		return
	}

	for _, handler := range s.handlers {
		go invoker(handler)
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
