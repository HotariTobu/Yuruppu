package history

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultStorageTimeout is the default timeout for storage operations.
// Per NFR-001: storage operations should add at most 100ms to message processing latency.
const DefaultStorageTimeout = 100 * time.Millisecond

// TimeoutStorage wraps a Storage implementation with timeout enforcement.
// Per NFR-001: storage operations should not add more than 100ms to message processing latency.
type TimeoutStorage struct {
	inner   Storage
	timeout time.Duration
}

// NewTimeoutStorage creates a new TimeoutStorage with the specified timeout.
// If timeout is 0, DefaultStorageTimeout (100ms) is used.
func NewTimeoutStorage(inner Storage, timeout time.Duration) *TimeoutStorage {
	if timeout == 0 {
		timeout = DefaultStorageTimeout
	}
	return &TimeoutStorage{
		inner:   inner,
		timeout: timeout,
	}
}

// GetHistory retrieves conversation history with timeout enforcement.
// The actual timeout is min(ctx deadline, s.timeout). If ctx already has
// a shorter deadline, that takes precedence.
func (s *TimeoutStorage) GetHistory(ctx context.Context, sourceID string) ([]Message, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	messages, err := s.inner.GetHistory(ctx, sourceID)
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return nil, &StorageTimeoutError{Message: fmt.Sprintf("GetHistory timed out after %v", s.timeout)}
	}
	return messages, err
}

// AppendMessages saves messages with timeout enforcement.
// The actual timeout is min(ctx deadline, s.timeout). If ctx already has
// a shorter deadline, that takes precedence.
func (s *TimeoutStorage) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg Message) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err := s.inner.AppendMessages(ctx, sourceID, userMsg, botMsg)
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		return &StorageTimeoutError{Message: fmt.Sprintf("AppendMessages timed out after %v", s.timeout)}
	}
	return err
}

// Close releases storage resources.
func (s *TimeoutStorage) Close(ctx context.Context) error {
	return s.inner.Close(ctx)
}
