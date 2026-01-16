package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// dispatchFollow dispatches the follow event to all registered handlers.
func (s *Server) dispatchFollow(followEvent webhook.FollowEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeFollowHandler(handler, followEvent)
	}
}

func (s *Server) invokeFollowHandler(handler Handler, followEvent webhook.FollowEvent) {
	chatType, sourceID, userID := extractSourceInfo(followEvent.Source)

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

	ctx = line.WithChatType(ctx, chatType)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)

	err := handler.HandleFollow(ctx)
	if err != nil {
		s.logger.Error("follow handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}
