package server

import (
	"context"
	"log/slog"
	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// MessageHandler handles incoming LINE messages by type.
// Each method receives a context with timeout and LINE-specific values
// (reply token, source ID, user ID) accessible via context accessor functions.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type MessageHandler interface {
	HandleText(ctx context.Context, text string) error
	HandleImage(ctx context.Context, messageID string) error
	HandleSticker(ctx context.Context, packageID, stickerID string) error
	HandleVideo(ctx context.Context, messageID string) error
	HandleAudio(ctx context.Context, messageID string) error
	HandleLocation(ctx context.Context, latitude, longitude float64) error
	HandleFile(ctx context.Context, messageID, fileName string, fileSize int64) error
}

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

func (s *Server) invokeMessageHandler(handler MessageHandler, msgEvent webhook.MessageEvent) {
	chatType, sourceID, userID := extractSourceInfo(msgEvent.Source)

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("message handler panicked",
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
	case webhook.FileMessageContent:
		err = handler.HandleFile(ctx, msg.Id, msg.FileName, int64(msg.FileSize))
	}

	if err != nil {
		s.logger.Error("message handler failed",
			slog.String("sourceID", sourceID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
	}
}
