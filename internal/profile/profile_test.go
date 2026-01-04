package profile_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/profile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GetUserProfile Tests
// =============================================================================

func TestService_GetUserProfile(t *testing.T) {
	t.Run("returns profile from cache when available", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		// Set profile first (populates cache)
		p := &profile.UserProfile{DisplayName: "Alice"}
		err := svc.SetUserProfile(t.Context(), "user-123", p)
		require.NoError(t, err)

		// Reset read count to verify cache is used
		store.readCallCount = 0

		// Get profile (should hit cache)
		got, err := svc.GetUserProfile(t.Context(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "Alice", got.DisplayName)
		assert.Equal(t, 0, store.readCallCount, "should not read from storage when cached")
	})

	t.Run("returns profile from storage when not in cache", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		// Pre-populate storage directly (bypassing cache)
		p := &profile.UserProfile{DisplayName: "Bob", StatusMessage: "Hello"}
		data, _ := json.Marshal(p)
		store.data["user-456"] = data

		got, err := svc.GetUserProfile(t.Context(), "user-456")

		require.NoError(t, err)
		assert.Equal(t, "Bob", got.DisplayName)
		assert.Equal(t, "Hello", got.StatusMessage)
		assert.Equal(t, 1, store.readCallCount)
	})

	t.Run("caches profile after reading from storage", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		// Pre-populate storage
		p := &profile.UserProfile{DisplayName: "Charlie"}
		data, _ := json.Marshal(p)
		store.data["user-789"] = data

		// First read (from storage)
		_, err := svc.GetUserProfile(t.Context(), "user-789")
		require.NoError(t, err)
		assert.Equal(t, 1, store.readCallCount)

		// Second read (from cache)
		_, err = svc.GetUserProfile(t.Context(), "user-789")
		require.NoError(t, err)
		assert.Equal(t, 1, store.readCallCount, "should use cache on second read")
	})

	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage error")
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		got, err := svc.GetUserProfile(t.Context(), "user-123")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to read profile")
	})

	t.Run("returns error when JSON unmarshal fails", func(t *testing.T) {
		store := newMockStorage()
		store.data["user-123"] = []byte("invalid json")
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		got, err := svc.GetUserProfile(t.Context(), "user-123")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to unmarshal profile")
	})
}

// =============================================================================
// SetUserProfile Tests
// =============================================================================

func TestService_SetUserProfile(t *testing.T) {
	t.Run("stores profile to cache and storage", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		p := &profile.UserProfile{
			DisplayName:   "Alice",
			PictureURL:    "https://example.com/pic.jpg",
			StatusMessage: "Hello!",
		}
		err := svc.SetUserProfile(t.Context(), "user-123", p)

		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)
		assert.Equal(t, "user-123", store.lastWriteKey)
		assert.Equal(t, "application/json", store.lastWriteMIMEType)

		// Verify JSON structure
		var stored profile.UserProfile
		err = json.Unmarshal(store.lastWriteData, &stored)
		require.NoError(t, err)
		assert.Equal(t, "Alice", stored.DisplayName)
		assert.Equal(t, "https://example.com/pic.jpg", stored.PictureURL)
		assert.Equal(t, "Hello!", stored.StatusMessage)
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		p := &profile.UserProfile{DisplayName: "Alice"}
		err := svc.SetUserProfile(t.Context(), "user-123", p)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write profile")
	})

	t.Run("updates cache even when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := profile.NewService(store, slog.New(slog.DiscardHandler))

		p := &profile.UserProfile{DisplayName: "Alice"}
		_ = svc.SetUserProfile(t.Context(), "user-123", p)

		// Cache should still be populated
		store.writeErr = nil
		store.readErr = errors.New("should not read from storage")
		got, err := svc.GetUserProfile(t.Context(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "Alice", got.DisplayName)
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockStorage struct {
	data              map[string][]byte
	readErr           error
	writeErr          error
	readCallCount     int
	writeCallCount    int
	lastWriteKey      string
	lastWriteMIMEType string
	lastWriteData     []byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
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
	return "", nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
