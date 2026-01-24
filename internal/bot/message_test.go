package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/bot"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleImage Tests
// =============================================================================

func TestHandleImage(t *testing.T) {
	t.Run("success - downloads and stores image", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("fake-image-data"),
			mimeType: "image/jpeg",
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "Nice image!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "msg-456", mockClient.lastMessageID)
		assert.Equal(t, "user-123", mockMedia.lastSourceID)
		assert.Equal(t, []byte("fake-image-data"), mockMedia.lastData)
		assert.Equal(t, "image/jpeg", mockMedia.lastMIMEType)
	})

	t.Run("success - storage key format is sourceID/uuid", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("image-data"),
			mimeType: "image/png",
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "Nice!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "group-789", mockMedia.lastSourceID)
	})

	t.Run("fallback - download error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			err: errors.New("LINE API failed"),
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})

	t.Run("fallback - storage error uses placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			data:     []byte("image-data"),
			mimeType: "image/jpeg",
		}
		mockMedia := &mockMediaService{storeErr: errors.New("GCS failed")}
		mockAg := &mockAgent{response: "I see!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleImage(ctx, "msg-456")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an image, but an error occurred while loading]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// Delayed Loading Indicator Tests (FR-001, FR-002, FR-006, NFR-001, NFR-002)
// =============================================================================

// TestHandleMessage_DelayedLoadingIndicator tests the delayed loading indicator
// behavior from the spec: 20260108-feat-typing-indicator
func TestHandleMessage_DelayedLoadingIndicator(t *testing.T) {
	// AC-001: Loading indicator shown when processing exceeds delay
	// FR-001: If processing takes longer than delay, ShowLoadingAnimation is called
	t.Run("AC-001: shows loading indicator when processing exceeds delay in 1:1 chat", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		// Mock agent that takes longer than delay to respond
		mockAg := &mockAgent{
			response:     "Slow response",
			processDelay: 200 * time.Millisecond, // Takes 200ms
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond, // Delay is 50ms
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		// 1:1 chat: sourceID == userID
		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		// Verify ShowLoadingAnimation was called
		assert.True(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should be called when processing exceeds delay")
		assert.Equal(t, "user-123", mockClient.showLoadingChatID, "chatID should be the sourceID (user-123)")
		assert.Equal(t, 30*time.Second, mockClient.showLoadingTimeout, "timeout should match config")
	})

	// AC-006: Loading indicator NOT shown when processing completes quickly
	// FR-006: If processing completes before delay, ShowLoadingAnimation is NOT called
	t.Run("AC-006: does NOT show loading indicator when processing completes before delay", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		// Mock agent that responds quickly
		mockAg := &mockAgent{
			response:     "Fast response",
			processDelay: 10 * time.Millisecond, // Completes in 10ms
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   100 * time.Millisecond, // Delay is 100ms
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		// 1:1 chat: sourceID == userID
		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		// Verify ShowLoadingAnimation was NOT called
		assert.False(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should NOT be called when processing completes before delay")
	})

	// AC-003: Loading indicator NOT called for group chats
	// FR-002: Only call ShowLoadingAnimation for 1:1 chats (sourceID == userID)
	t.Run("AC-003: does NOT show loading indicator in group chat even if processing is slow", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		// Mock agent that takes longer than delay
		mockAg := &mockAgent{
			response:     "Slow response",
			processDelay: 200 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		// Group chat: sourceID != userID
		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		// Verify ShowLoadingAnimation was NOT called for group chat
		assert.False(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should NOT be called in group chat")
	})

	// NFR-001: API call is non-blocking
	// AC-005: API call does not block message processing
	t.Run("AC-005/NFR-001: ShowLoadingAnimation call does not block message processing", func(t *testing.T) {
		mockStore := newMockStorage()
		// Mock client where ShowLoadingAnimation takes a long time
		mockClient := &mockLineClient{
			showLoadingDelay: 500 * time.Millisecond, // API call takes 500ms
		}
		mockMedia := &mockMediaService{}
		// Mock agent that responds in 100ms
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 100 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")

		start := time.Now()
		err = h.HandleText(ctx, "Hello")
		elapsed := time.Since(start)

		require.NoError(t, err)
		// Message processing should complete in ~100ms (agent delay), not 500ms (API delay)
		// Allow some margin for test execution overhead
		assert.Less(t, elapsed, 300*time.Millisecond, "message processing should not wait for ShowLoadingAnimation to complete")
		assert.True(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should still be called asynchronously")
	})

	// AC-004, NFR-002: API failure does not block processing and logs WARN
	// FR-004: If ShowLoadingAnimation fails, processing continues
	t.Run("AC-004/NFR-002: ShowLoadingAnimation failure does not prevent message processing", func(t *testing.T) {
		mockStore := newMockStorage()
		// Mock client that fails ShowLoadingAnimation
		mockClient := &mockLineClient{
			showLoadingErr: errors.New("LINE API error"),
		}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 100 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		// Processing should succeed even though ShowLoadingAnimation failed
		require.NoError(t, err, "message processing should succeed even if ShowLoadingAnimation fails")
		assert.True(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should have been called")
	})
}

// TestHandleMessage_LoadingIndicatorEdgeCases tests edge cases and boundary conditions
func TestHandleMessage_LoadingIndicatorEdgeCases(t *testing.T) {
	t.Run("zero delay config - indicator shown immediately if processing is slow", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 50 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   0, // Zero delay = immediate
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		// With zero delay, indicator should be shown immediately for any processing time
		assert.True(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should be called with zero delay")
	})

	t.Run("room chat (sourceID != userID) does not show indicator", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 200 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		// Room chat: sourceID != userID
		ctx := withLineContext(t.Context(), "reply-token", "room-456", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		assert.False(t, mockClient.showLoadingCalled, "ShowLoadingAnimation should NOT be called in room chat")
	})

	t.Run("context cancellation during processing stops goroutine cleanly", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 1 * time.Second, // Long delay
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		ctx = withLineContext(ctx, "reply-token", "user-123", "user-123")

		// Cancel context after a short delay
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err = h.HandleText(ctx, "Hello")

		// Processing should fail due to context cancellation
		require.Error(t, err, "processing should fail when context is cancelled")
	})

	t.Run("correct timeout value passed to ShowLoadingAnimation", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{}
		mockMedia := &mockMediaService{}
		mockAg := &mockAgent{
			response:     "Response",
			processDelay: 200 * time.Millisecond,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		config := bot.HandlerConfig{
			TypingIndicatorDelay:   50 * time.Millisecond,
			TypingIndicatorTimeout: 45 * time.Second, // Custom timeout
		}
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		assert.True(t, mockClient.showLoadingCalled)
		assert.Equal(t, 45*time.Second, mockClient.showLoadingTimeout, "timeout should match configured value")
	})
}

// =============================================================================
// HandleText Tests
// =============================================================================

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - generates response", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.NoError(t, err)
	})

	t.Run("agent error - returns error", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "LLM failed")
	})
}

// =============================================================================
// HandleSticker Tests
// =============================================================================

func TestHandler_HandleSticker(t *testing.T) {
	t.Run("converts sticker to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice sticker!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleSticker(ctx, "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a sticker]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// HandleVideo Tests
// =============================================================================

func TestHandler_HandleVideo(t *testing.T) {
	t.Run("converts video to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I see a video!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleVideo(ctx, "msg-789")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a video]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// HandleAudio Tests
// =============================================================================

func TestHandler_HandleAudio(t *testing.T) {
	t.Run("converts audio to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I hear audio!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleAudio(ctx, "msg-101")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an audio]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// HandleLocation Tests
// =============================================================================

func TestHandler_HandleLocation(t *testing.T) {
	t.Run("converts location to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice place!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleLocation(ctx, 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a location]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// HandleFile Tests
// =============================================================================

func TestHandler_HandleFile(t *testing.T) {
	t.Run("converts file to text placeholder with filename", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I got your file!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleFile(ctx, "msg-123", "document.pdf", 1024)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a file: document.pdf]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// History Integration Tests
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message to history", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.NoError(t, err)
		// Verify storage was called once (user message only, assistant message is saved by reply tool)
		require.Equal(t, 1, mockStore.writeCallCount)
	})

	t.Run("does not respond when history read fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.readErr = errors.New("GCS read failed")
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read history")
	})

	t.Run("does not respond when user message storage fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.writeResults = []writeResult{{gen: 0, err: errors.New("GCS failed")}}
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write history")
	})

	t.Run("saves only user message when agent fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// User message is saved before agent is called
		assert.Equal(t, 1, mockStore.writeCallCount)
	})
}

// =============================================================================
// Error Chain Tests (errors.Is verification)
// =============================================================================

func TestHandler_ErrorChain(t *testing.T) {
	t.Run("agent error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		agentErr := errors.New("LLM generation failed")
		mockAg := &mockAgent{err: agentErr}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, agentErr), "error chain should contain original agent error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to generate response")
	})

	t.Run("storage read error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS bucket not found")
		mockStore.readErr = storageErr
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to load history")
	})

	t.Run("storage write error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS write quota exceeded")
		mockStore.writeResults = []writeResult{{gen: 0, err: storageErr}}
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to save user message to history")
	})
}

// =============================================================================
// Group Member Count Context Tests (FR-005)
// =============================================================================

func TestHandleMessage_GroupMemberCount(t *testing.T) {
	// AC-004: Member count passed to LLM [FR-005]
	t.Run("AC-004: group message includes user_count from stored group profile", func(t *testing.T) {
		// Given: A group with a stored member count
		mockGroupProfile := &mockGroupProfileService{
			profile: &groupprofile.GroupProfile{
				DisplayName: "Test Group",
				PictureURL:  "",
				UserCount:   15, // 15 members in the group
			},
		}
		mockAg := &mockAgent{response: "Hello group!"}

		h := newTestHandler(t).
			WithGroupProfile(mockGroupProfile).
			WithAgent(mockAg).
			Build()

		// When: A user sends a message in the group chat
		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err := h.HandleText(ctx, "Hi everyone!")

		// Then: The group member count is included in the context
		require.NoError(t, err)
		require.NotEmpty(t, mockAg.lastContextText, "context should be captured")
		assert.Contains(t, mockAg.lastContextText, "[context]", "context should start with [context]")
		assert.Contains(t, mockAg.lastContextText, "user_count: 15", "context should include user_count: 15")
		assert.Contains(t, mockAg.lastContextText, "chat_type: group", "context should indicate group chat")
	})

	// AC-005: Handle missing member count gracefully [FR-005]
	t.Run("AC-005: group message continues when group profile unavailable", func(t *testing.T) {
		// Given: A group message is received but member count is not stored
		mockGroupProfile := &mockGroupProfileService{
			getErr: errors.New("profile not found"),
		}
		mockAg := &mockAgent{response: "Hello group!"}

		h := newTestHandler(t).
			WithGroupProfile(mockGroupProfile).
			WithAgent(mockAg).
			Build()

		// When: The message is processed
		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err := h.HandleText(ctx, "Hi everyone!")

		// Then: Message processing continues normally
		require.NoError(t, err, "message processing should continue despite missing group profile")
		// And: LLM request is sent without user_count (graceful degradation)
		require.NotEmpty(t, mockAg.lastContextText, "context should still be sent")
		assert.NotContains(t, mockAg.lastContextText, "user_count:", "context should not include user_count when profile unavailable")
		assert.Contains(t, mockAg.lastContextText, "chat_type: group", "context should still indicate group chat")
	})

	t.Run("1:1 chat does not include user_count", func(t *testing.T) {
		// Given: A 1:1 chat (sourceID == userID)
		mockAg := &mockAgent{response: "Hello!"}

		h := newTestHandler(t).
			WithAgent(mockAg).
			Build()

		// When: A user sends a message in 1:1 chat
		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err := h.HandleText(ctx, "Hi!")

		// Then: The context does not include user_count for 1:1 chats
		require.NoError(t, err)
		require.NotEmpty(t, mockAg.lastContextText, "context should be captured")
		assert.NotContains(t, mockAg.lastContextText, "user_count:", "1:1 chat should not have user_count")
		assert.Contains(t, mockAg.lastContextText, "chat_type: 1-on-1", "context should indicate 1:1 chat")
	})

	t.Run("group profile service not called for 1:1 chats", func(t *testing.T) {
		// Given: A 1:1 chat
		mockGroupProfile := &mockGroupProfileService{}
		mockAg := &mockAgent{response: "Hello!"}

		h := newTestHandler(t).
			WithGroupProfile(mockGroupProfile).
			WithAgent(mockAg).
			Build()

		// When: A user sends a message in 1:1 chat
		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err := h.HandleText(ctx, "Hi!")

		// Then: Group profile service should not be called
		require.NoError(t, err)
		assert.Empty(t, mockGroupProfile.lastGroupID, "group profile service should not be called for 1:1 chats")
	})

	t.Run("group profile service called with correct groupID", func(t *testing.T) {
		// Given: A group chat with stored profile
		mockGroupProfile := &mockGroupProfileService{
			profile: &groupprofile.GroupProfile{
				DisplayName: "Test Group",
				UserCount:   42,
			},
		}
		mockAg := &mockAgent{response: "Hello group!"}

		h := newTestHandler(t).
			WithGroupProfile(mockGroupProfile).
			WithAgent(mockAg).
			Build()

		// When: A user sends a message in the group
		ctx := withLineContext(t.Context(), "reply-token", "group-456", "user-123")
		err := h.HandleText(ctx, "Hi everyone!")

		// Then: Group profile service is called with correct groupID
		require.NoError(t, err)
		assert.Equal(t, "group-456", mockGroupProfile.lastGroupID, "group profile should be fetched with correct groupID")
	})

	t.Run("different member counts are correctly reflected in context", func(t *testing.T) {
		tests := []struct {
			name            string
			userCount       int
			expectedContext string
		}{
			{
				name:            "small group with 3 members",
				userCount:       3,
				expectedContext: "user_count: 3",
			},
			{
				name:            "medium group with 25 members",
				userCount:       25,
				expectedContext: "user_count: 25",
			},
			{
				name:            "large group with 100 members",
				userCount:       100,
				expectedContext: "user_count: 100",
			},
			{
				name:            "very large group with 500 members",
				userCount:       500,
				expectedContext: "user_count: 500",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given: A group with specific member count
				mockGroupProfile := &mockGroupProfileService{
					profile: &groupprofile.GroupProfile{
						DisplayName: "Test Group",
						UserCount:   tt.userCount,
					},
				}
				mockAg := &mockAgent{response: "Hello!"}

				h := newTestHandler(t).
					WithGroupProfile(mockGroupProfile).
					WithAgent(mockAg).
					Build()

				// When: A message is sent
				ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
				err := h.HandleText(ctx, "Hi!")

				// Then: The correct user_count is in the context
				require.NoError(t, err)
				assert.Contains(t, mockAg.lastContextText, tt.expectedContext,
					"context should include correct user_count")
			})
		}
	})
}

// =============================================================================
// Context Format Verification Tests
// =============================================================================

func TestHandleMessage_ContextFormat(t *testing.T) {
	t.Run("context contains all required fields for group chat", func(t *testing.T) {
		// Given: A group chat with stored profile
		mockGroupProfile := &mockGroupProfileService{
			profile: &groupprofile.GroupProfile{
				DisplayName: "Test Group",
				UserCount:   20,
			},
		}
		mockAg := &mockAgent{response: "Hello!"}

		h := newTestHandler(t).
			WithGroupProfile(mockGroupProfile).
			WithAgent(mockAg).
			Build()

		// When: A message is sent
		ctx := withLineContext(t.Context(), "reply-token", "group-789", "user-123")
		err := h.HandleText(ctx, "Hi!")

		// Then: Context contains all required fields
		require.NoError(t, err)
		context := mockAg.lastContextText

		assert.Contains(t, context, "[context]", "should contain header")
		assert.Contains(t, context, "current_local_time:", "should contain timestamp")
		assert.Contains(t, context, "chat_type: group", "should contain chat type")
		assert.Contains(t, context, "user_count: 20", "should contain user count")

		// Verify format matches template expectations
		assert.Contains(t, context, "\n", "should be multi-line")
	})

	t.Run("context contains all required fields for 1:1 chat", func(t *testing.T) {
		// Given: A 1:1 chat
		mockAg := &mockAgent{response: "Hello!"}

		h := newTestHandler(t).
			WithAgent(mockAg).
			Build()

		// When: A message is sent
		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err := h.HandleText(ctx, "Hi!")

		// Then: Context contains all required fields
		require.NoError(t, err)
		context := mockAg.lastContextText

		assert.Contains(t, context, "[context]", "should contain header")
		assert.Contains(t, context, "current_local_time:", "should contain timestamp")
		assert.Contains(t, context, "chat_type: 1-on-1", "should contain chat type")
		assert.NotContains(t, context, "user_count:", "should not contain user_count for 1:1 chat")
	})
}
