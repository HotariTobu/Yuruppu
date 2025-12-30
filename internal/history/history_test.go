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
// Repository Tests
// =============================================================================

// TestNewRepository_NilStorage tests that nil storage returns an error.
func TestNewRepository_NilStorage(t *testing.T) {
	t.Run("nil storage returns error", func(t *testing.T) {
		repo, err := history.NewRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "storage cannot be nil")
	})
}

// TestRepository_EmptySourceID tests that empty sourceID returns an error.
func TestRepository_EmptySourceID(t *testing.T) {
	t.Run("GetHistory with empty sourceID returns error", func(t *testing.T) {
		repo, err := history.NewRepository(newMockStorage())
		require.NoError(t, err)

		_, _, err = repo.GetHistory(t.Context(), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceID cannot be empty")
	})

	t.Run("GetHistory with whitespace-only sourceID returns error", func(t *testing.T) {
		repo, err := history.NewRepository(newMockStorage())
		require.NoError(t, err)

		_, _, err = repo.GetHistory(t.Context(), "   ")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceID cannot be empty")
	})

	t.Run("PutHistory with empty sourceID returns error", func(t *testing.T) {
		repo, err := history.NewRepository(newMockStorage())
		require.NoError(t, err)

		err = repo.PutHistory(t.Context(), "", []history.Message{}, 0)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceID cannot be empty")
	})
}

// TestRepository_RoundTrip tests PutHistory and GetHistory round-trip.
func TestRepository_RoundTrip(t *testing.T) {
	t.Run("round-trip with text message", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// Given: A user message with text
		timestamp := time.Date(2025, 12, 28, 12, 0, 0, 0, time.UTC)
		messages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: timestamp,
			},
		}

		// When: Put and Get
		err = repo.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := repo.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: Should match
		require.Len(t, retrieved, 1)
		userMsg, ok := retrieved[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "U123", userMsg.UserID)
		require.Len(t, userMsg.Parts, 1)
		textPart, ok := userMsg.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Hello", textPart.Text)
	})

	t.Run("round-trip with file data", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// Given: A user message with file data
		messages := []history.Message{
			&history.UserMessage{
				UserID: "U123",
				Parts: []history.UserPart{
					&history.UserFileDataPart{
						StorageKey:  "files/image.png",
						MIMEType:    "image/png",
						DisplayName: "photo.png",
					},
				},
				Timestamp: time.Now(),
			},
		}

		// When: Put and Get
		err = repo.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := repo.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: Should match
		require.Len(t, retrieved, 1)
		userMsg, ok := retrieved[0].(*history.UserMessage)
		require.True(t, ok)
		require.Len(t, userMsg.Parts, 1)
		filePart, ok := userMsg.Parts[0].(*history.UserFileDataPart)
		require.True(t, ok)
		assert.Equal(t, "files/image.png", filePart.StorageKey)
		assert.Equal(t, "image/png", filePart.MIMEType)
		assert.Equal(t, "photo.png", filePart.DisplayName)
	})

	t.Run("round-trip with multiple messages", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// Given: Multiple messages (user and assistant)
		messages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC),
			},
			&history.AssistantMessage{
				ModelName: "gemini-2.0",
				Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: "Hi there!"}},
				Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC),
			},
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "How are you?"}},
				Timestamp: time.Date(2025, 12, 28, 10, 2, 0, 0, time.UTC),
			},
		}

		// When: Put and Get
		err = repo.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := repo.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: Should have 3 messages in order
		require.Len(t, retrieved, 3)

		// First message: user
		userMsg1, ok := retrieved[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "U123", userMsg1.UserID)

		// Second message: assistant
		assistantMsg, ok := retrieved[1].(*history.AssistantMessage)
		require.True(t, ok)
		assert.Equal(t, "gemini-2.0", assistantMsg.ModelName)

		// Third message: user
		userMsg2, ok := retrieved[2].(*history.UserMessage)
		require.True(t, ok)
		textPart, ok := userMsg2.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "How are you?", textPart.Text)
	})

	t.Run("round-trip with assistant thought", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// Given: An assistant message with thought
		messages := []history.Message{
			&history.AssistantMessage{
				ModelName: "gemini-2.0",
				Parts: []history.AssistantPart{
					&history.AssistantTextPart{
						Text:    "thinking...",
						Thought: true,
					},
					&history.AssistantTextPart{
						Text: "Here's my answer",
					},
				},
				Timestamp: time.Now(),
			},
		}

		// When: Put and Get
		err = repo.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := repo.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: Should preserve thought flag
		require.Len(t, retrieved, 1)
		assistantMsg, ok := retrieved[0].(*history.AssistantMessage)
		require.True(t, ok)
		require.Len(t, assistantMsg.Parts, 2)

		thoughtPart, ok := assistantMsg.Parts[0].(*history.AssistantTextPart)
		require.True(t, ok)
		assert.True(t, thoughtPart.Thought)
		assert.Equal(t, "thinking...", thoughtPart.Text)

		answerPart, ok := assistantMsg.Parts[1].(*history.AssistantTextPart)
		require.True(t, ok)
		assert.False(t, answerPart.Thought)
		assert.Equal(t, "Here's my answer", answerPart.Text)
	})
}

// TestRepository_KeyIsolation tests that different keys store different data.
func TestRepository_KeyIsolation(t *testing.T) {
	t.Run("different keys store different data", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// Given: Two different sources with different messages
		messages1 := []history.Message{
			&history.UserMessage{
				UserID:    "U111",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message for source1"}},
				Timestamp: time.Now(),
			},
		}
		messages2 := []history.Message{
			&history.UserMessage{
				UserID:    "U222",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message for source2"}},
				Timestamp: time.Now(),
			},
		}

		// When: Put to different keys
		err = repo.PutHistory(t.Context(), "source1", messages1, 0)
		require.NoError(t, err)
		err = repo.PutHistory(t.Context(), "source2", messages2, 0)
		require.NoError(t, err)

		// Then: Each key returns its own data
		retrieved1, _, err := repo.GetHistory(t.Context(), "source1")
		require.NoError(t, err)
		require.Len(t, retrieved1, 1)
		userMsg1, ok := retrieved1[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "U111", userMsg1.UserID)
		textPart1, ok := userMsg1.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Message for source1", textPart1.Text)

		retrieved2, _, err := repo.GetHistory(t.Context(), "source2")
		require.NoError(t, err)
		require.Len(t, retrieved2, 1)
		userMsg2, ok := retrieved2[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "U222", userMsg2.UserID)
		textPart2, ok := userMsg2.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Message for source2", textPart2.Text)
	})

	t.Run("non-existent key returns empty history", func(t *testing.T) {
		storage := newMockStorage()
		repo, err := history.NewRepository(storage)
		require.NoError(t, err)

		// When: Get history for non-existent key
		retrieved, generation, err := repo.GetHistory(t.Context(), "non-existent")

		// Then: Should return empty slice, not error
		require.NoError(t, err)
		assert.Empty(t, retrieved)
		assert.Equal(t, int64(0), generation)
	})
}

// =============================================================================
// Mock Storage
// =============================================================================

type mockStorage struct {
	data       map[string][]byte
	generation map[string]int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: make(map[string]int64),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	data, exists := m.data[key]
	if !exists {
		return nil, 0, nil
	}
	return data, m.generation[key], nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) error {
	m.data[key] = data
	m.generation[key] = expectedGeneration + 1
	return nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "", nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
