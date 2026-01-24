package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// JoinHandler handles join and member events.
type JoinHandler interface {
	HandleJoin(ctx context.Context) error
	HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error
	HandleMemberLeft(ctx context.Context, leftUserIDs []string) error
}

func (s *Server) invokeJoin(handler JoinHandler, joinEvent webhook.JoinEvent) {
	chatType, sourceID, userID := extractSourceInfo(joinEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("join handler panicked",
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

	err := handler.HandleJoin(ctx)
	if err != nil {
		s.logger.Error("join handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}
