package line_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
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

// testTimeout is the default timeout for tests.
const testTimeout = 30 * time.Second

// discardLogger returns a logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// computeSignature computes the LINE webhook signature for a given body and secret.
func computeSignature(body []byte, channelSecret string) string {
	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// =============================================================================
// NewServer Tests
// =============================================================================

// TestNewServer tests server creation with various inputs.
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

			server, err := line.NewServer(tt.channelSecret, testTimeout, discardLogger())

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

// TestNewServer_ZeroTimeout tests that zero timeout is rejected.
func TestNewServer_ZeroTimeout(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", 0, discardLogger())

	require.Error(t, err, "zero timeout should be rejected")
	assert.Nil(t, server)

	var configErr *line.ConfigError
	require.ErrorAs(t, err, &configErr)
	assert.Equal(t, "timeout", configErr.Variable)
}

// TestNewServer_NegativeTimeout tests that negative timeout is rejected.
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
// OnMessage Tests
// =============================================================================

// TestServer_OnMessage tests callback registration.
func TestServer_OnMessage(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", testTimeout, discardLogger())
	require.NoError(t, err)

	callback := func(ctx context.Context, msg line.Message) error {
		return nil
	}

	// Should not panic
	server.OnMessage(callback)

	// Registration should be idempotent - can register again without error
	server.OnMessage(callback)
}

// =============================================================================
// Signature Verification Tests
// =============================================================================

// TestServer_HandleWebhook_InvalidSignature tests signature verification.
// AC-002: Signature is verified synchronously.
func TestServer_HandleWebhook_InvalidSignature(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", testTimeout, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", "invalid-signature")

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	// Should return 400 Bad Request for invalid signature
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestServer_HandleWebhook_MissingSignature tests missing signature header.
func TestServer_HandleWebhook_MissingSignature(t *testing.T) {
	t.Parallel()

	server, err := line.NewServer("test-secret", testTimeout, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	// No X-Line-Signature header

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	// Should return 400 Bad Request for missing signature
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestServer_HandleWebhook_ValidSignature_EmptyEvents tests valid signature with no events.
// AC-002: Events are parsed synchronously, HTTP 200 is returned.
func TestServer_HandleWebhook_ValidSignature_EmptyEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	body := `{"events":[]}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Message Processing Tests
// =============================================================================

// TestServer_HandleWebhook_CallbackInvoked tests that callback is invoked for message events.
// AC-002/AC-003: Callback is invoked asynchronously for each MessageEvent.
func TestServer_HandleWebhook_CallbackInvoked(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	var mu sync.Mutex
	var receivedMessages []line.Message
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		mu.Lock()
		receivedMessages = append(receivedMessages, msg)
		mu.Unlock()
		close(callbackDone)
		return nil
	})

	// Valid webhook payload with a text message
	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {
					"type": "user",
					"userId": "U1234567890"
				},
				"timestamp": 1625000000000,
				"message": {
					"type": "text",
					"id": "12345",
					"text": "Hello, World!"
				}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()
	server.HandleWebhook(w, req)

	// HTTP 200 should be returned immediately (before callback completes)
	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for callback to be invoked
	select {
	case <-callbackDone:
		// Callback was invoked
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked within timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, receivedMessages, 1)
	assert.Equal(t, "test-reply-token", receivedMessages[0].ReplyToken)
	assert.Equal(t, "text", receivedMessages[0].Type)
	assert.Equal(t, "Hello, World!", receivedMessages[0].Text)
	assert.Equal(t, "U1234567890", receivedMessages[0].UserID)
}

// TestServer_HandleWebhook_MultipleEvents tests multiple message events in one webhook.
// AC-002: Each MessageEvent spawns a new goroutine.
func TestServer_HandleWebhook_MultipleEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	var mu sync.Mutex
	var receivedMessages []line.Message
	var wg sync.WaitGroup
	wg.Add(2) // Expecting 2 message events

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		mu.Lock()
		receivedMessages = append(receivedMessages, msg)
		mu.Unlock()
		wg.Done()
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "token-1",
				"source": {"type": "user", "userId": "U123"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "First"}
			},
			{
				"type": "message",
				"replyToken": "token-2",
				"source": {"type": "user", "userId": "U456"},
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

	// Wait for both callbacks to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("not all callbacks were invoked within timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, receivedMessages, 2)
	// Order may vary due to concurrent execution
	texts := []string{receivedMessages[0].Text, receivedMessages[1].Text}
	assert.Contains(t, texts, "First")
	assert.Contains(t, texts, "Second")
}

// TestServer_HandleWebhook_NonMessageEvents tests that non-message events are ignored.
func TestServer_HandleWebhook_NonMessageEvents(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	callbackCalled := false
	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		callbackCalled = true
		return nil
	})

	// Follow event (not a message event)
	body := `{
		"events": [
			{
				"type": "follow",
				"replyToken": "test-token",
				"source": {"type": "user", "userId": "U123"},
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

	// Give some time for any potential callback invocation
	time.Sleep(50 * time.Millisecond)

	assert.False(t, callbackCalled, "callback should not be called for non-message events")
}

// TestServer_HandleWebhook_ImageMessage tests handling of image messages.
func TestServer_HandleWebhook_ImageMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	var receivedMsg line.Message
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		receivedMsg = msg
		close(callbackDone)
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
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
	case <-callbackDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked")
	}

	assert.Equal(t, "image", receivedMsg.Type)
	assert.Equal(t, "[User sent an image]", receivedMsg.Text)
}

// TestServer_HandleWebhook_StickerMessage tests handling of sticker messages.
func TestServer_HandleWebhook_StickerMessage(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	var receivedMsg line.Message
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		receivedMsg = msg
		close(callbackDone)
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
				"timestamp": 1625000000000,
				"message": {"type": "sticker", "id": "12345", "packageId": "1", "stickerId": "1"}
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
	case <-callbackDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked")
	}

	assert.Equal(t, "sticker", receivedMsg.Type)
	assert.Equal(t, "[User sent a sticker]", receivedMsg.Text)
}

// TestServer_HandleWebhook_NoCallback tests behavior when no callback is registered.
func TestServer_HandleWebhook_NoCallback(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	// No callback registered

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	// Should not panic when no callback is registered
	assert.NotPanics(t, func() {
		server.HandleWebhook(w, req)
	})

	// HTTP 200 should still be returned
	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Async Execution and Context Tests
// =============================================================================

// TestServer_HandleWebhook_AsyncExecution tests that HTTP response is sent before callback completes.
// AC-002: HTTP response time does not depend on callback execution time.
func TestServer_HandleWebhook_AsyncExecution(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	callbackStarted := make(chan struct{})
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		close(callbackStarted)
		// Simulate slow processing
		time.Sleep(500 * time.Millisecond)
		close(callbackDone)
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	// Measure time for HandleWebhook to return
	start := time.Now()
	server.HandleWebhook(w, req)
	responseTime := time.Since(start)

	// HTTP response should be sent quickly (before callback completes)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, responseTime, 100*time.Millisecond, "HTTP response should be sent before callback completes")

	// Wait for callback to complete
	select {
	case <-callbackDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not complete within timeout")
	}
}

// TestServer_HandleWebhook_ContextWithTimeout tests that callback receives context with timeout.
// Implementation note: Context propagation with configurable timeout.
func TestServer_HandleWebhook_ContextWithTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	var receivedCtx context.Context
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		receivedCtx = ctx
		close(callbackDone)
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
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

	select {
	case <-callbackDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked")
	}

	// Context should have a deadline
	_, hasDeadline := receivedCtx.Deadline()
	assert.True(t, hasDeadline, "context should have a timeout deadline")
}

// TestServer_NewServerWithTimeout tests timeout passed to NewServer.
func TestServer_NewServerWithTimeout(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	customTimeout := 10 * time.Second
	server, err := line.NewServer(channelSecret, customTimeout, discardLogger())
	require.NoError(t, err)

	var receivedCtx context.Context
	callbackDone := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		receivedCtx = ctx
		close(callbackDone)
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
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
	case <-callbackDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked")
	}

	deadline, hasDeadline := receivedCtx.Deadline()
	assert.True(t, hasDeadline, "context should have a timeout deadline")

	// The deadline should be approximately customTimeout from now
	// Allow some margin for test execution
	timeUntilDeadline := time.Until(deadline)
	assert.True(t, timeUntilDeadline > 5*time.Second && timeUntilDeadline <= customTimeout,
		"deadline should be approximately %v from now, got %v", customTimeout, timeUntilDeadline)
}

// TestServer_CallbackTimeout_Enforcement tests that long-running callbacks are cancelled.
func TestServer_CallbackTimeout_Enforcement(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	shortTimeout := 100 * time.Millisecond
	server, err := line.NewServer(channelSecret, shortTimeout, discardLogger())
	require.NoError(t, err)

	callbackStarted := make(chan struct{})
	contextCancelled := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		close(callbackStarted)

		// Wait for context cancellation
		select {
		case <-ctx.Done():
			close(contextCancelled)
			return ctx.Err()
		case <-time.After(5 * time.Second):
			t.Error("context was not cancelled within timeout")
		}
		return nil
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
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

	// Wait for callback to start
	select {
	case <-callbackStarted:
	case <-time.After(1 * time.Second):
		t.Fatal("callback was not invoked")
	}

	// Wait for context to be cancelled due to timeout
	select {
	case <-contextCancelled:
		// Success - timeout was enforced
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled by timeout")
	}
}

// TestServer_HandleWebhook_PanicRecovery tests panic recovery in callback.
// AC-008: Panics are recovered using defer/recover.
func TestServer_HandleWebhook_PanicRecovery(t *testing.T) {
	t.Parallel()

	channelSecret := "test-secret"
	server, err := line.NewServer(channelSecret, testTimeout, discardLogger())
	require.NoError(t, err)

	panicTriggered := make(chan struct{})

	server.OnMessage(func(ctx context.Context, msg line.Message) error {
		close(panicTriggered)
		panic("test panic")
	})

	body := `{
		"events": [
			{
				"type": "message",
				"replyToken": "test-reply-token",
				"source": {"type": "user", "userId": "U123"},
				"timestamp": 1625000000000,
				"message": {"type": "text", "id": "1", "text": "test"}
			}
		]
	}`
	signature := computeSignature([]byte(body), channelSecret)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Line-Signature", signature)

	w := httptest.NewRecorder()

	// Should not panic even if callback panics
	assert.NotPanics(t, func() {
		server.HandleWebhook(w, req)
	})

	// HTTP 200 should still be returned
	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for panic to be triggered
	select {
	case <-panicTriggered:
		// Panic was triggered and recovered
	case <-time.After(2 * time.Second):
		t.Fatal("callback was not invoked")
	}
}
