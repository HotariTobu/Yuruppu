// Package yuruppu provides the Yuruppu LINE bot agent.
package yuruppu

import (
	"context"
	_ "embed"
	"log/slog"
	"time"
	"yuruppu/internal/agent"
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

// GenerateText generates a text response given a user message.
func (y *Yuruppu) GenerateText(ctx context.Context, userMessage string) (string, error) {
	return y.agent.GenerateText(ctx, userMessage)
}

// Close cleans up the Yuruppu agent's resources.
func (y *Yuruppu) Close(ctx context.Context) error {
	return y.agent.Close(ctx)
}
