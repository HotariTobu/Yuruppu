//go:build !integration

package history_test

import (
	"context"
	"testing"
	"time"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TimeoutStorage Constructor Tests
// =============================================================================

// TestNewTimeoutStorage tests the TimeoutStorage constructor.
// NFR-001: storage operations should add at most 100ms to message processing latency
func TestNewTimeoutStorage(t *testing.T) {
	t.Run("should use default timeout when zero is provided", func(t *testing.T) {
		// Given: Inner storage
		inner := &delayMockStorage{}

		// When: Create with zero timeout
		storage := history.NewTimeoutStorage(inner, 0)

		// Then: Should use default timeout (verified via behavior in other tests)
		assert.NotNil(t, storage)
	})

	t.Run("should use custom timeout when provided", func(t *testing.T) {
		// Given: Inner storage
		inner := &delayMockStorage{}

		// When: Create with custom timeout
		storage := history.NewTimeoutStorage(inner, 50*time.Millisecond)

		// Then: Should create storage
		assert.NotNil(t, storage)
	})
}

// =============================================================================
// GetHistory Timeout Tests
// =============================================================================

// TestTimeoutStorage_GetHistory_EnforcesTimeout tests that GetHistory enforces timeout.
// NFR-001: storage operations should add at most 100ms to message processing latency
func TestTimeoutStorage_GetHistory_EnforcesTimeout(t *testing.T) {
	t.Run("should timeout when inner operation is slow", func(t *testing.T) {
		// Given: Inner storage that is slow
		inner := &delayMockStorage{
			getHistoryDelay: 200 * time.Millisecond,
			messages: []history.Message{
				{Role: "user", Content: "Test", Timestamp: time.Now()},
			},
		}
		storage := history.NewTimeoutStorage(inner, 50*time.Millisecond)

		// When: GetHistory is called
		ctx := context.Background()
		messages, err := storage.GetHistory(ctx, "U123")

		// Then: Should timeout with StorageTimeoutError
		require.Error(t, err)
		assert.Nil(t, messages)

		var timeoutErr *history.StorageTimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})

	t.Run("should succeed when inner operation is fast", func(t *testing.T) {
		// Given: Inner storage that is fast
		inner := &delayMockStorage{
			getHistoryDelay: 10 * time.Millisecond,
			messages: []history.Message{
				{Role: "user", Content: "Test", Timestamp: time.Now()},
			},
		}
		storage := history.NewTimeoutStorage(inner, 100*time.Millisecond)

		// When: GetHistory is called
		ctx := context.Background()
		messages, err := storage.GetHistory(ctx, "U123")

		// Then: Should succeed
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "Test", messages[0].Content)
	})

	t.Run("should use default 100ms timeout", func(t *testing.T) {
		// Given: Inner storage that is slow but under default timeout
		inner := &delayMockStorage{
			getHistoryDelay: 50 * time.Millisecond,
			messages: []history.Message{
				{Role: "user", Content: "Test", Timestamp: time.Now()},
			},
		}
		storage := history.NewTimeoutStorage(inner, 0) // Use default

		// When: GetHistory is called
		ctx := context.Background()
		messages, err := storage.GetHistory(ctx, "U123")

		// Then: Should succeed (50ms < 100ms default)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
	})
}

// TestTimeoutStorage_GetHistory_PropagatesContextCancellation tests context handling.
func TestTimeoutStorage_GetHistory_PropagatesContextCancellation(t *testing.T) {
	t.Run("should respect parent context cancellation", func(t *testing.T) {
		// Given: Already cancelled parent context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		inner := &delayMockStorage{
			messages: []history.Message{{Role: "user", Content: "Test", Timestamp: time.Now()}},
		}
		storage := history.NewTimeoutStorage(inner, 100*time.Millisecond)

		// When: GetHistory is called with cancelled context
		messages, err := storage.GetHistory(ctx, "U123")

		// Then: Should fail due to cancelled context
		require.Error(t, err)
		assert.Nil(t, messages)
	})
}

// =============================================================================
// AppendMessages Timeout Tests
// =============================================================================

// TestTimeoutStorage_AppendMessages_EnforcesTimeout tests that AppendMessages enforces timeout.
// NFR-001: storage operations should add at most 100ms to message processing latency
func TestTimeoutStorage_AppendMessages_EnforcesTimeout(t *testing.T) {
	t.Run("should timeout when inner operation is slow", func(t *testing.T) {
		// Given: Inner storage that is slow
		inner := &delayMockStorage{
			appendDelay: 200 * time.Millisecond,
		}
		storage := history.NewTimeoutStorage(inner, 50*time.Millisecond)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: AppendMessages is called
		ctx := context.Background()
		err := storage.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should timeout with StorageTimeoutError
		require.Error(t, err)

		var timeoutErr *history.StorageTimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})

	t.Run("should succeed when inner operation is fast", func(t *testing.T) {
		// Given: Inner storage that is fast
		inner := &delayMockStorage{
			appendDelay: 10 * time.Millisecond,
		}
		storage := history.NewTimeoutStorage(inner, 100*time.Millisecond)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: AppendMessages is called
		ctx := context.Background()
		err := storage.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should succeed
		require.NoError(t, err)
	})

	t.Run("should use default 100ms timeout", func(t *testing.T) {
		// Given: Inner storage that is slow but under default timeout
		inner := &delayMockStorage{
			appendDelay: 50 * time.Millisecond,
		}
		storage := history.NewTimeoutStorage(inner, 0) // Use default

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: AppendMessages is called
		ctx := context.Background()
		err := storage.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should succeed (50ms < 100ms default)
		require.NoError(t, err)
	})
}

// TestTimeoutStorage_AppendMessages_PropagatesContextCancellation tests context handling.
func TestTimeoutStorage_AppendMessages_PropagatesContextCancellation(t *testing.T) {
	t.Run("should respect parent context cancellation", func(t *testing.T) {
		// Given: Already cancelled parent context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		inner := &delayMockStorage{}
		storage := history.NewTimeoutStorage(inner, 100*time.Millisecond)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: AppendMessages is called with cancelled context
		err := storage.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should fail due to cancelled context
		require.Error(t, err)
	})
}

// =============================================================================
// Close Tests
// =============================================================================

// TestTimeoutStorage_Close tests the Close method.
func TestTimeoutStorage_Close(t *testing.T) {
	t.Run("should delegate to inner storage", func(t *testing.T) {
		// Given: Inner storage
		inner := &delayMockStorage{}
		storage := history.NewTimeoutStorage(inner, 100*time.Millisecond)

		// When: Close is called
		ctx := context.Background()
		err := storage.Close(ctx)

		// Then: Should succeed
		require.NoError(t, err)
		assert.True(t, inner.closeCalled)
	})
}

// =============================================================================
// NFR-001 Verification Tests
// =============================================================================

// TestTimeoutStorage_NFR001_LatencyBudget verifies the 100ms latency budget.
// NFR-001: storage operations should add at most 100ms to message processing latency
func TestTimeoutStorage_NFR001_LatencyBudget(t *testing.T) {
	t.Run("GetHistory should not exceed 100ms with default timeout", func(t *testing.T) {
		// Given: Inner storage that would exceed 100ms
		inner := &delayMockStorage{
			getHistoryDelay: 150 * time.Millisecond,
			messages:        []history.Message{{Role: "user", Content: "Test", Timestamp: time.Now()}},
		}
		storage := history.NewTimeoutStorage(inner, 0) // Use default 100ms

		// When: GetHistory is called and timed
		ctx := context.Background()
		start := time.Now()
		_, err := storage.GetHistory(ctx, "U123")
		elapsed := time.Since(start)

		// Then: Should timeout within 100ms (with some tolerance for test execution)
		require.Error(t, err)
		assert.Less(t, elapsed, 120*time.Millisecond, "operation should timeout within 100ms + tolerance")
	})

	t.Run("AppendMessages should not exceed 100ms with default timeout", func(t *testing.T) {
		// Given: Inner storage that would exceed 100ms
		inner := &delayMockStorage{
			appendDelay: 150 * time.Millisecond,
		}
		storage := history.NewTimeoutStorage(inner, 0) // Use default 100ms

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: AppendMessages is called and timed
		ctx := context.Background()
		start := time.Now()
		err := storage.AppendMessages(ctx, "U123", userMsg, botMsg)
		elapsed := time.Since(start)

		// Then: Should timeout within 100ms (with some tolerance for test execution)
		require.Error(t, err)
		assert.Less(t, elapsed, 120*time.Millisecond, "operation should timeout within 100ms + tolerance")
	})
}

// =============================================================================
// Mock Implementations
// =============================================================================

// delayMockStorage implements history.Storage for testing timeout behavior.
type delayMockStorage struct {
	messages        []history.Message
	getHistoryDelay time.Duration
	appendDelay     time.Duration
	closeCalled     bool
}

func (m *delayMockStorage) GetHistory(ctx context.Context, sourceID string) ([]history.Message, error) {
	if m.getHistoryDelay > 0 {
		select {
		case <-time.After(m.getHistoryDelay):
			// Delay completed
		case <-ctx.Done():
			// Return context error (simulates real GCS behavior)
			return nil, ctx.Err()
		}
	}

	// Check for cancellation after delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.messages, nil
}

func (m *delayMockStorage) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg history.Message) error {
	if m.appendDelay > 0 {
		select {
		case <-time.After(m.appendDelay):
			// Delay completed
		case <-ctx.Done():
			// Return context error (simulates real GCS behavior)
			return ctx.Err()
		}
	}

	// Check for cancellation after delay
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}

func (m *delayMockStorage) Close(ctx context.Context) error {
	m.closeCalled = true
	return nil
}
