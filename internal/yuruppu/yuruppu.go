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

// NewResponder creates a new Yuruppu agent with the given Agent.
// Calls agent.Configure with the embedded system prompt.
func NewResponder(a agent.Agent, logger *slog.Logger) (*Yuruppu, error) {
	ctx := context.Background()
	if err := a.Configure(ctx, systemPrompt); err != nil {
		return nil, err
	}

	return &Yuruppu{
		agent:  a,
		logger: logger,
	}, nil
}

// Respond generates a text response for the conversation history.
// The last message in history must be the user message to respond to.
func (y *Yuruppu) Respond(ctx context.Context, conversationHistory []history.Message) (string, error) {
	// Convert history.Message to agent.Message
	agentHistory := make([]agent.Message, len(conversationHistory))
	for i, msg := range conversationHistory {
		agentHistory[i] = agent.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return y.agent.GenerateText(ctx, agentHistory)
}

// Close cleans up the Yuruppu agent's resources.
func (y *Yuruppu) Close(ctx context.Context) error {
	return y.agent.Close(ctx)
}
