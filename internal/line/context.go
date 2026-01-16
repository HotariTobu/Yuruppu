package line

import "context"

type ChatType string

const (
	ChatTypeOneOnOne ChatType = "1-on-1"
	ChatTypeGroup    ChatType = "group"
)

type ctxKey int

const (
	ctxKeyChatType ctxKey = iota
	ctxKeySourceID
	ctxKeyUserID
	ctxKeyReplyToken
)

func WithChatType(ctx context.Context, chatType ChatType) context.Context {
	return context.WithValue(ctx, ctxKeyChatType, chatType)
}

func ChatTypeFromContext(ctx context.Context) (ChatType, bool) {
	v, ok := ctx.Value(ctxKeyChatType).(ChatType)
	return v, ok
}

func WithSourceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeySourceID, id)
}

func SourceIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeySourceID).(string)
	return v, ok
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}

func WithReplyToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxKeyReplyToken, token)
}

func ReplyTokenFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyReplyToken).(string)
	return v, ok
}
