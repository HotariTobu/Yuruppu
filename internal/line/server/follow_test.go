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

type followHandler struct {
	stubHandler
	called   bool
	sourceID string
	userID   string
	chatType line.ChatType
	onCall   func()
}

func (h *followHandler) HandleFollow(ctx context.Context) error {
	h.called = true
	h.sourceID, _ = line.SourceIDFromContext(ctx)
	h.userID, _ = line.UserIDFromContext(ctx)
	h.chatType, _ = line.ChatTypeFromContext(ctx)
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func TestFollow_ContextValues(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &followHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

	assert.True(t, handler.called)
	assert.Equal(t, "U1234567890abcdef", handler.sourceID)
	assert.Equal(t, "U1234567890abcdef", handler.userID)
	assert.Equal(t, line.ChatTypeOneOnOne, handler.chatType)
}

func TestFollow_ContextTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	s, err := server.NewServer(channelSecret, shortTimeout, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})
	handler := &followTimeoutHandler{
		stubHandler: stubHandler{},
		onFollow: func(ctx context.Context) {
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
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

type followTimeoutHandler struct {
	stubHandler
	onFollow func(ctx context.Context)
}

func (h *followTimeoutHandler) HandleFollow(ctx context.Context) error {
	if h.onFollow != nil {
		h.onFollow(ctx)
	}
	return nil
}

func TestFollow_PanicRecovery(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	panicTriggered := make(chan struct{})
	handler := &followPanicHandler{
		stubHandler: stubHandler{},
		onFollow: func() {
			close(panicTriggered)
			panic("test panic")
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

type followPanicHandler struct {
	stubHandler
	onFollow func()
}

func (h *followPanicHandler) HandleFollow(ctx context.Context) error {
	if h.onFollow != nil {
		h.onFollow()
	}
	return nil
}

func TestFollow_AsyncExecution(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerDone := make(chan struct{})
	handler := &followHandler{onCall: func() {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
	}}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

func TestFollow_MultipleHandlers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	handler1 := &followHandler{onCall: func() { wg.Done() }}
	handler2 := &followHandler{onCall: func() { wg.Done() }}
	s.RegisterHandler(handler1)
	s.RegisterHandler(handler2)

	body := `{
		"events": [{
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

func TestFollow_HandlerError(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerCalled := make(chan struct{})
	handler := &followErrorHandler{
		stubHandler: stubHandler{},
		onFollow: func() error {
			close(handlerCalled)
			return assert.AnError
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "follow",
			"replyToken": "test-reply-token",
			"source": {"type": "user", "userId": "U1234567890abcdef"},
			"timestamp": 1625000000000
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

type followErrorHandler struct {
	stubHandler
	onFollow func() error
}

func (h *followErrorHandler) HandleFollow(ctx context.Context) error {
	if h.onFollow != nil {
		return h.onFollow()
	}
	return nil
}
