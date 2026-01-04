package media_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/media"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewService Tests
// =============================================================================

func TestNewService(t *testing.T) {
	t.Run("creates service with valid dependencies", func(t *testing.T) {
		store := newMockStorage()
		logger := slog.New(slog.DiscardHandler)

		svc, err := media.NewService(store, logger)

		require.NoError(t, err)
		require.NotNil(t, svc)
	})

	t.Run("returns error when storage is nil", func(t *testing.T) {
		logger := slog.New(slog.DiscardHandler)

		svc, err := media.NewService(nil, logger)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "storage cannot be nil")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		store := newMockStorage()

		svc, err := media.NewService(store, nil)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "logger cannot be nil")
	})
}

// =============================================================================
// Store Tests
// =============================================================================

func TestService_Store(t *testing.T) {
	t.Run("stores media and returns storage key", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key, err := svc.Store(t.Context(), "user-123", []byte("image data"), "image/png")

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(key, "user-123/"))
		assert.Equal(t, 1, store.writeCallCount)
		assert.Equal(t, "image/png", store.lastWriteMIMEType)
		assert.Equal(t, []byte("image data"), store.lastWriteData)
	})

	t.Run("generates unique keys for multiple stores", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key1, err1 := svc.Store(t.Context(), "user-123", []byte("data1"), "image/png")
		key2, err2 := svc.Store(t.Context(), "user-123", []byte("data2"), "image/png")

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("returns error for empty sourceID", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key, err := svc.Store(t.Context(), "", []byte("data"), "image/png")

		require.Error(t, err)
		assert.Empty(t, key)
		assert.Contains(t, err.Error(), "invalid sourceID")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error for sourceID with path traversal", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key, err := svc.Store(t.Context(), "../etc/passwd", []byte("data"), "image/png")

		require.Error(t, err)
		assert.Empty(t, key)
		assert.Contains(t, err.Error(), "invalid sourceID")
		assert.Equal(t, 0, store.writeCallCount)
	})

	t.Run("returns error for sourceID with special characters", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key, err := svc.Store(t.Context(), "user/123", []byte("data"), "image/png")

		require.Error(t, err)
		assert.Empty(t, key)
		assert.Contains(t, err.Error(), "invalid sourceID")
	})

	t.Run("accepts valid LINE source IDs", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		// Typical LINE user ID format
		key, err := svc.Store(t.Context(), "U1234567890abcdef1234567890abcdef", []byte("data"), "image/png")

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(key, "U1234567890abcdef1234567890abcdef/"))
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("storage error")
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		key, err := svc.Store(t.Context(), "user-123", []byte("data"), "image/png")

		require.Error(t, err)
		assert.Empty(t, key)
		assert.Contains(t, err.Error(), "failed to write media to storage")
	})
}

// =============================================================================
// GetSignedURL Tests
// =============================================================================

func TestService_GetSignedURL(t *testing.T) {
	t.Run("returns signed URL from storage", func(t *testing.T) {
		store := newMockStorage()
		store.signedURL = "https://storage.example.com/signed-url"
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		url, err := svc.GetSignedURL(t.Context(), "user-123/uuid", 15*time.Minute)

		require.NoError(t, err)
		assert.Equal(t, "https://storage.example.com/signed-url", url)
		assert.Equal(t, "user-123/uuid", store.lastSignedURLKey)
		assert.Equal(t, "GET", store.lastSignedURLMethod)
		assert.Equal(t, 15*time.Minute, store.lastSignedURLTTL)
	})

	t.Run("returns error when storage fails", func(t *testing.T) {
		store := newMockStorage()
		store.signedURLErr = errors.New("signing error")
		svc, _ := media.NewService(store, slog.New(slog.DiscardHandler))

		url, err := svc.GetSignedURL(t.Context(), "user-123/uuid", 15*time.Minute)

		require.Error(t, err)
		assert.Empty(t, url)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockStorage struct {
	data              map[string][]byte
	writeErr          error
	writeCallCount    int
	lastWriteKey      string
	lastWriteMIMEType string
	lastWriteData     []byte

	signedURL           string
	signedURLErr        error
	lastSignedURLKey    string
	lastSignedURLMethod string
	lastSignedURLTTL    time.Duration
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	data, ok := m.data[key]
	if !ok {
		return nil, 0, nil
	}
	return data, 1, nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimeType string, data []byte, expectedGen int64) (int64, error) {
	m.writeCallCount++
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimeType
	m.lastWriteData = data
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.data[key] = data
	return 1, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	m.lastSignedURLKey = key
	m.lastSignedURLMethod = method
	m.lastSignedURLTTL = ttl
	if m.signedURLErr != nil {
		return "", m.signedURLErr
	}
	return m.signedURL, nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
