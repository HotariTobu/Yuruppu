package bot

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/message"
)

// Sender sends a reply message.
type Sender interface {
	SendReply(replyToken string, text string) error
}

// Handler implements the server.Handler interface for handling LINE messages.
type Handler struct {
	history *history.Repository
	agent   agent.Agent
	sender  Sender
	logger  *slog.Logger
}

// NewHandler creates a new Handler with the given dependencies.
// Returns error if any dependency is nil.
func NewHandler(historyRepo *history.Repository, agent agent.Agent, sender Sender, logger *slog.Logger) (*Handler, error) {
	if historyRepo == nil {
		return nil, fmt.Errorf("historyRepo is required")
	}
	if agent == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("sender is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	return &Handler{
		history: historyRepo,
		agent:   agent,
		sender:  sender,
		logger:  logger,
	}, nil
}

func (h *Handler) handleMessage(ctx context.Context, replyToken, sourceID, text string) error {
	// Step 1: Load history
	conversationHistory, generation, err := h.history.GetHistory(ctx, sourceID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to load history",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 2: Append user message and save to history
	userMessage := message.Message{Role: "user", Content: text, Timestamp: time.Now()}
	historyWithUser := slices.Concat(conversationHistory, []message.Message{userMessage})
	if err := h.history.PutHistory(ctx, sourceID, historyWithUser, generation); err != nil {
		h.logger.ErrorContext(ctx, "failed to save user message to history",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 3: Generate response
	response, err := h.agent.GenerateText(ctx, historyWithUser)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate response",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 4: Send reply
	if err := h.sender.SendReply(replyToken, response); err != nil {
		h.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 5: Append assistant message and save to history
	// Re-read to get current generation after first write
	currentHistory, newGeneration, err := h.history.GetHistory(ctx, sourceID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to read history for assistant message",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}
	assistantMessage := message.Message{Role: "assistant", Content: response, Timestamp: time.Now()}
	historyWithAssistant := slices.Concat(currentHistory, []message.Message{assistantMessage})
	if err := h.history.PutHistory(ctx, sourceID, historyWithAssistant, newGeneration); err != nil {
		h.logger.ErrorContext(ctx, "failed to save assistant message to history",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

func (h *Handler) HandleText(ctx context.Context, replyToken, sourceID, text string) error {
	return h.handleMessage(ctx, replyToken, sourceID, text)
}

func (h *Handler) HandleImage(ctx context.Context, replyToken, sourceID, messageID string) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent an image]")
}

func (h *Handler) HandleSticker(ctx context.Context, replyToken, sourceID, packageID, stickerID string) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent a sticker]")
}

func (h *Handler) HandleVideo(ctx context.Context, replyToken, sourceID, messageID string) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent a video]")
}

func (h *Handler) HandleAudio(ctx context.Context, replyToken, sourceID, messageID string) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent an audio]")
}

func (h *Handler) HandleLocation(ctx context.Context, replyToken, sourceID string, latitude, longitude float64) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent a location]")
}

func (h *Handler) HandleUnknown(ctx context.Context, replyToken, sourceID string) error {
	return h.handleMessage(ctx, replyToken, sourceID, "[User sent a message]")
}
