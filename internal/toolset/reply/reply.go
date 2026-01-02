package reply

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// Sender sends a reply message.
type Sender interface {
	SendReply(replyToken string, text string) error
}

// HistoryRepository provides access to conversation history.
type HistoryRepository interface {
	GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error)
	PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error)
}

// Tool implements the reply tool for sending LINE messages.
type Tool struct {
	sender  Sender
	history HistoryRepository
	logger  *slog.Logger
}

// NewTool creates a new reply tool with the specified dependencies.
func NewTool(sender Sender, historyRepo HistoryRepository, logger *slog.Logger) *Tool {
	return &Tool{
		sender:  sender,
		history: historyRepo,
		logger:  logger,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "reply"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Send a reply message to the user. Only call this tool if you want to send a message."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback sends a reply message to the user.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("invalid message")
	}

	// Get replyToken and sourceID from context
	replyToken, ok := line.ReplyTokenFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "reply token not found in context")
		return nil, fmt.Errorf("internal error")
	}

	sourceID, ok := line.SourceIDFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "source ID not found in context")
		return nil, fmt.Errorf("internal error")
	}

	// Load history
	hist, gen, err := t.history.GetHistory(ctx, sourceID)
	if err != nil {
		t.logger.ErrorContext(ctx, "failed to load history",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to load conversation")
	}

	// Send reply
	if err := t.sender.SendReply(replyToken, message); err != nil {
		t.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to send reply")
	}

	// Append assistant message to history
	assistantMsg := &history.AssistantMessage{
		Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: message}},
		Timestamp: time.Now(),
	}
	hist = append(hist, assistantMsg)

	// Save history
	if _, err := t.history.PutHistory(ctx, sourceID, hist, gen); err != nil {
		t.logger.ErrorContext(ctx, "failed to save history",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to save message")
	}

	return map[string]any{
		"status": "sent",
	}, nil
}
