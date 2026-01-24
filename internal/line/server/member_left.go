package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func (s *Server) invokeMemberLeftHandler(handler JoinHandler, event webhook.MemberLeftEvent) {
	chatType, sourceID, userID := extractSourceInfo(event.Source)

	leftUserIDs := make([]string, 0, len(event.Left.Members))
	for _, member := range event.Left.Members {
		leftUserIDs = append(leftUserIDs, member.UserId)
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("member left handler panicked",
				slog.String("sourceID", sourceID),
				slog.String("userID", userID),
				slog.Any("leftUserIDs", leftUserIDs),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	ctx = line.WithChatType(ctx, chatType)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)

	err := handler.HandleMemberLeft(ctx, leftUserIDs)
	if err != nil {
		s.logger.Error("member left handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("leftUserIDs", leftUserIDs),
			slog.Any("error", err),
		)
	}
}
