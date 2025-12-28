package bot

import (
	"context"
	"log/slog"
	"time"
	"yuruppu/internal/history"
)

// Responder generates a response for a given message.
type Responder interface {
	// Respond generates a response for a given message with optional conversation history.
	// history may be nil if no history is available.
	Respond(ctx context.Context, userMessage string, history []history.Message) (string, error)
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
	history   *history.Repository
}

// New creates a new Handler with the given dependencies.
// historyRepo can be nil if history storage is not needed.
// logger defaults to a discard handler if nil.
func New(responder Responder, sender Sender, logger *slog.Logger, historyRepo *history.Repository) *Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Handler{
		responder: responder,
		sender:    sender,
		logger:    logger,
		history:   historyRepo,
	}
}

func (h *Handler) handleMessage(ctx context.Context, replyToken, userID, text string) error {
	// Load history if configured (FR-002)
	var conversationHistory []history.Message
	if h.history != nil {
		var err error
		conversationHistory, err = h.history.GetHistory(ctx, userID)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to load history",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			return err
		}
	}

	response, err := h.responder.Respond(ctx, text, conversationHistory)
	if err != nil {
		h.logger.ErrorContext(ctx, "LLM call failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// Save to history if configured (FR-001)
	if h.history != nil {
		now := time.Now()
		userMsg := history.Message{
			Role:      "user",
			Content:   text,
			Timestamp: now,
		}
		botMsg := history.Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: now,
		}
		if err := h.history.AppendMessages(ctx, userID, userMsg, botMsg); err != nil {
			h.logger.ErrorContext(ctx, "failed to save history",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			return err
		}
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
