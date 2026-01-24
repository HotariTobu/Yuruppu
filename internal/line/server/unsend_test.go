package server_test

import (
	"context"
	"sync"
	"testing"
	"time"
	"yuruppu/internal/line"
	"yuruppu/internal/line/server"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// FR-001: System receives and processes LINE unsend webhook events
// =============================================================================

func TestServer_HandleWebhook_UnsendEvent_OneOnOneChat(t *testing.T) {
	t.Parallel()

	// AC-001: Unsend event triggers message removal [FR-001]
	// Given: A user has sent a message in a 1:1 chat
	// When: The user unsends that message
	// Then: The unsend webhook event is received and dispatched

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code, "webhook should return 200 OK")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.unsendMessages, 1, "HandleUnsend should be called once")
	assert.Equal(t, "12345678901234", handler.unsendMessages[0].messageID,
		"messageID should be extracted from unsend event")
	assert.Equal(t, "U1234567890abcdef", handler.unsendMessages[0].sourceID,
		"sourceID should be userId for 1:1 chat")
	assert.Equal(t, "U1234567890abcdef", handler.unsendMessages[0].userID,
		"userID should match source userId")
	assert.Equal(t, line.ChatTypeOneOnOne, handler.unsendMessages[0].chatType,
		"chatType should be one-on-one for user source")
}

func TestServer_HandleWebhook_UnsendEvent_GroupChat(t *testing.T) {
	t.Parallel()

	// AC-003: Unsend in group chat [FR-001, FR-003]
	// Given: A message exists in a group chat history
	// When: A user unsends that message
	// Then: The message is removed from the group's history

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "group", "groupId": "C1234567890abcdef", "userId": "U9876543210fedcba"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "98765432109876"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code, "webhook should return 200 OK")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.unsendMessages, 1, "HandleUnsend should be called once")
	assert.Equal(t, "98765432109876", handler.unsendMessages[0].messageID,
		"messageID should be extracted from unsend event")
	assert.Equal(t, "C1234567890abcdef", handler.unsendMessages[0].sourceID,
		"sourceID should be groupId for group chat")
	assert.Equal(t, "U9876543210fedcba", handler.unsendMessages[0].userID,
		"userID should identify who unsent the message")
	assert.Equal(t, line.ChatTypeGroup, handler.unsendMessages[0].chatType,
		"chatType should be group for group source")
}

func TestServer_HandleWebhook_UnsendEvent_RoomChat(t *testing.T) {
	t.Parallel()

	// AC-003: Unsend in room chat (similar to group chat)
	// Given: A message exists in a room chat history
	// When: A user unsends that message
	// Then: The message is removed from the room's history

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "room", "roomId": "R1234567890abcdef", "userId": "U9876543210fedcba"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "11111111111111"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code, "webhook should return 200 OK")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked within timeout")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.unsendMessages, 1, "HandleUnsend should be called once")
	assert.Equal(t, "11111111111111", handler.unsendMessages[0].messageID,
		"messageID should be extracted from unsend event")
	assert.Equal(t, "R1234567890abcdef", handler.unsendMessages[0].sourceID,
		"sourceID should be roomId for room chat")
	assert.Equal(t, "U9876543210fedcba", handler.unsendMessages[0].userID,
		"userID should identify who unsent the message")
	assert.Equal(t, line.ChatTypeGroup, handler.unsendMessages[0].chatType,
		"chatType should be group for room source")
}

func TestServer_HandleWebhook_UnsendEvent_MultipleHandlers(t *testing.T) {
	t.Parallel()

	// Given: Multiple handlers are registered
	// When: An unsend event is received
	// Then: All handlers receive the event

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler1 := &mockHandler{}
	handler2 := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)

	handler1.onUnsend = func(ctx context.Context, messageID string) error {
		wg.Done()
		return nil
	}
	handler2.onUnsend = func(ctx context.Context, messageID string) error {
		wg.Done()
		return nil
	}

	s.RegisterHandler(handler1)
	s.RegisterHandler(handler2)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code)

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
	handler2.mu.Lock()
	defer handler1.mu.Unlock()
	defer handler2.mu.Unlock()

	assert.Len(t, handler1.unsendMessages, 1, "handler1 should receive event")
	assert.Len(t, handler2.unsendMessages, 1, "handler2 should receive event")
}

func TestServer_HandleWebhook_UnsendEvent_AsyncExecution(t *testing.T) {
	t.Parallel()

	// Given: Handler takes time to process unsend event
	// When: Webhook is called
	// Then: HTTP 200 is returned before handler completes

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerDone := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		time.Sleep(500 * time.Millisecond)
		close(handlerDone)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()

	start := time.Now()
	s.HandleWebhook(w, req)
	responseTime := time.Since(start)

	assert.Equal(t, 200, w.Code)
	assert.Less(t, responseTime, 100*time.Millisecond,
		"HTTP response should be sent before handler completes")

	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not complete")
	}
}

func TestServer_HandleWebhook_UnsendEvent_ContextTimeout(t *testing.T) {
	t.Parallel()

	// Given: Server configured with short timeout
	// When: Unsend handler runs longer than timeout
	// Then: Context is cancelled

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	s, err := server.NewServer(channelSecret, shortTimeout, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	handlerStarted := make(chan struct{})
	contextCancelled := make(chan struct{})

	handler.onUnsend = func(ctx context.Context, messageID string) error {
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
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
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

func TestServer_HandleWebhook_UnsendEvent_PanicRecovery(t *testing.T) {
	t.Parallel()

	// Given: Handler panics during unsend processing
	// When: Webhook is called
	// Then: Server recovers and logs the panic

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	panicTriggered := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		close(panicTriggered)
		panic("test panic in unsend handler")
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()

	assert.NotPanics(t, func() {
		s.HandleWebhook(w, req)
	}, "server should not propagate panic")

	assert.Equal(t, 200, w.Code)

	select {
	case <-panicTriggered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}
}

func TestServer_HandleWebhook_UnsendEvent_NoHandler(t *testing.T) {
	t.Parallel()

	// Given: No handlers are registered
	// When: An unsend event is received
	// Then: Server does not crash

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()

	assert.NotPanics(t, func() {
		s.HandleWebhook(w, req)
	})

	assert.Equal(t, 200, w.Code)
}

func TestServer_HandleWebhook_MultipleUnsendEvents(t *testing.T) {
	t.Parallel()

	// Given: Multiple unsend events in a single webhook request
	// When: Webhook is called
	// Then: All unsend events are dispatched

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	var wg sync.WaitGroup
	wg.Add(2)

	handler.onUnsend = func(ctx context.Context, messageID string) error {
		wg.Done()
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			},
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000001,
				"unsend": {"messageId": "98765432109876"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("not all events were processed")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.unsendMessages, 2, "both unsend events should be handled")
	messageIDs := []string{handler.unsendMessages[0].messageID, handler.unsendMessages[1].messageID}
	assert.Contains(t, messageIDs, "12345678901234")
	assert.Contains(t, messageIDs, "98765432109876")
}

func TestServer_HandleWebhook_MixedEventsWithUnsend(t *testing.T) {
	t.Parallel()

	// Given: Webhook contains both message and unsend events
	// When: Webhook is called
	// Then: Both event types are dispatched correctly

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
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		wg.Done()
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
				"message": {"type": "text", "id": "msg123", "text": "Hello"}
			},
			{
				"type": "unsend",
				"source": {"type": "user", "userId": "U1234567890abcdef"},
				"timestamp": 1625000000001,
				"unsend": {"messageId": "msg123"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("not all events were processed")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	assert.Len(t, handler.textMessages, 1, "message event should be handled")
	assert.Len(t, handler.unsendMessages, 1, "unsend event should be handled")
	assert.Equal(t, "Hello", handler.textMessages[0].text)
	assert.Equal(t, "msg123", handler.unsendMessages[0].messageID)
}

func TestServer_HandleWebhook_UnsendEvent_MissingSource(t *testing.T) {
	t.Parallel()

	// Given: Unsend event without source field (edge case)
	// When: Webhook is called
	// Then: Server handles gracefully with empty context values

	channelSecret := "test-secret"
	s, err := server.NewServer(channelSecret, 30*time.Second, discardLogger())
	require.NoError(t, err)

	handler := &mockHandler{}
	done := make(chan struct{})
	handler.onUnsend = func(ctx context.Context, messageID string) error {
		close(done)
		return nil
	}
	s.RegisterHandler(handler)

	body := `{
		"events": [
			{
				"type": "unsend",
				"timestamp": 1625000000000,
				"unsend": {"messageId": "12345678901234"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := newWebhookRequest(body, signature)
	w := newRecorder()
	s.HandleWebhook(w, req)

	assert.Equal(t, 200, w.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not invoked")
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()

	require.Len(t, handler.unsendMessages, 1)
	assert.Equal(t, "12345678901234", handler.unsendMessages[0].messageID)
	assert.Equal(t, "", handler.unsendMessages[0].sourceID, "missing source should result in empty sourceID")
	assert.Equal(t, "", handler.unsendMessages[0].userID, "missing source should result in empty userID")
}
