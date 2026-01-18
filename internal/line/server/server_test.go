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
	"sync"
	"testing"
	"time"
	"yuruppu/internal/line"
	"yuruppu/internal/line/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// computeSignature computes the LINE webhook signature for a given body and secret.
func computeSignature(body []byte, channelSecret string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// mockHandler is a test implementation of server.Handler.
type mockHandler struct {
	mu              sync.Mutex
	textMessages    []textMessage
	imageMessages   []imageMessage
	stickerMessages []stickerMessage
	videoMessages   []videoMessage
	audioMessages   []audioMessage
	locationMsgs    []locationMessage
	unknownMessages []unknownMessage
	onText          func(ctx context.Context, text string) error
	onImage         func(ctx context.Context, messageID string) error
	onSticker       func(ctx context.Context, packageID, stickerID string) error
	onVideo         func(ctx context.Context, messageID string) error
	onAudio         func(ctx context.Context, messageID string) error
	onLocation      func(ctx context.Context, lat, lng float64) error
	onUnknown       func(ctx context.Context) error
}

type textMessage struct {
	replyToken, sourceID, text string
}

type imageMessage struct {
	replyToken, sourceID, messageID string
}

type stickerMessage struct {
	replyToken, sourceID, packageID, stickerID string
}

type videoMessage struct {
	replyToken, sourceID, messageID string
}

type audioMessage struct {
	replyToken, sourceID, messageID string
}

type locationMessage struct {
	replyToken, sourceID string
	latitude, longitude  float64
}

type unknownMessage struct {
	replyToken, sourceID string
}

func (m *mockHandler) HandleText(ctx context.Context, text string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.textMessages = append(m.textMessages, textMessage{replyToken, sourceID, text})
	m.mu.Unlock()
	if m.onText != nil {
		return m.onText(ctx, text)
	}
	return nil
}

func (m *mockHandler) HandleImage(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.imageMessages = append(m.imageMessages, imageMessage{replyToken, sourceID, messageID})
	m.mu.Unlock()
	if m.onImage != nil {
		return m.onImage(ctx, messageID)
	}
	return nil
}

func (m *mockHandler) HandleSticker(ctx context.Context, packageID, stickerID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.stickerMessages = append(m.stickerMessages, stickerMessage{replyToken, sourceID, packageID, stickerID})
	m.mu.Unlock()
	if m.onSticker != nil {
		return m.onSticker(ctx, packageID, stickerID)
	}
	return nil
}

func (m *mockHandler) HandleVideo(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.videoMessages = append(m.videoMessages, videoMessage{replyToken, sourceID, messageID})
	m.mu.Unlock()
	if m.onVideo != nil {
		return m.onVideo(ctx, messageID)
	}
	return nil
}

func (m *mockHandler) HandleAudio(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.audioMessages = append(m.audioMessages, audioMessage{replyToken, sourceID, messageID})
	m.mu.Unlock()
	if m.onAudio != nil {
		return m.onAudio(ctx, messageID)
	}
	return nil
}

func (m *mockHandler) HandleLocation(ctx context.Context, latitude, longitude float64) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.locationMsgs = append(m.locationMsgs, locationMessage{replyToken, sourceID, latitude, longitude})
	m.mu.Unlock()
	if m.onLocation != nil {
		return m.onLocation(ctx, latitude, longitude)
	}
	return nil
}

func (m *mockHandler) HandleUnknown(ctx context.Context) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	m.mu.Lock()
	m.unknownMessages = append(m.unknownMessages, unknownMessage{replyToken, sourceID})
	m.mu.Unlock()
	if m.onUnknown != nil {
		return m.onUnknown(ctx)
	}
	return nil
}

func (m *mockHandler) HandleFollow(ctx context.Context) error {
	return nil
}

func (m *mockHandler) HandleJoin(ctx context.Context) error {
	return nil
}

func (m *mockHandler) HandleMemberJoined(ctx context.Context, joinedUserIDs []string) error {
	return nil
}

func (m *mockHandler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	return nil
}

// =============================================================================
// New Tests
// =============================================================================

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		channelSecret string
		wantErr       bool
	}{
		{
			name:          "valid channel secret",
			channelSecret: "test-secret",
			wantErr:       false,
		},
		{
			name:          "empty channel secret returns error",
			channelSecret: "",
			wantErr:       true,
		},
		{
			name:          "whitespace-only channel secret returns error",
			channelSecret: "   ",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := server.NewServer(tt.channelSecret, 30*time.Second, discardLogger())

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

func TestNewServer_ZeroTimeout(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 0, discardLogger())

	require.Error(t, err, "zero timeout should be rejected")
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "timeout")
}

func TestNewServer_NegativeTimeout(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", -5*time.Second, discardLogger())

	require.Error(t, err, "negative timeout should be rejected")
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "timeout")
}

// =============================================================================
// RegisterHandler Tests
// =============================================================================

func TestServer_RegisterHandler(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}

	// Should not panic
	s.RegisterHandler(handler)

	// Can register multiple handlers
	s.RegisterHandler(&mockHandler{})
}

// =============================================================================
// Signature Verification Tests
// =============================================================================

func TestServer_HandleWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalid-signature")

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServer_HandleWebhook_MissingSignature(t *testing.T) {
	t.Parallel()

	s, err := server.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServer_HandleWebhook_ValidSignature_EmptyEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
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
// Message Processing Tests
// =============================================================================

func TestServer_HandleWebhook_TextMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "12345", "text": "Hello, World!"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 1)
	assert.Equal(t, "test-reply-token", handler.textMessages[0].replyToken)
	assert.Equal(t, "test-user-id", handler.textMessages[0].sourceID)
	assert.Equal(t, "Hello, World!", handler.textMessages[0].text)
}

func TestServer_HandleWebhook_MultipleEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)
	handler.onText = func(ctx context.Context, text string) error {
		wg.Done()
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-token-1",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "First"}
			},
			{
				"type": "message",
				"replyToken": "test-token-2",
				"source": {"type": "user", "userId": "test-user-id-2"},
				"timestamp": 1625000000001,
				"message": {"type": "text", "id": "2", "text": "Second"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("not all handlers were invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 2)
	texts := []string{handler.textMessages[0].text, handler.textMessages[1].text}
	assert.Contains(t, texts, "First")
	assert.Contains(t, texts, "Second")
}

func TestServer_HandleWebhook_NonMessageEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerCalled := false
	handler.onText = func(ctx context.Context, text string) error {
		handlerCalled = true
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "follow",
				"replyToken": "test-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(50 * time.Millisecond)

	assert.False(t, handlerCalled, "handler should not be called for non-message events")
}

func TestServer_HandleWebhook_ImageMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onImage = func(ctx context.Context, messageID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "image", "id": "12345"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.imageMessages, 1)
	assert.Equal(t, "12345", handler.imageMessages[0].messageID)
}

func TestServer_HandleWebhook_StickerMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onSticker = func(ctx context.Context, packageID, stickerID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "sticker", "id": "12345", "packageId": "446", "stickerId": "1988"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.stickerMessages, 1)
	assert.Equal(t, "446", handler.stickerMessages[0].packageID)
	assert.Equal(t, "1988", handler.stickerMessages[0].stickerID)
}

func TestServer_HandleWebhook_LocationMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onLocation = func(ctx context.Context, lat, lng float64) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "location", "id": "12345", "latitude": 35.6895, "longitude": 139.6917}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.locationMsgs, 1)
	assert.InDelta(t, 35.6895, handler.locationMsgs[0].latitude, 0.0001)
	assert.InDelta(t, 139.6917, handler.locationMsgs[0].longitude, 0.0001)
}

func TestServer_HandleWebhook_NoHandler(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		s.HandleWebhook(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_HandleWebhook_MultipleHandlers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler1 := &mockHandler{}
	handler2 := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)
	handler1.onText = func(ctx context.Context, text string) error {
		wg.Done()
		return nil
	}
	handler2.onText = func(ctx context.Context, text string) error {
		wg.Done()
		return nil
	}
	s.RegisterHandler(handler1)
	s.RegisterHandler(handler2)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("not all handlers were invoked")
	}

	handler1.mu.Lock()
	defer handler1.mu.Unlock()
	handler2.mu.Lock()
	defer handler2.mu.Unlock()

	assert.Len(t, handler1.textMessages, 1)
	assert.Len(t, handler2.textMessages, 1)
}

// =============================================================================
// Async Execution and Context Tests
// =============================================================================

func TestServer_HandleWebhook_AsyncExecution(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerDone := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	start := time.Now()
	s.HandleWebhook(w, req)
	responseTime := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, responseTime, 100*time.Millisecond, "HTTP response should be sent before handler completes")

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete within timeout")
	}
}

func TestServer_HandleWebhook_ContextWithTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	var receivedCtx context.Context
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		receivedCtx = ctx
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
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

	_, hasDeadline := receivedCtx.Deadline()
	assert.True(t, hasDeadline, "context should have a timeout deadline")
}

func TestServer_CallbackTimeout_Enforcement(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	s, err := server.NewServer(channelSecret, shortTimeout, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(handlerStarted)
		select {
		case <-ctx.Done():
			close(contextCancelled)
			return ctx.Err()
		case <-time.After(5 * time.Second):
			t.Error("context was not cancelled within timeout")
		}
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	select {
	case <-handlerStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("handler was not invoked")
	}

	select {
	case <-contextCancelled:
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled by timeout")
	}
}

func TestServer_HandleWebhook_PanicRecovery(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	panicTriggered := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(panicTriggered)
		panic("test panic")
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "test-user-id"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		s.HandleWebhook(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-panicTriggered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}
}

// =============================================================================
// Source ID Extraction Tests (FR-003)
// =============================================================================

func TestServer_HandleWebhook_GroupSource(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "group", "groupId": "C1234567890abcdef", "userId": "U9876543210fedcba"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "12345", "text": "Hello from group!"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 1)
	assert.Equal(t, "test-reply-token", handler.textMessages[0].replyToken)
	// For group source, sourceID should be groupId (not userId)
	assert.Equal(t, "C1234567890abcdef", handler.textMessages[0].sourceID)
	assert.Equal(t, "Hello from group!", handler.textMessages[0].text)
}

func TestServer_HandleWebhook_RoomSource(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "room", "roomId": "R1234567890abcdef", "userId": "U9876543210fedcba"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "12345", "text": "Hello from room!"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 1)
	assert.Equal(t, "test-reply-token", handler.textMessages[0].replyToken)
	// For room source, sourceID should be roomId (not userId)
	assert.Equal(t, "R1234567890abcdef", handler.textMessages[0].sourceID)
	assert.Equal(t, "Hello from room!", handler.textMessages[0].text)
}

func TestServer_HandleWebhook_UserSource(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "12345", "text": "Hello from 1:1 chat!"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 1)
	assert.Equal(t, "test-reply-token", handler.textMessages[0].replyToken)
	// For user source, sourceID should be userId
	assert.Equal(t, "U1234567890abcdef", handler.textMessages[0].sourceID)
	assert.Equal(t, "Hello from 1:1 chat!", handler.textMessages[0].text)
}

func TestServer_HandleWebhook_MissingSource(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, text string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	// Webhook event without source field - should not crash
	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "12345", "text": "No source!"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.textMessages, 1)
	// Missing source should result in empty sourceID
	assert.Equal(t, "", handler.textMessages[0].sourceID)
}
