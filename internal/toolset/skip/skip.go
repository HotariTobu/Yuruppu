package skip

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
)

//go:embed parameters.json
var parametersSchema []byte

//go:embed response.json
var responseSchema []byte

// Tool implements the skip tool for explicitly not replying.
type Tool struct {
	logger *slog.Logger
}

// NewTool creates a new skip tool.
func NewTool(logger *slog.Logger) (*Tool, error) {
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	return &Tool{
		logger: logger,
	}, nil
}

// Name returns the tool name.
func (t *Tool) Name() string {
	return "skip"
}

// Description returns a description for the LLM.
func (t *Tool) Description() string {
	return "Use this tool ONLY when you decide to do nothing for the user(s)."
}

// ParametersJsonSchema returns the JSON Schema for input parameters.
func (t *Tool) ParametersJsonSchema() []byte {
	return parametersSchema
}

// ResponseJsonSchema returns the JSON Schema for the response.
func (t *Tool) ResponseJsonSchema() []byte {
	return responseSchema
}

// Callback does nothing and returns success.
func (t *Tool) Callback(ctx context.Context, args map[string]any) (map[string]any, error) {
	reason, _ := args["reason"].(string)
	t.logger.DebugContext(ctx, "skip tool called", slog.String("reason", reason))
	return map[string]any{
		"status": "skipped",
	}, nil
}

// IsFinal returns true if the skip was successful.
func (t *Tool) IsFinal(validatedResult map[string]any) bool {
	status, ok := validatedResult["status"].(string)
	return ok && status == "skipped"
}
