// Package yuruppu provides the Yuruppu LINE bot agent.
package yuruppu

import (
	// Standard library
	"context"
	_ "embed"
	"log/slog"
	"time"

	// Internal packages
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
	"yuruppu/internal/llm"
)

//go:embed prompt/system.txt
var systemPrompt string

// Yuruppu is the Yuruppu character agent.
// It wraps a generic Agent with the Yuruppu-specific system prompt.
type Yuruppu struct {
	agent  *agent.Agent
	logger *slog.Logger
}

// New creates a new Yuruppu agent with the given LLM provider.
// cacheTTL specifies the TTL for the cached system prompt.
func New(provider llm.Provider, cacheTTL time.Duration, logger *slog.Logger) *Yuruppu {
	a := agent.New(provider, systemPrompt, cacheTTL, logger)
	return &Yuruppu{
		agent:  a,
		logger: logger,
	}
}

// Respond generates a text response given a user message with optional conversation history.
// history may be nil if no history is available.
func (y *Yuruppu) Respond(ctx context.Context, userMessage string, conversationHistory []history.Message) (string, error) {
	// Convert history.Message to llm.Message
	var llmHistory []llm.Message
	if conversationHistory != nil {
		llmHistory = make([]llm.Message, len(conversationHistory))
		for i, msg := range conversationHistory {
			llmHistory[i] = llm.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
	}
	return y.agent.GenerateText(ctx, userMessage, llmHistory)
}

// Close cleans up the Yuruppu agent's resources.
func (y *Yuruppu) Close(ctx context.Context) error {
	return y.agent.Close(ctx)
}
