package agent

import "context"

type ctxKey int

const (
	ctxKeyModelName ctxKey = iota
)

// WithModelName returns a new context with the model name set.
func WithModelName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, ctxKeyModelName, name)
}

// ModelNameFromContext retrieves the model name from the context.
// Returns the name and true if present, or empty string and false if not.
func ModelNameFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyModelName).(string)
	return v, ok
}
