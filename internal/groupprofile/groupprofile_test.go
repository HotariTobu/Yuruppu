package groupprofile_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/groupprofile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// NewService Tests
// =============================================================================

func TestNewService(t *testing.T) {
	t.Run("returns service with valid inputs", func(t *testing.T) {
		store := newMockStorage()
		logger := slog.New(slog.DiscardHandler)

		svc, err := groupprofile.NewService(store, logger)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("returns error when storage is nil", func(t *testing.T) {
		logger := slog.New(slog.DiscardHandler)

		svc, err := groupprofile.NewService(nil, logger)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "storage")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		store := newMockStorage()

		svc, err := groupprofile.NewService(store, nil)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "logger")
	})
}

// =============================================================================
// GetGroupProfile Tests
// =============================================================================

func TestService_GetGroupProfile(t *testing.T) {
	t.Run("returns profile from cache when available", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		// Set profile first (populates cache)
		p := &groupprofile.GroupProfile{DisplayName: "Group A"}
		err := svc.SetGroupProfile(t.Context(), "group-123", p)
		require.NoError(t, err)

		// Reset read count to verify cache is used
		store.readCallCount = 0

		// Get profile (should hit cache)
		got, err := svc.GetGroupProfile(t.Context(), "group-123")

		require.NoError(t, err)
		assert.Equal(t, "Group A", got.DisplayName)
		assert.Equal(t, 0, store.readCallCount, "should not read from storage when cached")
	})

	t.Run("returns profile from storage when not in cache", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		// Pre-populate storage directly (bypassing cache)
		p := &groupprofile.GroupProfile{DisplayName: "Group B", PictureURL: "https://example.com/group.jpg"}
		data, _ := json.Marshal(p)
		store.data["group-456"] = data

		got, err := svc.GetGroupProfile(t.Context(), "group-456")

		require.NoError(t, err)
		assert.Equal(t, "Group B", got.DisplayName)
		assert.Equal(t, "https://example.com/group.jpg", got.PictureURL)
		assert.Equal(t, 1, store.readCallCount)
	})

	t.Run("caches profile after reading from storage", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		// Pre-populate storage
		p := &groupprofile.GroupProfile{DisplayName: "Group C"}
		data, _ := json.Marshal(p)
		store.data["group-789"] = data

		// First read (from storage)
		_, err := svc.GetGroupProfile(t.Context(), "group-789")
		require.NoError(t, err)
		assert.Equal(t, 1, store.readCallCount)

		// Second read (from cache)
		_, err = svc.GetGroupProfile(t.Context(), "group-789")
		require.NoError(t, err)
		assert.Equal(t, 1, store.readCallCount, "should use cache on second read")
	})

	t.Run("returns error when storage read fails", func(t *testing.T) {
		store := newMockStorage()
		store.readErr = errors.New("storage error")
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		got, err := svc.GetGroupProfile(t.Context(), "group-123")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to read group profile")
	})

	t.Run("returns error when JSON unmarshal fails", func(t *testing.T) {
		store := newMockStorage()
		store.data["group-123"] = []byte("invalid json")
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		got, err := svc.GetGroupProfile(t.Context(), "group-123")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to unmarshal group profile")
	})

	t.Run("returns error when profile not found", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		got, err := svc.GetGroupProfile(t.Context(), "nonexistent")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "group profile not found")
	})
}

// =============================================================================
// SetGroupProfile Tests
// =============================================================================

func TestService_SetGroupProfile(t *testing.T) {
	t.Run("stores profile to cache and storage", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		p := &groupprofile.GroupProfile{
			DisplayName: "Group A",
			PictureURL:  "https://example.com/group.jpg",
		}
		err := svc.SetGroupProfile(t.Context(), "group-123", p)

		require.NoError(t, err)
		assert.Equal(t, 1, store.writeCallCount)
		assert.Equal(t, "group-123", store.lastWriteKey)
		assert.Equal(t, "application/json", store.lastWriteMIMEType)

		// Verify JSON structure
		var stored groupprofile.GroupProfile
		err = json.Unmarshal(store.lastWriteData, &stored)
		require.NoError(t, err)
		assert.Equal(t, "Group A", stored.DisplayName)
		assert.Equal(t, "https://example.com/group.jpg", stored.PictureURL)
	})

	t.Run("returns error when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		p := &groupprofile.GroupProfile{DisplayName: "Group A"}
		err := svc.SetGroupProfile(t.Context(), "group-123", p)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write group profile")
	})

	t.Run("does not update cache when storage write fails", func(t *testing.T) {
		store := newMockStorage()
		store.writeErr = errors.New("write failed")
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		p := &groupprofile.GroupProfile{DisplayName: "Group A"}
		_ = svc.SetGroupProfile(t.Context(), "group-123", p)

		// Cache should NOT be populated after write failure
		store.writeErr = nil
		got, err := svc.GetGroupProfile(t.Context(), "group-123")

		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "group profile not found")
	})

	t.Run("omits empty picture URL in JSON", func(t *testing.T) {
		store := newMockStorage()
		svc, _ := groupprofile.NewService(store, slog.New(slog.DiscardHandler))

		p := &groupprofile.GroupProfile{
			DisplayName: "Group A",
			PictureURL:  "", // empty
		}
		err := svc.SetGroupProfile(t.Context(), "group-123", p)

		require.NoError(t, err)
		// Verify omitempty works
		assert.NotContains(t, string(store.lastWriteData), "pictureUrl")
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
