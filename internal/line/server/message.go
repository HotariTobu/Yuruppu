package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// dispatchMessage dispatches the message event to all registered handlers.
// Each handler runs asynchronously in its own goroutine with panic recovery.
func (s *Server) dispatchMessage(msgEvent webhook.MessageEvent) {
	if len(s.handlers) == 0 {
		return
	}

	for _, handler := range s.handlers {
		go s.invokeMessageHandler(handler, msgEvent)
	}
}

func (s *Server) invokeMessageHandler(handler Handler, msgEvent webhook.MessageEvent) {
	chatType, sourceID, userID := extractSourceInfo(msgEvent.Source)

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

	ctx = line.WithChatType(ctx, chatType)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	ctx = line.WithReplyToken(ctx, msgEvent.ReplyToken)

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
