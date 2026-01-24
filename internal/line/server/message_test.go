package server_test

import (
	"context"
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

type messageHandler struct {
	stubHandler
	mu       sync.Mutex
	messages []receivedMessage
	onCall   func()
}

type receivedMessage struct {
	messageType string
	messageID   string
	replyToken  string
	sourceID    string
	text        string
	packageID   string
	stickerID   string
	latitude    float64
	longitude   float64
	fileName    string
	fileSize    int64
}

func (h *messageHandler) HandleText(ctx context.Context, messageID, text string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "text",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
		text:        text,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleImage(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "image",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleSticker(ctx context.Context, messageID, packageID, stickerID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "sticker",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
		packageID:   packageID,
		stickerID:   stickerID,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleVideo(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "video",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleAudio(ctx context.Context, messageID string) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "audio",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleLocation(ctx context.Context, messageID string, latitude, longitude float64) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "location",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
		latitude:    latitude,
		longitude:   longitude,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func (h *messageHandler) HandleFile(ctx context.Context, messageID, fileName string, fileSize int64) error {
	replyToken, _ := line.ReplyTokenFromContext(ctx)
	sourceID, _ := line.SourceIDFromContext(ctx)
	h.mu.Lock()
	h.messages = append(h.messages, receivedMessage{
		messageType: "file",
		messageID:   messageID,
		replyToken:  replyToken,
		sourceID:    sourceID,
		fileName:    fileName,
		fileSize:    fileSize,
	})
	h.mu.Unlock()
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func TestMessage_Text(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "12345", "text": "Hello, World!"}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "text", handler.messages[0].messageType)
	assert.Equal(t, "12345", handler.messages[0].messageID)
	assert.Equal(t, "test-reply-token", handler.messages[0].replyToken)
	assert.Equal(t, "test-user-id", handler.messages[0].sourceID)
	assert.Equal(t, "Hello, World!", handler.messages[0].text)
}

func TestMessage_Image(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "image", "id": "12345"}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "image", handler.messages[0].messageType)
	assert.Equal(t, "12345", handler.messages[0].messageID)
}

func TestMessage_Sticker(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "sticker", "id": "12345", "packageId": "446", "stickerId": "1988"}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "sticker", handler.messages[0].messageType)
	assert.Equal(t, "446", handler.messages[0].packageID)
	assert.Equal(t, "1988", handler.messages[0].stickerID)
}

func TestMessage_Location(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "location", "id": "12345", "latitude": 35.6895, "longitude": 139.6917}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "location", handler.messages[0].messageType)
	assert.InDelta(t, 35.6895, handler.messages[0].latitude, 0.0001)
	assert.InDelta(t, 139.6917, handler.messages[0].longitude, 0.0001)
}

func TestMessage_MultipleEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)
	handler := &messageHandler{onCall: func() { wg.Done() }}
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
		t.Fatal("not all handlers were invoked")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.messages, 2)
	texts := []string{handler.messages[0].text, handler.messages[1].text}
	assert.Contains(t, texts, "First")
	assert.Contains(t, texts, "Second")
}

func TestMessage_AsyncExecution(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerDone := make(chan struct{})
	handler := &messageHandler{onCall: func() {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
	}}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "1", "text": "test"}
		}]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	start := time.Now()
	s.HandleWebhook(w, req)
	responseTime := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, responseTime, 100*time.Millisecond)

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete")
	}
}

func TestMessage_ContextTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	s, err := server.NewServer(channelSecret, shortTimeout, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})
	handler := &contextCheckHandler{
		stubHandler: stubHandler{},
		onText: func(ctx context.Context) {
			close(handlerStarted)
			select {
			case <-ctx.Done():
				close(contextCancelled)
			case <-time.After(5 * time.Second):
				t.Error("context was not cancelled")
			}
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "1", "text": "test"}
		}]
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
		t.Fatal("context was not cancelled")
	}
}

type contextCheckHandler struct {
	stubHandler
	onText func(ctx context.Context)
}

func (h *contextCheckHandler) HandleText(ctx context.Context, messageID, text string) error {
	if h.onText != nil {
		h.onText(ctx)
	}
	return nil
}

func TestMessage_PanicRecovery(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	panicTriggered := make(chan struct{})
	handler := &panicHandler{
		stubHandler: stubHandler{},
		onText: func() {
			close(panicTriggered)
			panic("test panic")
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "1", "text": "test"}
		}]
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

type panicHandler struct {
	stubHandler
	onText func()
}

func (h *panicHandler) HandleText(ctx context.Context, messageID, text string) error {
	if h.onText != nil {
		h.onText()
	}
	return nil
}

func TestMessage_Video(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "video", "id": "12345"}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "video", handler.messages[0].messageType)
	assert.Equal(t, "12345", handler.messages[0].messageID)
}

func TestMessage_Audio(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "audio", "id": "12345", "duration": 60000}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "audio", handler.messages[0].messageType)
	assert.Equal(t, "12345", handler.messages[0].messageID)
}

func TestMessage_File(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &messageHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "file", "id": "12345", "fileName": "document.pdf", "fileSize": 1024}
		}]
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

	require.Len(t, handler.messages, 1)
	assert.Equal(t, "file", handler.messages[0].messageType)
	assert.Equal(t, "12345", handler.messages[0].messageID)
	assert.Equal(t, "document.pdf", handler.messages[0].fileName)
	assert.Equal(t, int64(1024), handler.messages[0].fileSize)
}

func TestMessage_HandlerError(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerCalled := make(chan struct{})
	handler := &errorHandler{
		stubHandler: stubHandler{},
		onText: func() error {
			close(handlerCalled)
			return assert.AnError
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "12345", "text": "Hello"}
		}]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	select {
	case <-handlerCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}
}

type errorHandler struct {
	stubHandler
	onText func() error
}

func (h *errorHandler) HandleText(ctx context.Context, messageID, text string) error {
	if h.onText != nil {
		return h.onText()
	}
	return nil
}

func TestMessage_MultipleHandlers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	handler1 := &messageHandler{onCall: func() { wg.Done() }}
	handler2 := &messageHandler{onCall: func() { wg.Done() }}
	s.RegisterHandler(handler1)
	s.RegisterHandler(handler2)

	body := `{
		"events": [{
			"type": "message",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "test-user-id"},
			"timestamp": 1625000000000,
			"message": {"type": "text", "id": "1", "text": "test"}
		}]
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
}
