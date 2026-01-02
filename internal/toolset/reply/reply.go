package reply

import (
	"context"
	_ "embed"
	"fmt"
	"time"
	"yuruppu/internal/history"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

type contextKey string

const (
	ReplyTokenKey contextKey = "replyToken"
	SourceIDKey   contextKey = "sourceID"
)

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
}

// NewTool creates a new reply tool with the specified dependencies.
func NewTool(sender Sender, historyRepo HistoryRepository) *Tool {
	return &Tool{
		sender:  sender,
		history: historyRepo,
	}
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "reply"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Send a reply message to the user. Call this tool when you want to respond. If you don't call this tool, no message will be sent."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback sends a reply and saves the assistant message to history.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message is required")
	}

	replyToken, ok := ctx.Value(ReplyTokenKey).(string)
	if !ok || replyToken == "" {
		return nil, fmt.Errorf("replyToken not found in context")
	}

	sourceID, ok := ctx.Value(SourceIDKey).(string)
	if !ok || sourceID == "" {
		return nil, fmt.Errorf("sourceID not found in context")
	}

	// Send reply
	if err := t.sender.SendReply(replyToken, message); err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}

	// Load history, append assistant message, and save
	hist, gen, err := t.history.GetHistory(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	assistantMsg := &history.AssistantMessage{
		Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: message}},
		Timestamp: time.Now(),
	}
	hist = append(hist, assistantMsg)
	if _, err := t.history.PutHistory(ctx, sourceID, hist, gen); err != nil {
		return nil, fmt.Errorf("failed to save history: %w", err)
	}

	return map[string]any{"status": "sent"}, nil
}
