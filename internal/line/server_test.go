package line_test

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

// mockHandler is a test implementation of line.MessageHandler.
type mockHandler struct {
	mu              sync.Mutex
	textMessages    []textMessage
	imageMessages   []imageMessage
	stickerMessages []stickerMessage
	videoMessages   []videoMessage
	audioMessages   []audioMessage
	locationMsgs    []locationMessage
	unknownMessages []unknownMessage
	onText          func(ctx context.Context, replyToken, userID, text string) error
	onImage         func(ctx context.Context, replyToken, userID, messageID string) error
	onSticker       func(ctx context.Context, replyToken, userID, packageID, stickerID string) error
	onVideo         func(ctx context.Context, replyToken, userID, messageID string) error
	onAudio         func(ctx context.Context, replyToken, userID, messageID string) error
	onLocation      func(ctx context.Context, replyToken, userID string, lat, lng float64) error
	onUnknown       func(ctx context.Context, replyToken, userID string) error
}

type textMessage struct {
	replyToken, userID, text string
}

type imageMessage struct {
	replyToken, userID, messageID string
}

type stickerMessage struct {
	replyToken, userID, packageID, stickerID string
}

type videoMessage struct {
	replyToken, userID, messageID string
}

type audioMessage struct {
	replyToken, userID, messageID string
}

type locationMessage struct {
	replyToken, userID string
	latitude, longitude float64
}

type unknownMessage struct {
	replyToken, userID string
}

func (m *mockHandler) HandleText(ctx context.Context, replyToken, userID, text string) error {
	m.mu.Lock()
	m.textMessages = append(m.textMessages, textMessage{replyToken, userID, text})
	m.mu.Unlock()
	if m.onText != nil {
		return m.onText(ctx, replyToken, userID, text)
	}
	return nil
}

func (m *mockHandler) HandleImage(ctx context.Context, replyToken, userID, messageID string) error {
	m.mu.Lock()
	m.imageMessages = append(m.imageMessages, imageMessage{replyToken, userID, messageID})
	m.mu.Unlock()
	if m.onImage != nil {
		return m.onImage(ctx, replyToken, userID, messageID)
	}
	return nil
}

func (m *mockHandler) HandleSticker(ctx context.Context, replyToken, userID, packageID, stickerID string) error {
	m.mu.Lock()
	m.stickerMessages = append(m.stickerMessages, stickerMessage{replyToken, userID, packageID, stickerID})
	m.mu.Unlock()
	if m.onSticker != nil {
		return m.onSticker(ctx, replyToken, userID, packageID, stickerID)
	}
	return nil
}

func (m *mockHandler) HandleVideo(ctx context.Context, replyToken, userID, messageID string) error {
	m.mu.Lock()
	m.videoMessages = append(m.videoMessages, videoMessage{replyToken, userID, messageID})
	m.mu.Unlock()
	if m.onVideo != nil {
		return m.onVideo(ctx, replyToken, userID, messageID)
	}
	return nil
}

func (m *mockHandler) HandleAudio(ctx context.Context, replyToken, userID, messageID string) error {
	m.mu.Lock()
	m.audioMessages = append(m.audioMessages, audioMessage{replyToken, userID, messageID})
	m.mu.Unlock()
	if m.onAudio != nil {
		return m.onAudio(ctx, replyToken, userID, messageID)
	}
	return nil
}

func (m *mockHandler) HandleLocation(ctx context.Context, replyToken, userID string, latitude, longitude float64) error {
	m.mu.Lock()
	m.locationMsgs = append(m.locationMsgs, locationMessage{replyToken, userID, latitude, longitude})
	m.mu.Unlock()
	if m.onLocation != nil {
		return m.onLocation(ctx, replyToken, userID, latitude, longitude)
	}
	return nil
}

func (m *mockHandler) HandleUnknown(ctx context.Context, replyToken, userID string) error {
	m.mu.Lock()
	m.unknownMessages = append(m.unknownMessages, unknownMessage{replyToken, userID})
	m.mu.Unlock()
	if m.onUnknown != nil {
		return m.onUnknown(ctx, replyToken, userID)
	}
	return nil
}

// =============================================================================
// NewServer Tests
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

			server, err := line.NewServer(tt.channelSecret, 30*time.Second, discardLogger())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, server)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server)
			}
		})
	}
}

func TestNewServer_ZeroTimeout(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", 0, discardLogger())

	require.Error(t, err, "zero timeout should be rejected")
	assert.Nil(t, server)

	var configErr *line.ConfigError
	require.ErrorAs(t, err, &configErr)
	assert.Equal(t, "timeout", configErr.Variable)
}

func TestNewServer_NegativeTimeout(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", -5*time.Second, discardLogger())

	require.Error(t, err, "negative timeout should be rejected")
	assert.Nil(t, server)

	var configErr *line.ConfigError
	require.ErrorAs(t, err, &configErr)
	assert.Equal(t, "timeout", configErr.Variable)
}

// =============================================================================
// RegisterHandler Tests
// =============================================================================

func TestServer_RegisterHandler(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}

	// Should not panic
	server.RegisterHandler(handler)

	// Can register multiple handlers
	server.RegisterHandler(&mockHandler{})
}

// =============================================================================
// Signature Verification Tests
// =============================================================================

func TestServer_HandleWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalid-signature")

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServer_HandleWebhook_MissingSignature(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServer_HandleWebhook_ValidSignature_EmptyEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Message Processing Tests
// =============================================================================

func TestServer_HandleWebhook_TextMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		close(done)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	assert.Equal(t, "test-user-id", handler.textMessages[0].userID)
	assert.Equal(t, "Hello, World!", handler.textMessages[0].text)
}

func TestServer_HandleWebhook_MultipleEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		wg.Done()
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerCalled := false
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		handlerCalled = true
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	time.Sleep(50 * time.Millisecond)

	assert.False(t, handlerCalled, "handler should not be called for non-message events")
}

func TestServer_HandleWebhook_ImageMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onImage = func(ctx context.Context, replyToken, userID, messageID string) error {
		close(done)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onSticker = func(ctx context.Context, replyToken, userID, packageID, stickerID string) error {
		close(done)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onLocation = func(ctx context.Context, replyToken, userID string, lat, lng float64) error {
		close(done)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
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
		server.HandleWebhook(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_HandleWebhook_MultipleHandlers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler1 := &mockHandler{}
	handler2 := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)
	handler1.onText = func(ctx context.Context, replyToken, userID, text string) error {
		wg.Done()
		return nil
	}
	handler2.onText = func(ctx context.Context, replyToken, userID, text string) error {
		wg.Done()
		return nil
	}
	server.RegisterHandler(handler1)
	server.RegisterHandler(handler2)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerDone := make(chan struct{})
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)
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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	var receivedCtx context.Context
	done := make(chan struct{})
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		receivedCtx = ctx
		close(done)
		return nil
	}
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, shortTimeout, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
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
	server.RegisterHandler(handler)

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
	server.HandleWebhook(w, req)

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
	server, err := line.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	panicTriggered := make(chan struct{})
	handler.onText = func(ctx context.Context, replyToken, userID, text string) error {
		close(panicTriggered)
		panic("test panic")
	}
	server.RegisterHandler(handler)

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
		server.HandleWebhook(w, req)
	})

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-panicTriggered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}
}
