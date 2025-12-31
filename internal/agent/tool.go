// Package agent provides the Agent interface for LLM interactions.
package agent

import (
	"context"

	"google.golang.org/genai"
)

// Tool represents a callable tool that the LLM can invoke.
type Tool interface {
	// Name returns the tool name (must match LLM's function call name).
	Name() string

	// Declaration returns the tool's function declaration for the LLM.
	// This includes name, description, and parameter schema.
	Declaration() *genai.FunctionDeclaration

	// Execute runs the tool with the given arguments from the LLM.
	// Args is a map[string]any containing the parameters.
	// Returns the tool result as a string and any error.
	// Per NFR-002, errors should be returned as strings for the LLM to process.
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// toGenaiTools converts a slice of Tools to genai.Tool format.
func toGenaiTools(tools []Tool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	declarations := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		declarations[i] = t.Declaration()
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}
}

// buildToolMap creates a map from tool name to Tool for quick lookup.
func buildToolMap(tools []Tool) map[string]Tool {
	m := make(map[string]Tool, len(tools))
	for _, t := range tools {
		m[t.Name()] = t
	}
	return m
}
