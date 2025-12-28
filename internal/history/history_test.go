package history_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/history"

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
		msg := history.Message{
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
		msg := history.Message{
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
		msg := history.Message{
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
		var msg history.Message
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
		var msg history.Message
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
		var msg history.Message
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
		msg := history.Message{
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
		msg := history.Message{
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
		original := history.Message{
			Role:      "user",
			Content:   "Test",
			Timestamp: time.Date(2025, 12, 28, 12, 0, 0, 0, time.UTC),
		}

		// When: Marshal and unmarshal
		data, err := json.Marshal(original)
		require.NoError(t, err)

		var restored history.Message
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
		messages := []history.Message{
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
			var msg history.Message
			err := json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err, "Each line should be valid JSON")
		}
	})

	t.Run("encode empty message slice to JSONL", func(t *testing.T) {
		// Given: Empty message slice
		messages := []history.Message{}

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
		messages := []history.Message{
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
		var messages []history.Message
		for _, line := range lines {
			var msg history.Message
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
		var messages []history.Message
		for _, line := range lines {
			if line == "" {
				continue
			}
			var msg history.Message
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
		var messages []history.Message
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var msg history.Message
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
		var msg history.Message
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
		var messages []history.Message
		var decodeErr error
		for _, line := range lines {
			var msg history.Message
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
// ConversationHistory Tests
// =============================================================================

// TestConversationHistory_Struct tests ConversationHistory struct.
func TestConversationHistory_Struct(t *testing.T) {
	t.Run("ConversationHistory holds SourceID and Messages", func(t *testing.T) {
		// Given: ConversationHistory struct
		hist := history.ConversationHistory{
			SourceID: "U123abc",
			Messages: []history.Message{
				{
					Role:      "user",
					Content:   "Hello",
					Timestamp: time.Now(),
				},
				{
					Role:      "assistant",
					Content:   "Hi there",
					Timestamp: time.Now(),
				},
			},
		}

		// Then: Should hold SourceID and Messages
		assert.Equal(t, "U123abc", hist.SourceID)
		assert.Len(t, hist.Messages, 2)
		assert.Equal(t, "user", hist.Messages[0].Role)
		assert.Equal(t, "assistant", hist.Messages[1].Role)
	})

	t.Run("ConversationHistory with empty messages", func(t *testing.T) {
		// Given: ConversationHistory with no messages
		hist := history.ConversationHistory{
			SourceID: "C789ghi",
			Messages: []history.Message{},
		}

		// Then: Should have empty messages slice
		assert.Equal(t, "C789ghi", hist.SourceID)
		assert.Empty(t, hist.Messages)
	})

	t.Run("ConversationHistory for group chat", func(t *testing.T) {
		// Given: Group chat conversation
		hist := history.ConversationHistory{
			SourceID: "C123group",
			Messages: []history.Message{
				{Role: "user", Content: "Group message 1", Timestamp: time.Now()},
				{Role: "assistant", Content: "Response 1", Timestamp: time.Now()},
				{Role: "user", Content: "Group message 2", Timestamp: time.Now()},
			},
		}

		// Then: Should hold group SourceID
		assert.Equal(t, "C123group", hist.SourceID)
		assert.Len(t, hist.Messages, 3)
	})

	t.Run("different SourceIDs maintain separate histories", func(t *testing.T) {
		// Given: Two different conversation histories
		hist1 := history.ConversationHistory{
			SourceID: "U111",
			Messages: []history.Message{
				{Role: "user", Content: "User 1 message", Timestamp: time.Now()},
			},
		}

		hist2 := history.ConversationHistory{
			SourceID: "U222",
			Messages: []history.Message{
				{Role: "user", Content: "User 2 message", Timestamp: time.Now()},
			},
		}

		// Then: Should be independent
		assert.NotEqual(t, hist1.SourceID, hist2.SourceID)
		assert.NotEqual(t, hist1.Messages[0].Content, hist2.Messages[0].Content)
	})
}

// =============================================================================
// Error Type Tests
// =============================================================================

// TestStorageReadError tests the StorageReadError type.
// Spec: StorageReadError - 履歴の読み込みに失敗
func TestStorageReadError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create StorageReadError
		err := &history.StorageReadError{
			Message: "failed to read history from GCS",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains read error details", func(t *testing.T) {
		// Given: StorageReadError with specific message
		err := &history.StorageReadError{
			Message: "GCS bucket not found",
		}

		// When: Get error string
		errMsg := err.Error()

		// Then: Should contain error details
		assert.Contains(t, errMsg, "GCS bucket not found")
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create StorageReadError
		readErr := &history.StorageReadError{
			Message: "read failed",
		}

		// When: Check with errors.As
		var target *history.StorageReadError
		directErr := error(readErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target))
		assert.Equal(t, "read failed", target.Message)
	})
}

// TestStorageWriteError tests the StorageWriteError type.
// Spec: StorageWriteError - 履歴の保存に失敗
func TestStorageWriteError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create StorageWriteError
		err := &history.StorageWriteError{
			Message: "failed to write history to GCS",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains write error details", func(t *testing.T) {
		// Given: StorageWriteError with specific message
		err := &history.StorageWriteError{
			Message: "precondition failed (412)",
		}

		// When: Get error string
		errMsg := err.Error()

		// Then: Should contain error details
		assert.Contains(t, errMsg, "precondition failed")
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create StorageWriteError
		writeErr := &history.StorageWriteError{
			Message: "write failed",
		}

		// When: Check with errors.As
		var target *history.StorageWriteError
		directErr := error(writeErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target))
		assert.Equal(t, "write failed", target.Message)
	})
}

// TestStorageTimeoutError tests the StorageTimeoutError type.
// Spec: StorageTimeoutError - ストレージ操作がタイムアウト
func TestStorageTimeoutError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		// Given: Create StorageTimeoutError
		err := &history.StorageTimeoutError{
			Message: "GCS operation timed out after 5s",
		}

		// When: Use as error interface
		var _ error = err

		// Then: Should implement error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})

	t.Run("error message contains timeout details", func(t *testing.T) {
		// Given: StorageTimeoutError with timeout duration
		err := &history.StorageTimeoutError{
			Message: "timeout after 10 seconds",
		}

		// When: Get error string
		errMsg := err.Error()

		// Then: Should contain timeout information
		assert.Contains(t, errMsg, "timeout")
		assert.Contains(t, errMsg, "10 seconds")
	})

	t.Run("can be identified with errors.As", func(t *testing.T) {
		// Given: Create StorageTimeoutError
		timeoutErr := &history.StorageTimeoutError{
			Message: "operation timed out",
		}

		// When: Check with errors.As
		var target *history.StorageTimeoutError
		directErr := error(timeoutErr)

		// Then: Should match with errors.As
		assert.True(t, errors.As(directErr, &target))
		assert.Equal(t, "operation timed out", target.Message)
	})
}

// TestErrorTypes_Distinction tests different error types can be distinguished.
func TestErrorTypes_Distinction(t *testing.T) {
	t.Run("can distinguish between different storage error types", func(t *testing.T) {
		// Given: Different storage error types
		readErr := &history.StorageReadError{Message: "read error"}
		writeErr := &history.StorageWriteError{Message: "write error"}
		timeoutErr := &history.StorageTimeoutError{Message: "timeout error"}

		// When: Check each error type
		errs := []error{readErr, writeErr, timeoutErr}

		// Then: Each should be distinguishable
		for i, err1 := range errs {
			for j, err2 := range errs {
				if i == j {
					assert.Equal(t, err1, err2)
				} else {
					assert.NotEqual(t, err1, err2)
				}
			}
		}
	})

	t.Run("can use errors.As to identify error types", func(t *testing.T) {
		t.Run("read error matches StorageReadError", func(t *testing.T) {
			err := &history.StorageReadError{Message: "read"}
			var target *history.StorageReadError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("read error does not match StorageWriteError", func(t *testing.T) {
			err := &history.StorageReadError{Message: "read"}
			var target *history.StorageWriteError
			assert.False(t, errors.As(err, &target))
		})

		t.Run("write error matches StorageWriteError", func(t *testing.T) {
			err := &history.StorageWriteError{Message: "write"}
			var target *history.StorageWriteError
			assert.True(t, errors.As(err, &target))
		})

		t.Run("timeout error matches StorageTimeoutError", func(t *testing.T) {
			err := &history.StorageTimeoutError{Message: "timeout"}
			var target *history.StorageTimeoutError
			assert.True(t, errors.As(err, &target))
		})
	})
}

// =============================================================================
// Storage Interface Tests
// =============================================================================

// TestStorage_Interface tests that Storage interface exists and can be implemented.
// Based on ADR: 20251228-chat-history-storage
func TestStorage_Interface(t *testing.T) {
	t.Run("interface can be implemented by mock", func(t *testing.T) {
		// Given: A mock implementation of Storage interface
		var storage history.Storage = &mockStorage{
			messages: []history.Message{
				{Role: "user", Content: "Test", Timestamp: time.Now()},
			},
		}

		// When: Call GetHistory
		ctx := context.Background()
		messages, err := storage.GetHistory(ctx, "U123")

		// Then: Should work as expected
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
	})
}

// TestStorage_GetHistory tests the GetHistory method signature.
// Spec: GetHistory retrieves conversation history for a source
func TestStorage_GetHistory(t *testing.T) {
	t.Run("GetHistory returns messages for sourceID", func(t *testing.T) {
		// Given: Mock storage with history
		mock := &mockStorage{
			messages: []history.Message{
				{Role: "user", Content: "Message 1", Timestamp: time.Now()},
				{Role: "assistant", Content: "Response 1", Timestamp: time.Now()},
			},
		}

		// When: Get history
		ctx := context.Background()
		messages, err := mock.GetHistory(ctx, "U123")

		// Then: Should return messages
		require.NoError(t, err)
		assert.Len(t, messages, 2)
	})

	t.Run("GetHistory returns empty slice when no history exists", func(t *testing.T) {
		// Given: Mock storage with no history
		mock := &mockStorage{
			messages: []history.Message{},
		}

		// When: Get history
		ctx := context.Background()
		messages, err := mock.GetHistory(ctx, "U999")

		// Then: Should return empty slice
		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("GetHistory returns error on storage failure", func(t *testing.T) {
		// Given: Mock storage that returns error
		mock := &mockStorage{
			getHistoryErr: &history.StorageReadError{Message: "GCS error"},
		}

		// When: Get history
		ctx := context.Background()
		messages, err := mock.GetHistory(ctx, "U123")

		// Then: Should return error
		require.Error(t, err)
		assert.Nil(t, messages)
		var readErr *history.StorageReadError
		assert.ErrorAs(t, err, &readErr)
	})

	t.Run("GetHistory respects context cancellation", func(t *testing.T) {
		// Given: Mock storage that checks context
		mock := &mockStorage{
			checkContext: true,
		}

		// Given: Cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// When: Get history with cancelled context
		messages, err := mock.GetHistory(ctx, "U123")

		// Then: Should return context error
		require.Error(t, err)
		assert.Nil(t, messages)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// TestStorage_AppendMessages tests the AppendMessages method.
// Spec: AppendMessages saves user message and bot response atomically
func TestStorage_AppendMessages(t *testing.T) {
	t.Run("AppendMessages saves both messages atomically", func(t *testing.T) {
		// Given: Mock storage
		mock := &mockStorage{
			messages: []history.Message{},
		}

		userMsg := history.Message{
			Role:      "user",
			Content:   "Hello",
			Timestamp: time.Now(),
		}
		botMsg := history.Message{
			Role:      "assistant",
			Content:   "Hi",
			Timestamp: time.Now(),
		}

		// When: Append messages
		ctx := context.Background()
		err := mock.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should succeed
		require.NoError(t, err)
		assert.Len(t, mock.messages, 2)
		assert.Equal(t, "user", mock.messages[0].Role)
		assert.Equal(t, "assistant", mock.messages[1].Role)
	})

	t.Run("AppendMessages maintains consistency", func(t *testing.T) {
		// Given: Mock storage
		mock := &mockStorage{
			messages: []history.Message{},
		}

		// When: Append multiple message pairs
		ctx := context.Background()

		msg1User := history.Message{Role: "user", Content: "First", Timestamp: time.Now()}
		msg1Bot := history.Message{Role: "assistant", Content: "Response 1", Timestamp: time.Now()}
		err := mock.AppendMessages(ctx, "U123", msg1User, msg1Bot)
		require.NoError(t, err)

		msg2User := history.Message{Role: "user", Content: "Second", Timestamp: time.Now()}
		msg2Bot := history.Message{Role: "assistant", Content: "Response 2", Timestamp: time.Now()}
		err = mock.AppendMessages(ctx, "U123", msg2User, msg2Bot)
		require.NoError(t, err)

		// Then: Should maintain order
		assert.Len(t, mock.messages, 4)
		assert.Equal(t, "First", mock.messages[0].Content)
		assert.Equal(t, "Response 1", mock.messages[1].Content)
		assert.Equal(t, "Second", mock.messages[2].Content)
		assert.Equal(t, "Response 2", mock.messages[3].Content)
	})

	t.Run("AppendMessages returns error on storage failure", func(t *testing.T) {
		// Given: Mock storage that fails writes
		mock := &mockStorage{
			appendMessagesErr: &history.StorageWriteError{Message: "write failed"},
		}

		// When: Append messages
		ctx := context.Background()
		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}
		err := mock.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should return error
		require.Error(t, err)
		var writeErr *history.StorageWriteError
		assert.ErrorAs(t, err, &writeErr)
	})

	t.Run("AppendMessages handles timeout", func(t *testing.T) {
		// Given: Mock storage that times out
		mock := &mockStorage{
			appendMessagesErr: &history.StorageTimeoutError{Message: "timeout"},
		}

		// When: Append messages
		ctx := context.Background()
		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}
		err := mock.AppendMessages(ctx, "U123", userMsg, botMsg)

		// Then: Should return timeout error
		require.Error(t, err)
		var timeoutErr *history.StorageTimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})
}

// TestStorage_Close tests the Close method.
// Spec: Close releases storage resources
func TestStorage_Close(t *testing.T) {
	t.Run("Close releases resources", func(t *testing.T) {
		// Given: Mock storage
		mock := &mockStorage{}

		// When: Close storage
		ctx := context.Background()
		err := mock.Close(ctx)

		// Then: Should succeed
		require.NoError(t, err)
		assert.True(t, mock.closed)
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		// Given: Mock storage
		mock := &mockStorage{}

		ctx := context.Background()

		// When: Close multiple times
		err1 := mock.Close(ctx)
		err2 := mock.Close(ctx)
		err3 := mock.Close(ctx)

		// Then: All should succeed
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})
}

// =============================================================================
// Mock Storage Implementation
// =============================================================================

// mockStorage is a test implementation of the Storage interface.
type mockStorage struct {
	messages          []history.Message
	getHistoryErr     error
	appendMessagesErr error
	checkContext      bool
	closed            bool
}

func (m *mockStorage) GetHistory(ctx context.Context, sourceID string) ([]history.Message, error) {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	if m.getHistoryErr != nil {
		return nil, m.getHistoryErr
	}

	return m.messages, nil
}

func (m *mockStorage) AppendMessages(ctx context.Context, sourceID string, userMsg, botMsg history.Message) error {
	if m.checkContext {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if m.appendMessagesErr != nil {
		return m.appendMessagesErr
	}

	m.messages = append(m.messages, userMsg, botMsg)
	return nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	m.closed = true
	return nil
}
