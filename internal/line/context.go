package line

import "context"

type ctxKey int

const (
	ctxKeyReplyToken ctxKey = iota
	ctxKeySourceID
	ctxKeyUserID
)

// WithReplyToken returns a new context with the reply token set.
func WithReplyToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxKeyReplyToken, token)
}

// WithSourceID returns a new context with the source ID set.
func WithSourceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeySourceID, id)
}

// WithUserID returns a new context with the user ID set.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

// ReplyTokenFromContext retrieves the reply token from the context.
// Returns the token and true if present, or empty string and false if not.
func ReplyTokenFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyReplyToken).(string)
	return v, ok
}

// SourceIDFromContext retrieves the source ID from the context.
// Returns the ID and true if present, or empty string and false if not.
func SourceIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeySourceID).(string)
	return v, ok
}

// UserIDFromContext retrieves the user ID from the context.
// Returns the ID and true if present, or empty string and false if not.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}
