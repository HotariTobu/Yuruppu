package history_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixed timestamps for deterministic tests
var (
	testTime1 = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	testTime2 = time.Date(2025, 1, 1, 10, 1, 0, 0, time.UTC)
	testTime3 = time.Date(2025, 1, 1, 10, 2, 0, 0, time.UTC)
)

// =============================================================================
// Service Tests
// =============================================================================

// TestNewService_NilStorage tests that nil storage returns an error.
func TestNewService_NilStorage(t *testing.T) {
	t.Run("nil storage returns error", func(t *testing.T) {
		svc, err := history.NewService(nil)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "storage cannot be nil")
	})
}

// TestService_SourceIDValidation tests sourceID validation for both Get and Put operations.
func TestService_SourceIDValidation(t *testing.T) {
	tests := []struct {
		name       string
		sourceID   string
		wantErrMsg string
	}{
		{"empty", "", "sourceID cannot be empty"},
		{"whitespace only", "   ", "sourceID cannot be empty"},
		{"contains slash", "path/to/file", "invalid characters"},
		{"contains double dots", "parent..child", "invalid characters"},
		{"starts with double dots", "..escape", "invalid characters"},
	}

	for _, tt := range tests {
		t.Run("GetHistory_"+tt.name, func(t *testing.T) {
			svc, err := history.NewService(newMockStorage())
			require.NoError(t, err)

			_, _, err = svc.GetHistory(t.Context(), tt.sourceID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})

		t.Run("PutHistory_"+tt.name, func(t *testing.T) {
			svc, err := history.NewService(newMockStorage())
			require.NoError(t, err)

			_, err = svc.PutHistory(t.Context(), tt.sourceID, []history.Message{}, 0)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

// TestService_RoundTrip tests PutHistory and GetHistory round-trip.
func TestService_RoundTrip(t *testing.T) {
	t.Run("round-trip with text message", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: A user message with text
		messages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: testTime1,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
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
		svc, err := history.NewService(storage)
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
				Timestamp: testTime1,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
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
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: Multiple messages (user and assistant)
		messages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: testTime1,
			},
			&history.AssistantMessage{
				ModelName: "gemini-2.0",
				Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: "Hi there!"}},
				Timestamp: testTime2,
			},
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "How are you?"}},
				Timestamp: testTime3,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
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
		svc, err := history.NewService(storage)
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
				Timestamp: testTime1,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
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

// TestService_KeyIsolation tests that different keys store different data.
func TestService_KeyIsolation(t *testing.T) {
	t.Run("different keys store different data", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: Two different sources with different messages
		messages1 := []history.Message{
			&history.UserMessage{
				UserID:    "U111",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message for source1"}},
				Timestamp: testTime1,
			},
		}
		messages2 := []history.Message{
			&history.UserMessage{
				UserID:    "U222",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Message for source2"}},
				Timestamp: testTime1,
			},
		}

		// When: Put to different keys
		_, err = svc.PutHistory(t.Context(), "source1", messages1, 0)
		require.NoError(t, err)
		_, err = svc.PutHistory(t.Context(), "source2", messages2, 0)
		require.NoError(t, err)

		// Then: Each key returns its own data
		retrieved1, _, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)
		require.Len(t, retrieved1, 1)
		userMsg1, ok := retrieved1[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "U111", userMsg1.UserID)
		textPart1, ok := userMsg1.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Message for source1", textPart1.Text)

		retrieved2, _, err := svc.GetHistory(t.Context(), "source2")
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
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// When: Get history for non-existent key
		retrieved, generation, err := svc.GetHistory(t.Context(), "non-existent")

		// Then: Should return empty slice, not error
		require.NoError(t, err)
		assert.Empty(t, retrieved)
		assert.Equal(t, int64(0), generation)
	})
}

// =============================================================================
// Optimistic Locking Tests
// =============================================================================

// TestService_OptimisticLocking tests generation-based concurrent modification detection.
func TestService_OptimisticLocking(t *testing.T) {
	t.Run("write with stale generation fails", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		messages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: testTime1,
			},
		}

		// First write succeeds (generation 0 -> 1)
		gen1, err := svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), gen1)

		// Second write with stale generation (0) fails
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generation mismatch")
	})

	t.Run("write with correct generation succeeds", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		messages1 := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "First"}},
				Timestamp: testTime1,
			},
		}
		messages2 := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Second"}},
				Timestamp: testTime1,
			},
		}

		// First write (generation 0 -> 1)
		gen1, err := svc.PutHistory(t.Context(), "source1", messages1, 0)
		require.NoError(t, err)

		// Second write with correct generation succeeds (generation 1 -> 2)
		gen2, err := svc.PutHistory(t.Context(), "source1", messages2, gen1)
		require.NoError(t, err)
		assert.Equal(t, int64(2), gen2)

		// Verify the latest data
		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)
		require.Len(t, retrieved, 1)
		userMsg := retrieved[0].(*history.UserMessage)
		textPart := userMsg.Parts[0].(*history.UserTextPart)
		assert.Equal(t, "Second", textPart.Text)
	})

	t.Run("concurrent modification detected", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Initial state
		initialMessages := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Initial"}},
				Timestamp: testTime1,
			},
		}
		gen, err := svc.PutHistory(t.Context(), "source1", initialMessages, 0)
		require.NoError(t, err)

		// Simulate two concurrent reads (both get same generation)
		_, genA, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)
		_, genB, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)
		assert.Equal(t, gen, genA)
		assert.Equal(t, gen, genB)

		// First concurrent write succeeds
		messagesA := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Writer A"}},
				Timestamp: testTime1,
			},
		}
		_, err = svc.PutHistory(t.Context(), "source1", messagesA, genA)
		require.NoError(t, err)

		// Second concurrent write fails (stale generation)
		messagesB := []history.Message{
			&history.UserMessage{
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Writer B"}},
				Timestamp: testTime1,
			},
		}
		_, err = svc.PutHistory(t.Context(), "source1", messagesB, genB)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "generation mismatch")
	})
}

// =============================================================================
// JSONL Parsing Error Tests
// =============================================================================

// TestService_ParseErrors tests error handling for malformed JSONL data.
func TestService_ParseErrors(t *testing.T) {
	t.Run("invalid JSON returns error", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte(`{invalid json}`)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		_, _, err = svc.GetHistory(t.Context(), "source1")

		require.Error(t, err)
	})

	t.Run("unknown role returns error", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte(`{"role":"unknown","parts":[],"timestamp":"2025-01-01T00:00:00Z"}`)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		_, _, err = svc.GetHistory(t.Context(), "source1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown role")
	})

	t.Run("unknown user part type returns error", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte(`{"role":"user","userId":"U123","parts":[{"type":"unknown"}],"timestamp":"2025-01-01T00:00:00Z"}`)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		_, _, err = svc.GetHistory(t.Context(), "source1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown user part type")
	})

	t.Run("unknown assistant part type returns error", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte(`{"role":"assistant","modelName":"test","parts":[{"type":"unknown"}],"timestamp":"2025-01-01T00:00:00Z"}`)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		_, _, err = svc.GetHistory(t.Context(), "source1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown assistant part type")
	})

	t.Run("empty lines are skipped", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte("\n\n\n")
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		messages, _, err := svc.GetHistory(t.Context(), "source1")

		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("whitespace-only lines are skipped", func(t *testing.T) {
		storage := newMockStorage()
		storage.data["source1"] = []byte("   \n\t\n  \t  \n")
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		messages, _, err := svc.GetHistory(t.Context(), "source1")

		require.NoError(t, err)
		assert.Empty(t, messages)
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

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	currentGen := m.generation[key]
	if currentGen != expectedGeneration {
		return 0, fmt.Errorf("generation mismatch: expected %d, got %d", expectedGeneration, currentGen)
	}
	m.data[key] = data
	newGen := expectedGeneration + 1
	m.generation[key] = newGen
	return newGen, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "", nil
}
