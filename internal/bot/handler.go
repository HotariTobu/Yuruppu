package bot

import (
	"context"
	"log/slog"
	"slices"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
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
// Panics if historyRepo, agent, or sender is nil.
// logger defaults to a discard handler if nil.
func NewHandler(historyRepo *history.Repository, ag agent.Agent, sender Sender, logger *slog.Logger) *Handler {
	if historyRepo == nil {
		panic("historyRepo is required")
	}
	if ag == nil {
		panic("agent is required")
	}
	if sender == nil {
		panic("sender is required")
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Handler{
		history: historyRepo,
		agent:   ag,
		sender:  sender,
		logger:  logger,
	}
}

func (h *Handler) handleMessage(ctx context.Context, replyToken, userID, text string) error {
	now := time.Now()

	// Step 1: Load history
	conversationHistory, generation, err := h.history.GetHistory(ctx, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to load history",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 2: Append user message and save to history
	userMessage := history.Message{Role: "user", Content: text, Timestamp: now}
	historyWithUser := slices.Concat(conversationHistory, []history.Message{userMessage})
	if err := h.history.PutHistory(ctx, userID, historyWithUser, generation); err != nil {
		h.logger.ErrorContext(ctx, "failed to save user message to history",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 3: Generate response
	// Convert history.Message to agent.Message
	agentHistory := make([]agent.Message, len(historyWithUser))
	for i, m := range historyWithUser {
		agentHistory[i] = agent.Message{Role: m.Role, Content: m.Content}
	}
	response, err := h.agent.GenerateText(ctx, agentHistory)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate response",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 4: Send reply
	if err := h.sender.SendReply(replyToken, response); err != nil {
		h.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// Step 5: Append assistant message and save to history
	// Re-read to get current generation after first write
	currentHistory, newGeneration, err := h.history.GetHistory(ctx, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to read history for assistant message",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}
	assistantMessage := history.Message{Role: "assistant", Content: response, Timestamp: time.Now()}
	historyWithAssistant := slices.Concat(currentHistory, []history.Message{assistantMessage})
	if err := h.history.PutHistory(ctx, userID, historyWithAssistant, newGeneration); err != nil {
		h.logger.ErrorContext(ctx, "failed to save assistant message to history",
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
