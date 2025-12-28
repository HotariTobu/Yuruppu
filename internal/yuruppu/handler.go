package yuruppu

import (
	"context"
	"log/slog"
)

// LLMProvider is the interface for LLM operations.
type LLMProvider interface {
	GenerateText(ctx context.Context, userMessage string) (string, error)
}

// Replier is the interface for sending replies.
// This allows Handler to send replies without depending on the line package directly.
type Replier interface {
	SendReply(replyToken string, text string) error
}

// Message represents an incoming message.
// This mirrors line.Message but avoids circular imports.
type Message struct {
	ReplyToken string
	Type       string // "text", "image", "sticker", etc.
	Text       string // For text messages; formatted for others
	UserID     string
}

// Handler processes incoming messages using LLM and sends replies.
// Handler is created from Yuruppu using yuruppu.NewHandler(client).
type Handler struct {
	llm    LLMProvider
	client Replier
	logger *slog.Logger
}

// NewHandler creates a Handler for testing purposes.
// In production, use Yuruppu.NewHandler(client) instead.
func NewHandler(llm LLMProvider, client Replier, logger *slog.Logger) *Handler {
	return &Handler{
		llm:    llm,
		client: client,
		logger: logger,
	}
}

// HandleMessage processes an incoming message.
// It calls the LLM with the user message and sends the reply via the client.
// Returns an error if LLM call or reply sending fails.
func (h *Handler) HandleMessage(ctx context.Context, msg Message) error {
	// Log incoming message at DEBUG level
	h.logger.DebugContext(ctx, "handling message",
		slog.String("userID", msg.UserID),
		slog.String("type", msg.Type),
		slog.String("text", msg.Text),
	)

	// Format user message based on type
	userMessage := msg.Text
	if msg.Type != "text" {
		userMessage = formatNonTextMessage(msg.Type)
	}

	// Call LLM to generate response
	response, err := h.llm.GenerateText(ctx, userMessage)
	if err != nil {
		h.logger.ErrorContext(ctx, "LLM call failed",
			slog.String("userID", msg.UserID),
			slog.String("replyToken", msg.ReplyToken),
			slog.Any("error", err),
		)
		return err
	}

	// Log LLM response at DEBUG level
	h.logger.DebugContext(ctx, "LLM response",
		slog.String("response", response),
	)

	// Send reply
	if err := h.client.SendReply(msg.ReplyToken, response); err != nil {
		h.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("userID", msg.UserID),
			slog.String("replyToken", msg.ReplyToken),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// formatNonTextMessage returns a formatted string for non-text message types.
// FR-008: For non-text messages, use format "[User sent a {type}]"
func formatNonTextMessage(msgType string) string {
	switch msgType {
	case "image":
		return "[User sent an image]"
	case "sticker":
		return "[User sent a sticker]"
	case "video":
		return "[User sent a video]"
	case "audio":
		return "[User sent an audio]"
	case "location":
		return "[User sent a location]"
	default:
		return "[User sent a message]"
	}
}
