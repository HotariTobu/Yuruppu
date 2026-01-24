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

type memberLeftHandler struct {
	stubHandler
	called      bool
	sourceID    string
	chatType    line.ChatType
	leftUserIDs []string
	onCall      func()
}

func (h *memberLeftHandler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	h.called = true
	h.sourceID, _ = line.SourceIDFromContext(ctx)
	h.chatType, _ = line.ChatTypeFromContext(ctx)
	h.leftUserIDs = leftUserIDs
	if h.onCall != nil {
		h.onCall()
	}
	return nil
}

func TestMemberLeft_ContextValues(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &memberLeftHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {
				"members": [
					{"type": "user", "userId": "U333333333333"}
				]
			}
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
	assert.Equal(t, "C1234567890abcdef", handler.sourceID)
	assert.Equal(t, line.ChatTypeGroup, handler.chatType)
	assert.Equal(t, []string{"U333333333333"}, handler.leftUserIDs)
}

func TestMemberLeft_ContextTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	s, err := server.NewServer(channelSecret, shortTimeout, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})
	handler := &memberLeftTimeoutHandler{
		stubHandler: stubHandler{},
		onMemberLeft: func(ctx context.Context) {
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
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": [{"type": "user", "userId": "U333333333333"}]}
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

type memberLeftTimeoutHandler struct {
	stubHandler
	onMemberLeft func(ctx context.Context)
}

func (h *memberLeftTimeoutHandler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	if h.onMemberLeft != nil {
		h.onMemberLeft(ctx)
	}
	return nil
}

func TestMemberLeft_PanicRecovery(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	panicTriggered := make(chan struct{})
	handler := &memberLeftPanicHandler{
		stubHandler: stubHandler{},
		onMemberLeft: func() {
			close(panicTriggered)
			panic("test panic")
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": [{"type": "user", "userId": "U333333333333"}]}
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

type memberLeftPanicHandler struct {
	stubHandler
	onMemberLeft func()
}

func (h *memberLeftPanicHandler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	if h.onMemberLeft != nil {
		h.onMemberLeft()
	}
	return nil
}

func TestMemberLeft_AsyncExecution(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerDone := make(chan struct{})
	handler := &memberLeftHandler{onCall: func() {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
	}}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": [{"type": "user", "userId": "U333333333333"}]}
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

func TestMemberLeft_MultipleHandlers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	handler1 := &memberLeftHandler{onCall: func() { wg.Done() }}
	handler2 := &memberLeftHandler{onCall: func() { wg.Done() }}
	s.RegisterHandler(handler1)
	s.RegisterHandler(handler2)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": [{"type": "user", "userId": "U333333333333"}]}
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

func TestMemberLeft_HandlerError(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	handlerCalled := make(chan struct{})
	handler := &memberLeftErrorHandler{
		stubHandler: stubHandler{},
		onMemberLeft: func() error {
			close(handlerCalled)
			return assert.AnError
		},
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": [{"type": "user", "userId": "U333333333333"}]}
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

type memberLeftErrorHandler struct {
	stubHandler
	onMemberLeft func() error
}

func (h *memberLeftErrorHandler) HandleMemberLeft(ctx context.Context, leftUserIDs []string) error {
	if h.onMemberLeft != nil {
		return h.onMemberLeft()
	}
	return nil
}

func TestMemberLeft_EmptyMembers(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	done := make(chan struct{})
	handler := &memberLeftHandler{onCall: func() { close(done) }}
	s.RegisterHandler(handler)

	body := `{
		"events": [{
			"type": "memberLeft",
			"source": {"type": "group", "groupId": "C1234567890abcdef"},
			"timestamp": 1625000000000,
			"left": {"members": []}
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

	assert.Empty(t, handler.leftUserIDs)
}
