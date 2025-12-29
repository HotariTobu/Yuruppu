// Package yuruppu provides the Yuruppu LINE bot agent.
package yuruppu

import (
	"context"
	_ "embed"
	"log/slog"
	"yuruppu/internal/agent"
	"yuruppu/internal/history"
)

//go:embed prompt/system.txt
var systemPrompt string

// Yuruppu is the Yuruppu character agent.
// It wraps an Agent with the Yuruppu-specific system prompt.
type Yuruppu struct {
	agent  agent.Agent
	logger *slog.Logger
}

// New creates a new Yuruppu agent with the given Agent.
// Calls agent.Configure with the embedded system prompt.
func New(a agent.Agent, logger *slog.Logger) (*Yuruppu, error) {
	ctx := context.Background()
	if err := a.Configure(ctx, systemPrompt); err != nil {
		return nil, err
	}

	return &Yuruppu{
		agent:  a,
		logger: logger,
	}, nil
}

// Respond generates a text response given a user message with optional conversation history.
// history may be nil if no history is available.
func (y *Yuruppu) Respond(ctx context.Context, userMessage string, conversationHistory []history.Message) (string, error) {
	// Convert history.Message to agent.Message
	var agentHistory []agent.Message
	if conversationHistory != nil {
		agentHistory = make([]agent.Message, len(conversationHistory))
		for i, msg := range conversationHistory {
			agentHistory[i] = agent.Message{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}
	}
	return y.agent.GenerateText(ctx, userMessage, agentHistory)
}

// Close cleans up the Yuruppu agent's resources.
func (y *Yuruppu) Close(ctx context.Context) error {
	return y.agent.Close(ctx)
}
