package history_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/history"
	"yuruppu/internal/message"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Message Type Tests
// =============================================================================

// TestMessage_JSONMarshaling tests that Message can be marshaled to JSON.
func TestMessage_JSONMarshaling(t *testing.T) {
	t.Run("marshal message to JSON", func(t *testing.T) {
		// Given: A Message struct
		msg := message.Message{
			Role:      "user",
			Content:   "Hello, Yuruppu!",
			Timestamp: time.Date(2025, 12, 28, 12, 0, 0, 0, time.UTC),
		}

		// When: Marshal to JSON
		data, err := json.Marshal(msg)

		// Then: Should succeed and contain expected fields
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), `"Role":"user"`)
		assert.Contains(t, string(data), `"Content":"Hello, Yuruppu!"`)
	})

	t.Run("marshal assistant message", func(t *testing.T) {
		// Given: An assistant message
		msg := message.Message{
			Role:      "assistant",
			Content:   "こんにちは！",
			Timestamp: time.Now(),
		}

		// When: Marshal to JSON
		data, err := json.Marshal(msg)

		// Then: Should succeed
		require.NoError(t, err)
		assert.Contains(t, string(data), `"Role":"assistant"`)
		assert.Contains(t, string(data), "こんにちは")
	})

	t.Run("marshal empty content message", func(t *testing.T) {
		// Given: A message with empty content
		msg := message.Message{
			Role:      "user",
			Content:   "",
			Timestamp: time.Now(),
		}

		// When: Marshal to JSON
		data, err := json.Marshal(msg)

		// Then: Should succeed
		require.NoError(t, err)
		assert.Contains(t, string(data), `"Content":""`)
	})
}

// TestMessage_JSONUnmarshaling tests that Message can be unmarshaled from JSON.
func TestMessage_JSONUnmarshaling(t *testing.T) {
	t.Run("unmarshal JSON to message", func(t *testing.T) {
		// Given: JSON data
		jsonData := `{"Role":"user","Content":"Test message","Timestamp":"2025-12-28T12:00:00Z"}`

		// When: Unmarshal to Message
		var msg message.Message
		err := json.Unmarshal([]byte(jsonData), &msg)

		// Then: Should succeed with correct values
		require.NoError(t, err)
		assert.Equal(t, "user", msg.Role)
		assert.Equal(t, "Test message", msg.Content)
		assert.Equal(t, 2025, msg.Timestamp.Year())
	})

	t.Run("unmarshal assistant message", func(t *testing.T) {
		// Given: Assistant message JSON
		jsonData := `{"Role":"assistant","Content":"応答メッセージ","Timestamp":"2025-12-28T15:30:00Z"}`

		// When: Unmarshal
		var msg message.Message
		err := json.Unmarshal([]byte(jsonData), &msg)

		// Then: Should succeed
		require.NoError(t, err)
		assert.Equal(t, "assistant", msg.Role)
		assert.Equal(t, "応答メッセージ", msg.Content)
	})

	t.Run("unmarshal invalid JSON returns error", func(t *testing.T) {
		// Given: Invalid JSON
		invalidJSON := `{"Role":"user","Content":`

		// When: Unmarshal
		var msg message.Message
		err := json.Unmarshal([]byte(invalidJSON), &msg)

		// Then: Should return error
		require.Error(t, err)
	})
}

// TestMessage_TimestampFormat tests that timestamp is serialized as RFC3339.
func TestMessage_TimestampFormat(t *testing.T) {
	t.Run("timestamp serialized as RFC3339", func(t *testing.T) {
		// Given: Message with specific timestamp
		timestamp := time.Date(2025, 12, 28, 12, 30, 45, 0, time.UTC)
		msg := message.Message{
			Role:      "user",
			Content:   "Test",
			Timestamp: timestamp,
		}

		// When: Marshal to JSON
		data, err := json.Marshal(msg)

		// Then: Should contain RFC3339 timestamp
		require.NoError(t, err)
		assert.Contains(t, string(data), "2025-12-28T12:30:45Z")
	})

	t.Run("timestamp with timezone", func(t *testing.T) {
		// Given: Timestamp with JST timezone
		jst := time.FixedZone("JST", 9*60*60)
		timestamp := time.Date(2025, 12, 28, 21, 30, 0, 0, jst)
		msg := message.Message{
			Role:      "user",
			Content:   "Test",
			Timestamp: timestamp,
		}

		// When: Marshal
		data, err := json.Marshal(msg)

		// Then: Should include timezone info
		require.NoError(t, err)
		assert.Contains(t, string(data), "2025-12-28T21:30:00+09:00")
	})

	t.Run("round-trip preserves timestamp", func(t *testing.T) {
		// Given: Original message
		original := message.Message{
			Role:      "user",
			Content:   "Test",
			Timestamp: time.Date(2025, 12, 28, 12, 0, 0, 0, time.UTC),
		}

		// When: Marshal and unmarshal
		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored message.Message
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Then: Timestamp should be preserved (within precision)
		assert.Equal(t, original.Timestamp.Unix(), restored.Timestamp.Unix())
	})
}

// =============================================================================
// JSONL Serialization Tests
// =============================================================================

// TestJSONL_EncodeMessages tests encoding multiple messages to JSONL format.
func TestJSONL_EncodeMessages(t *testing.T) {
	t.Run("encode multiple messages to JSONL", func(t *testing.T) {
		// Given: Multiple messages
		messages := []message.Message{
			{
				Role:      "user",
				Content:   "First message",
				Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC),
			},
			{
				Role:      "assistant",
				Content:   "First response",
				Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC),
			},
			{
				Role:      "user",
				Content:   "Second message",
				Timestamp: time.Date(2025, 12, 28, 10, 2, 0, 0, time.UTC),
			},
		}

		// When: Encode to JSONL
		var builder strings.Builder
		for _, msg := range messages {
			data, err := json.Marshal(msg)
			require.NoError(t, err)
			builder.Write(data)
			builder.WriteString("\n")
		}
		jsonl := builder.String()

		// Then: Should have one line per message
		lines := strings.Split(strings.TrimSpace(jsonl), "\n")
		assert.Len(t, lines, 3)

		// Each line should be valid JSON
		for _, line := range lines {
			var msg message.Message
			err := json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err, "Each line should be valid JSON")
		}
	})

	t.Run("encode empty message slice to JSONL", func(t *testing.T) {
		// Given: Empty message slice
		messages := []message.Message{}

		// When: Encode to JSONL
		var builder strings.Builder
		for _, msg := range messages {
			data, err := json.Marshal(msg)
			require.NoError(t, err)
			builder.Write(data)
			builder.WriteString("\n")
		}

		// Then: Should be empty
		assert.Empty(t, builder.String())
	})

	t.Run("encode single message to JSONL", func(t *testing.T) {
		// Given: Single message
		messages := []message.Message{
			{
				Role:      "user",
				Content:   "Only message",
				Timestamp: time.Now(),
			},
		}

		// When: Encode
		var builder strings.Builder
		for _, msg := range messages {
			data, err := json.Marshal(msg)
			require.NoError(t, err)
			builder.Write(data)
			builder.WriteString("\n")
		}
		jsonl := builder.String()

		// Then: Should have one line
		lines := strings.Split(strings.TrimSpace(jsonl), "\n")
		assert.Len(t, lines, 1)
	})
}

// TestJSONL_DecodeMessages tests decoding JSONL to []Message.
func TestJSONL_DecodeMessages(t *testing.T) {
	t.Run("decode JSONL to messages", func(t *testing.T) {
		// Given: JSONL data
		jsonl := `{"Role":"user","Content":"Message 1","Timestamp":"2025-12-28T10:00:00Z"}
{"Role":"assistant","Content":"Response 1","Timestamp":"2025-12-28T10:01:00Z"}
{"Role":"user","Content":"Message 2","Timestamp":"2025-12-28T10:02:00Z"}`

		// When: Decode JSONL
		lines := strings.Split(strings.TrimSpace(jsonl), "\n")
		var messages []message.Message
		for _, line := range lines {
			var msg message.Message
			err := json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err)
			messages = append(messages, msg)
		}

		// Then: Should decode all messages
		assert.Len(t, messages, 3)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Message 1", messages[0].Content)
		assert.Equal(t, "assistant", messages[1].Role)
		assert.Equal(t, "Response 1", messages[1].Content)
	})

	t.Run("decode empty JSONL returns empty slice", func(t *testing.T) {
		// Given: Empty JSONL
		jsonl := ""

		// When: Decode
		lines := strings.Split(strings.TrimSpace(jsonl), "\n")
		var messages []message.Message
		for _, line := range lines {
			if line == "" {
				continue
			}
			var msg message.Message
			err := json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err)
			messages = append(messages, msg)
		}

		// Then: Should return empty slice
		assert.Empty(t, messages)
	})

	t.Run("decode JSONL with blank lines", func(t *testing.T) {
		// Given: JSONL with blank lines
		jsonl := `{"Role":"user","Content":"Message 1","Timestamp":"2025-12-28T10:00:00Z"}

{"Role":"assistant","Content":"Response 1","Timestamp":"2025-12-28T10:01:00Z"}`

		// When: Decode (skipping blank lines)
		lines := strings.Split(jsonl, "\n")
		var messages []message.Message
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var msg message.Message
			err := json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err)
			messages = append(messages, msg)
		}

		// Then: Should skip blank lines
		assert.Len(t, messages, 2)
	})
}

// TestJSONL_DecodeInvalidJSON tests that invalid JSON lines return errors.
func TestJSONL_DecodeInvalidJSON(t *testing.T) {
	t.Run("decode invalid JSON line returns error", func(t *testing.T) {
		// Given: Invalid JSON line
		invalidLine := `{"Role":"user","Content":`

		// When: Decode
		var msg message.Message
		err := json.Unmarshal([]byte(invalidLine), &msg)

		// Then: Should return error
		require.Error(t, err)
	})

	t.Run("decode JSONL with one invalid line", func(t *testing.T) {
		// Given: JSONL with one invalid line
		jsonl := `{"Role":"user","Content":"Valid","Timestamp":"2025-12-28T10:00:00Z"}
{"Role":"user","Content":
{"Role":"assistant","Content":"Also valid","Timestamp":"2025-12-28T10:02:00Z"}`

		// When: Decode
		lines := strings.Split(strings.TrimSpace(jsonl), "\n")
		var messages []message.Message
		var decodeErr error
		for _, line := range lines {
			var msg message.Message
			err := json.Unmarshal([]byte(line), &msg)
			if err != nil {
				decodeErr = err
				break
			}
			messages = append(messages, msg)
		}

		// Then: Should encounter error on invalid line
		assert.Error(t, decodeErr)
		assert.Len(t, messages, 1) // Only first valid message decoded
	})
}

// =============================================================================
// Repository Tests
// =============================================================================

// TestNewRepository_NilStorage tests that nil storage returns ErrNilStorage.
func TestNewRepository_NilStorage(t *testing.T) {
	t.Run("nil storage returns ErrNilStorage", func(t *testing.T) {
		repo, err := history.NewRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.ErrorIs(t, err, history.ErrNilStorage)
	})
}

// TestRepository_EmptySourceID tests that empty sourceID returns ValidationError.
func TestRepository_EmptySourceID(t *testing.T) {
	t.Run("GetHistory with empty sourceID returns ValidationError", func(t *testing.T) {
		repo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)

		_, _, err = repo.GetHistory(t.Context(), "")

		require.Error(t, err)
		var validationErr *history.ValidationError
		assert.True(t, errors.As(err, &validationErr), "error should be ValidationError")
	})

	t.Run("GetHistory with whitespace-only sourceID returns ValidationError", func(t *testing.T) {
		repo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)

		_, _, err = repo.GetHistory(t.Context(), "   ")

		require.Error(t, err)
		var validationErr *history.ValidationError
		assert.True(t, errors.As(err, &validationErr), "error should be ValidationError")
	})

	t.Run("PutHistory with empty sourceID returns ValidationError", func(t *testing.T) {
		repo, err := history.NewRepository(&mockStorage{})
		require.NoError(t, err)

		err = repo.PutHistory(t.Context(), "", []message.Message{}, 0)

		require.Error(t, err)
		var validationErr *history.ValidationError
		assert.True(t, errors.As(err, &validationErr), "error should be ValidationError")
	})
}

// =============================================================================
// Mock Storage for Repository Tests
// =============================================================================

type mockStorage struct{}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	return nil, 0, nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) error {
	return nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}
