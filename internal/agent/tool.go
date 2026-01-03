package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Tool defines a provider-agnostic interface for function calling tools.
// Implementations must be thread-safe for concurrent execution.
type Tool interface {
	// Name returns the function name (must be unique across tools).
	Name() string

	// Description returns a human-readable description for the LLM.
	Description() string

	// ParametersJsonSchema returns the JSON Schema for input parameters as bytes.
	ParametersJsonSchema() []byte

	// ResponseJsonSchema returns the JSON Schema for the response as bytes.
	ResponseJsonSchema() []byte

	// Callback is invoked by the LLM with validated arguments.
	Callback(ctx context.Context, validatedArgs map[string]any) (map[string]any, error)
}

// FinalAction is an optional interface for tools that can end the tool loop.
// If a tool implements this interface, IsFinal is called after successful execution.
type FinalAction interface {
	IsFinal(validatedResult map[string]any) bool
}

// Validator validates data against a schema.
type Validator interface {
	Validate(data any) error
}

// tool wraps Tool with validators.
type tool struct {
	impl                Tool
	parametersValidator Validator
	responseValidator   Validator
}

// newTool creates a tool from Tool interface.
func newTool(t Tool) (tool, error) {
	paramsValidator, err := compileSchema(t.ParametersJsonSchema())
	if err != nil {
		return tool{}, fmt.Errorf("invalid parameters schema: %w", err)
	}

	respValidator, err := compileSchema(t.ResponseJsonSchema())
	if err != nil {
		return tool{}, fmt.Errorf("invalid response schema: %w", err)
	}

	return tool{
		impl:                t,
		parametersValidator: paramsValidator,
		responseValidator:   respValidator,
	}, nil
}

// UseResult contains the result of a tool execution.
type UseResult struct {
	Response map[string]any
	Final    bool
}

// Use validates args, executes callback, and validates response.
func (t *tool) Use(ctx context.Context, args map[string]any) (UseResult, error) {
	if err := t.parametersValidator.Validate(args); err != nil {
		return UseResult{}, fmt.Errorf("invalid parameters: %w", err)
	}

	result, err := t.impl.Callback(ctx, args)
	if err != nil {
		return UseResult{}, err
	}

	if err := t.responseValidator.Validate(result); err != nil {
		return UseResult{}, fmt.Errorf("invalid response: %w", err)
	}

	final := false
	if fa, ok := t.impl.(FinalAction); ok {
		final = fa.IsFinal(result)
	}

	return UseResult{Response: result, Final: final}, nil
}

// compileSchema compiles JSON Schema bytes into a validator.
func compileSchema(schemaBytes []byte) (*jsonschema.Schema, error) {
	var schemaData any
	if err := json.Unmarshal(schemaBytes, &schemaData); err != nil {
		return nil, err
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaData); err != nil {
		return nil, err
	}

	return compiler.Compile("schema.json")
}
