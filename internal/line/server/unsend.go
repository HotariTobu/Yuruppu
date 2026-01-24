package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// UnsendHandler handles LINE unsend events.
type UnsendHandler interface {
	HandleUnsend(ctx context.Context, messageID string) error
}

func (s *Server) invokeUnsendHandler(handler UnsendHandler, unsendEvent webhook.UnsendEvent) {
	chatType, sourceID, userID := extractSourceInfo(unsendEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("unsend handler panicked",
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

	err := handler.HandleUnsend(ctx, unsendEvent.Unsend.MessageId)
	if err != nil {
		s.logger.Error("unsend handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}
