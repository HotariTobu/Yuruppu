package server_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/line"
	"yuruppu/internal/line/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func computeSignature(body []byte, channelSecret string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type stubHandler struct{}

func (stubHandler) HandleText(context.Context, string, string) error               { return nil }
func (stubHandler) HandleImage(context.Context, string) error                      { return nil }
func (stubHandler) HandleSticker(context.Context, string, string, string) error    { return nil }
func (stubHandler) HandleVideo(context.Context, string) error                      { return nil }
func (stubHandler) HandleAudio(context.Context, string) error                      { return nil }
func (stubHandler) HandleLocation(context.Context, string, float64, float64) error { return nil }
func (stubHandler) HandleFile(context.Context, string, string, int64) error        { return nil }
func (stubHandler) HandleFollow(context.Context) error                             { return nil }
func (stubHandler) HandleJoin(context.Context) error                               { return nil }
func (stubHandler) HandleMemberJoined(context.Context, []string) error             { return nil }
func (stubHandler) HandleMemberLeft(context.Context, []string) error               { return nil }
func (stubHandler) HandleUnsend(context.Context, string) error                     { return nil }

// =============================================================================
// NewServer
// =============================================================================

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		channelSecret string
		timeout       time.Duration
		wantErr       bool
	}{
		{
			name:          "valid",
			channelSecret: "test-secret",
			timeout:       30 * time.Second,
			wantErr:       false,
		},
		{
			name:          "empty channel secret",
			channelSecret: "",
			timeout:       30 * time.Second,
			wantErr:       true,
		},
		{
			name:          "whitespace-only channel secret",
			channelSecret: "   ",
			timeout:       30 * time.Second,
			wantErr:       true,
		},
		{
			name:          "zero timeout",
			channelSecret: "test-secret",
			timeout:       0,
			wantErr:       true,
		},
		{
			name:          "negative timeout",
			channelSecret: "test-secret",
			timeout:       -5 * time.Second,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := server.NewServer(tt.channelSecret, tt.timeout, slog.New(slog.DiscardHandler))

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, s)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}

// =============================================================================
// Signature Verification
// =============================================================================

func TestHandleWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalid-signature")

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_MissingSignature(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_EmptyEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	body := `{"events":[]}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Source ID Extraction
// =============================================================================

func TestHandleWebhook_SourceID_User(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	var gotSourceID, gotUserID string
	var gotChatType line.ChatType
	handler := &sourceTestHandler{
		stubHandler: stubHandler{},
		onText: func(ctx context.Context) {
			gotSourceID, _ = line.SourceIDFromContext(ctx)
			gotUserID, _ = line.UserIDFromContext(ctx)
			gotChatType, _ = line.ChatTypeFromContext(ctx)
			close(done)
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "12345", "text": "Hello"}
		}]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	assert.Equal(t, "U1234567890abcdef", gotSourceID)
	assert.Equal(t, "U1234567890abcdef", gotUserID)
	assert.Equal(t, line.ChatTypeOneOnOne, gotChatType)
}

func TestHandleWebhook_SourceID_Group(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	var gotSourceID, gotUserID string
	var gotChatType line.ChatType
	handler := &sourceTestHandler{
		stubHandler: stubHandler{},
		onText: func(ctx context.Context) {
			gotSourceID, _ = line.SourceIDFromContext(ctx)
			gotUserID, _ = line.UserIDFromContext(ctx)
			gotChatType, _ = line.ChatTypeFromContext(ctx)
			close(done)
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "group", "groupId": "C1234567890abcdef", "userId": "U9876543210fedcba"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "12345", "text": "Hello"}
		}]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	assert.Equal(t, "C1234567890abcdef", gotSourceID)
	assert.Equal(t, "U9876543210fedcba", gotUserID)
	assert.Equal(t, line.ChatTypeGroup, gotChatType)
}

func TestHandleWebhook_SourceID_Room(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	var gotSourceID, gotUserID string
	var gotChatType line.ChatType
	handler := &sourceTestHandler{
		stubHandler: stubHandler{},
		onText: func(ctx context.Context) {
			gotSourceID, _ = line.SourceIDFromContext(ctx)
			gotUserID, _ = line.UserIDFromContext(ctx)
			gotChatType, _ = line.ChatTypeFromContext(ctx)
			close(done)
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "room", "roomId": "R1234567890abcdef", "userId": "U9876543210fedcba"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "12345", "text": "Hello"}
		}]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	assert.Equal(t, "R1234567890abcdef", gotSourceID)
	assert.Equal(t, "U9876543210fedcba", gotUserID)
	assert.Equal(t, line.ChatTypeGroup, gotChatType)
}

type sourceTestHandler struct {
	stubHandler
	onText func(ctx context.Context)
}

func (h *sourceTestHandler) HandleText(ctx context.Context, messageID, text string) error {
	if h.onText != nil {
		h.onText(ctx)
	}
	return nil
}
