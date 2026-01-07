package reply

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// LineClient provides access to LINE API.
type LineClient interface {
	SendReply(replyToken string, text string) error
}

// HistoryService provides access to conversation history.
type HistoryService interface {
	GetHistory(ctx context.Context, sourceID string) ([]history.Message, int64, error)
	PutHistory(ctx context.Context, sourceID string, messages []history.Message, expectedGeneration int64) (int64, error)
}

// Tool implements the reply tool for sending LINE messages.
type Tool struct {
	lineClient LineClient
	history    HistoryService
	logger     *slog.Logger
}

// NewTool creates a new reply tool with the specified dependencies.
func NewTool(lineClient LineClient, historySvc HistoryService, logger *slog.Logger) (*Tool, error) {
	if lineClient == nil {
		return nil, errors.New("lineClient cannot be nil")
	}
	if historySvc == nil {
		return nil, errors.New("historySvc cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		lineClient: lineClient,
		history:    historySvc,
		logger:     logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "reply"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool to send a reply message to the user(s)."
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

	modelName, ok := agent.ModelNameFromContext(ctx)
	if !ok {
		t.logger.ErrorContext(ctx, "model name not found in context")
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
	if err := t.lineClient.SendReply(replyToken, message); err != nil {
		t.logger.ErrorContext(ctx, "failed to send reply",
			slog.String("sourceID", sourceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to send reply")
	}

	// Append assistant message to history
	assistantMsg := &history.AssistantMessage{
		ModelName: modelName,
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

// IsFinal returns true if the reply was sent successfully.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	status, ok := validatedResult["status"].(string)
	return ok && status == "sent"
}
