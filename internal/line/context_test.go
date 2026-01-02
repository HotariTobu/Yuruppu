package line_test

import (
	"context"
	"testing"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
)

func TestWithReplyToken_And_ReplyTokenFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   string
		wantOK  bool
		wantVal string
	}{
		{
			name:    "set and retrieve token",
			token:   "test-reply-token",
			wantOK:  true,
			wantVal: "test-reply-token",
		},
		{
			name:    "empty token is valid",
			token:   "",
			wantOK:  true,
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := line.WithReplyToken(context.Background(), tt.token)
			got, ok := line.ReplyTokenFromContext(ctx)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVal, got)
		})
	}
}

func TestReplyTokenFromContext_NotSet(t *testing.T) {
	t.Parallel()

	got, ok := line.ReplyTokenFromContext(context.Background())

	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestWithSourceID_And_SourceIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantOK  bool
		wantVal string
	}{
		{
			name:    "set and retrieve user ID",
			id:      "U1234567890abcdef",
			wantOK:  true,
			wantVal: "U1234567890abcdef",
		},
		{
			name:    "set and retrieve group ID",
			id:      "C1234567890abcdef",
			wantOK:  true,
			wantVal: "C1234567890abcdef",
		},
		{
			name:    "set and retrieve room ID",
			id:      "R1234567890abcdef",
			wantOK:  true,
			wantVal: "R1234567890abcdef",
		},
		{
			name:    "empty ID is valid",
			id:      "",
			wantOK:  true,
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := line.WithSourceID(context.Background(), tt.id)
			got, ok := line.SourceIDFromContext(ctx)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVal, got)
		})
	}
}

func TestSourceIDFromContext_NotSet(t *testing.T) {
	t.Parallel()

	got, ok := line.SourceIDFromContext(context.Background())

	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestWithUserID_And_UserIDFromContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantOK  bool
		wantVal string
	}{
		{
			name:    "set and retrieve user ID",
			id:      "U1234567890abcdef",
			wantOK:  true,
			wantVal: "U1234567890abcdef",
		},
		{
			name:    "empty ID is valid",
			id:      "",
			wantOK:  true,
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := line.WithUserID(context.Background(), tt.id)
			got, ok := line.UserIDFromContext(ctx)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVal, got)
		})
	}
}

func TestUserIDFromContext_NotSet(t *testing.T) {
	t.Parallel()

	got, ok := line.UserIDFromContext(context.Background())

	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestContextValues_MultipleValuesChained(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = line.WithReplyToken(ctx, "reply-token")
	ctx = line.WithSourceID(ctx, "source-id")
	ctx = line.WithUserID(ctx, "user-id")

	token, ok := line.ReplyTokenFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "reply-token", token)

	sourceID, ok := line.SourceIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "source-id", sourceID)

	userID, ok := line.UserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "user-id", userID)
}
