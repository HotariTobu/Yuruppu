package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func (s *Server) invokeMemberJoinedHandler(handler JoinHandler, event webhook.MemberJoinedEvent) {
	chatType, sourceID, userID := extractSourceInfo(event.Source)

	joinedUserIDs := make([]string, 0, len(event.Joined.Members))
	for _, member := range event.Joined.Members {
		joinedUserIDs = append(joinedUserIDs, member.UserId)
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("member joined handler panicked",
				slog.String("sourceID", sourceID),
				slog.String("userID", userID),
				slog.Any("joinedUserIDs", joinedUserIDs),
				slog.Any("panic", r),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), s.handlerTimeout)
	defer cancel()

	ctx = line.WithChatType(ctx, chatType)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)

	err := handler.HandleMemberJoined(ctx, joinedUserIDs)
	if err != nil {
		s.logger.Error("member joined handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("joinedUserIDs", joinedUserIDs),
			slog.Any("error", err),
		)
	}
}
