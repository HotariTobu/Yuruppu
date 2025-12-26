package line

import "context"

// Message represents an incoming LINE message.
type Message struct {
	ReplyToken string
	Type       string // "text", "image", "sticker", etc.
	Text       string // For text messages; formatted for others
	UserID     string
}

// MessageHandler is the callback signature for message processing.
// It receives a context with timeout and a Message.
// The error return is used for logging purposes only - the HTTP response
// is already sent before callback execution.
type MessageHandler func(ctx context.Context, msg Message) error
