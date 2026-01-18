package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// dispatchJoin dispatches the join event to all registered handlers.
func (s *Server) dispatchJoin(joinEvent webhook.JoinEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeJoinHandler(handler, joinEvent)
	}
}

func (s *Server) invokeJoinHandler(handler Handler, joinEvent webhook.JoinEvent) {
	chatType, sourceID, userID := extractSourceInfo(joinEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("join handler panicked",
				slog.String("sourceID", sourceID),
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
			slog.Any("error", err),
		)
	}
}

// dispatchMemberJoined dispatches the member joined event to all registered handlers.
func (s *Server) dispatchMemberJoined(event webhook.MemberJoinedEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeMemberJoinedHandler(handler, event)
	}
}

func (s *Server) invokeMemberJoinedHandler(handler Handler, event webhook.MemberJoinedEvent) {
	chatType, sourceID, userID := extractSourceInfo(event.Source)

	joinedUserIDs := make([]string, 0, len(event.Joined.Members))
	for _, member := range event.Joined.Members {
		joinedUserIDs = append(joinedUserIDs, member.UserId)
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("member joined handler panicked",
				slog.String("sourceID", sourceID),
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
			slog.Any("joinedUserIDs", joinedUserIDs),
			slog.Any("error", err),
		)
	}
}

// dispatchMemberLeft dispatches the member left event to all registered handlers.
func (s *Server) dispatchMemberLeft(event webhook.MemberLeftEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeMemberLeftHandler(handler, event)
	}
}

func (s *Server) invokeMemberLeftHandler(handler Handler, event webhook.MemberLeftEvent) {
	chatType, sourceID, userID := extractSourceInfo(event.Source)

	leftUserIDs := make([]string, 0, len(event.Left.Members))
	for _, member := range event.Left.Members {
		leftUserIDs = append(leftUserIDs, member.UserId)
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("member left handler panicked",
				slog.String("sourceID", sourceID),
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
			slog.Any("leftUserIDs", leftUserIDs),
			slog.Any("error", err),
		)
	}
}
