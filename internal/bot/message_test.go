package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/bot"
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

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
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

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
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

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
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

		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, validHandlerConfig(), logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
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
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, mockMedia, mockAg, config, logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hello")

		require.NoError(t, err)
		assert.True(t, mockClient.showLoadingCalled)
		assert.Equal(t, 45*time.Second, mockClient.showLoadingTimeout, "timeout should match configured value")
	})
}
