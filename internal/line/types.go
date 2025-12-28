package line

import "context"

// MessageHandler handles incoming LINE messages by type.
// Each method receives a context with timeout and message-specific parameters.
// The error return is used for logging purposes only - the HTTP response
// is already sent before handler execution.
type MessageHandler interface {
	HandleText(ctx context.Context, replyToken, userID, text string) error
	HandleImage(ctx context.Context, replyToken, userID, messageID string) error
	HandleSticker(ctx context.Context, replyToken, userID, packageID, stickerID string) error
	HandleVideo(ctx context.Context, replyToken, userID, messageID string) error
	HandleAudio(ctx context.Context, replyToken, userID, messageID string) error
	HandleLocation(ctx context.Context, replyToken, userID string, latitude, longitude float64) error
	HandleUnknown(ctx context.Context, replyToken, userID string) error
}
