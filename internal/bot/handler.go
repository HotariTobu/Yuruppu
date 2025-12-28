package bot

import (
	"context"
	"log/slog"
)

// Responder generates a response for a given message.
type Responder interface {
	Respond(ctx context.Context, userMessage string) (string, error)
}

// Sender sends a reply message.
type Sender interface {
	SendReply(replyToken string, text string) error
}

// Handler implements the server.Handler interface for handling LINE messages.
type Handler struct {
	responder Responder
	sender    Sender
	logger    *slog.Logger
}

// New creates a new Handler with the given dependencies.
func New(responder Responder, sender Sender, logger *slog.Logger) *Handler {
	return &Handler{
		responder: responder,
		sender:    sender,
		logger:    logger,
	}
}

func (h *Handler) handleMessage(ctx context.Context, replyToken, userID, text string) error {
	response, err := h.responder.Respond(ctx, text)
	if err != nil {
		h.logger.ErrorContext(ctx, "LLM call failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	if err := h.sender.SendReply(replyToken, response); err != nil {
		h.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

func (h *Handler) HandleText(ctx context.Context, replyToken, userID, text string) error {
	return h.handleMessage(ctx, replyToken, userID, text)
}

func (h *Handler) HandleImage(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent an image]")
}

func (h *Handler) HandleSticker(ctx context.Context, replyToken, userID, packageID, stickerID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a sticker]")
}

func (h *Handler) HandleVideo(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a video]")
}

func (h *Handler) HandleAudio(ctx context.Context, replyToken, userID, messageID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent an audio]")
}

func (h *Handler) HandleLocation(ctx context.Context, replyToken, userID string, latitude, longitude float64) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a location]")
}

func (h *Handler) HandleUnknown(ctx context.Context, replyToken, userID string) error {
	return h.handleMessage(ctx, replyToken, userID, "[User sent a message]")
}
