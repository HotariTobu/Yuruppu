//go:build !integration

package history_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"yuruppu/internal/history"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GCS Storage Constructor Tests
// =============================================================================

// TestNewGCSStorageWithBucket tests the GCS storage constructor with mock bucket.
// AC-001: メッセージ履歴の保存 [FR-001]
func TestNewGCSStorageWithBucket(t *testing.T) {
	t.Run("should create GCS storage with valid bucket", func(t *testing.T) {
		// Given: Mock bucket
		mockBucket := &mockBucketHandle{}

		// When: Create new GCS storage
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// Then: Should create storage successfully
		assert.NotNil(t, gcsStorage)
	})
}

// =============================================================================
// GetHistory Tests - Happy Path
// =============================================================================

// TestGCSStorage_GetHistory_EmptyHistory tests getting history when no file exists.
// AC-001: メッセージ履歴の保存 - empty history should return empty slice
func TestGCSStorage_GetHistory_EmptyHistory(t *testing.T) {
	t.Run("should return empty slice when history file does not exist", func(t *testing.T) {
		// Given: GCS storage with no existing history (object not found)
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {
					notFound: true,
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history for sourceID
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return empty slice with no error
		require.NoError(t, err)
		assert.Empty(t, messages)
		assert.NotNil(t, messages) // Should be empty slice, not nil
	})
}

// TestGCSStorage_GetHistory_ValidHistory tests retrieving existing history.
// AC-002: コンテキストを含む応答生成 [FR-002]
func TestGCSStorage_GetHistory_ValidHistory(t *testing.T) {
	t.Run("should parse JSONL and return messages in order", func(t *testing.T) {
		// Given: JSONL data in GCS
		jsonlData := buildJSONL([]history.Message{
			{
				Role:      "user",
				Content:   "私の名前は太郎です",
				Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC),
			},
			{
				Role:      "assistant",
				Content:   "太郎さん、こんにちは！",
				Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC),
			},
			{
				Role:      "user",
				Content:   "私の名前を覚えてる？",
				Timestamp: time.Date(2025, 12, 28, 10, 5, 0, 0, time.UTC),
			},
		})

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {
					data: jsonlData,
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return all messages in order
		require.NoError(t, err)
		assert.Len(t, messages, 3)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "私の名前は太郎です", messages[0].Content)
		assert.Equal(t, "assistant", messages[1].Role)
		assert.Equal(t, "太郎さん、こんにちは！", messages[1].Content)
		assert.Equal(t, "user", messages[2].Role)
	})

	t.Run("should handle single message", func(t *testing.T) {
		// Given: Single message in JSONL
		jsonlData := buildJSONL([]history.Message{
			{
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
		})

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U456def.jsonl": {
					data: jsonlData,
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U456def")

		// Then: Should return single message
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
	})

	t.Run("should preserve timestamps when parsing", func(t *testing.T) {
		// Given: Messages with specific timestamps
		expectedTime := time.Date(2025, 12, 28, 15, 30, 45, 0, time.UTC)
		jsonlData := buildJSONL([]history.Message{
			{
				Role:      "user",
				Content:   "Test",
				Timestamp: expectedTime,
			},
		})

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U789ghi.jsonl": {
					data: jsonlData,
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U789ghi")

		// Then: Timestamp should be preserved
		require.NoError(t, err)
		assert.Equal(t, expectedTime.Unix(), messages[0].Timestamp.Unix())
	})
}

// TestGCSStorage_GetHistory_SourceIDIsolation tests that different SourceIDs have separate histories.
// AC-003: 会話ソース間の履歴分離 [FR-003]
func TestGCSStorage_GetHistory_SourceIDIsolation(t *testing.T) {
	t.Run("should retrieve history for correct sourceID only", func(t *testing.T) {
		// Given: Different histories for different sourceIDs
		user1History := buildJSONL([]history.Message{
			{Role: "user", Content: "好きな食べ物はラーメン", Timestamp: time.Now()},
		})
		groupHistory := buildJSONL([]history.Message{
			{Role: "user", Content: "好きな食べ物は寿司", Timestamp: time.Now()},
		})

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U111aaa.jsonl": {data: user1History},
				"C222bbb.jsonl": {data: groupHistory},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history for user1
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U111aaa")

		// Then: Should only return user1's history
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Contains(t, messages[0].Content, "ラーメン")
		assert.NotContains(t, messages[0].Content, "寿司")
	})

	t.Run("should use sourceID as filename", func(t *testing.T) {
		// Given: Storage with multiple source files
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {data: buildJSONL([]history.Message{{Role: "user", Content: "User chat", Timestamp: time.Now()}})},
				"C456def.jsonl": {data: buildJSONL([]history.Message{{Role: "user", Content: "Group chat", Timestamp: time.Now()}})},
				"R789ghi.jsonl": {data: buildJSONL([]history.Message{{Role: "user", Content: "Room chat", Timestamp: time.Now()}})},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Request each sourceID
		ctx := context.Background()

		userMsgs, _ := gcsStorage.GetHistory(ctx, "U123abc")
		groupMsgs, _ := gcsStorage.GetHistory(ctx, "C456def")
		roomMsgs, _ := gcsStorage.GetHistory(ctx, "R789ghi")

		// Then: Should get correct content for each
		assert.Contains(t, userMsgs[0].Content, "User chat")
		assert.Contains(t, groupMsgs[0].Content, "Group chat")
		assert.Contains(t, roomMsgs[0].Content, "Room chat")
	})
}

// =============================================================================
// GetHistory Tests - Error Cases
// =============================================================================

// TestGCSStorage_GetHistory_ReadError tests error handling when reading fails.
// AC-004: ストレージ障害時の動作 [NFR-002]
func TestGCSStorage_GetHistory_ReadError(t *testing.T) {
	t.Run("should return StorageReadError when GCS read fails", func(t *testing.T) {
		// Given: GCS object that returns read error
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {
					readError: errors.New("GCS bucket permission denied"),
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return StorageReadError
		require.Error(t, err)
		assert.Nil(t, messages)

		var readErr *history.StorageReadError
		assert.ErrorAs(t, err, &readErr)
		assert.Contains(t, readErr.Message, "failed to read")
	})

	t.Run("should return StorageReadError when JSONL parsing fails", func(t *testing.T) {
		// Given: Invalid JSONL data
		invalidJSONL := `{"Role":"user","Content":"Valid"}
{"Role":"user","Content":INVALID_JSON}
{"Role":"assistant","Content":"Also valid"}`

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {
					data: invalidJSONL,
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history
		ctx := context.Background()
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return StorageReadError
		require.Error(t, err)
		assert.Nil(t, messages)

		var readErr *history.StorageReadError
		assert.ErrorAs(t, err, &readErr)
	})
}

// TestGCSStorage_GetHistory_Timeout tests timeout handling.
// NFR-001: 履歴の読み書きはメッセージ処理のレイテンシに大きな影響を与えない（+100ms以内）
func TestGCSStorage_GetHistory_Timeout(t *testing.T) {
	t.Run("should return StorageTimeoutError when context times out", func(t *testing.T) {
		// Given: Context that is already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": {
					data: buildJSONL([]history.Message{{Role: "user", Content: "Test", Timestamp: time.Now()}}),
				},
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history with cancelled context
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return StorageTimeoutError
		require.Error(t, err)
		assert.Nil(t, messages)

		var timeoutErr *history.StorageTimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})

	t.Run("should respect context cancellation", func(t *testing.T) {
		// Given: Cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockBucket := &mockBucketHandle{}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Get history with cancelled context
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")

		// Then: Should return error
		require.Error(t, err)
		assert.Nil(t, messages)
	})
}

// =============================================================================
// AppendMessages Tests - Happy Path
// =============================================================================

// TestGCSStorage_AppendMessages_NewHistory tests appending to new (empty) history.
// AC-001: メッセージ履歴の保存 [FR-001]
func TestGCSStorage_AppendMessages_NewHistory(t *testing.T) {
	t.Run("should create new JSONL file when history does not exist", func(t *testing.T) {
		// Given: Empty GCS storage
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{
			Role:      "user",
			Content:   "Hello, Yuruppu!",
			Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC),
		}
		botMsg := history.Message{
			Role:      "assistant",
			Content:   "こんにちは！",
			Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC),
		}

		// When: Append messages
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Should create new file with both messages
		require.NoError(t, err)

		// Verify JSONL content
		written := mockObj.writtenData.String()
		lines := strings.Split(strings.TrimSpace(written), "\n")
		assert.Len(t, lines, 2)

		var msg1, msg2 history.Message
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &msg1))
		require.NoError(t, json.Unmarshal([]byte(lines[1]), &msg2))

		assert.Equal(t, "user", msg1.Role)
		assert.Equal(t, "Hello, Yuruppu!", msg1.Content)
		assert.Equal(t, "assistant", msg2.Role)
		assert.Equal(t, "こんにちは！", msg2.Content)
	})

	t.Run("should preserve timestamp accuracy", func(t *testing.T) {
		// Given: Messages with precise timestamps
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userTime := time.Date(2025, 12, 28, 15, 30, 45, 0, time.UTC)
		botTime := time.Date(2025, 12, 28, 15, 30, 46, 0, time.UTC)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: userTime}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: botTime}

		// When: Append messages
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Timestamps should be preserved
		require.NoError(t, err)

		written := mockObj.writtenData.String()
		lines := strings.Split(strings.TrimSpace(written), "\n")

		var parsedUser history.Message
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &parsedUser))
		assert.Equal(t, userTime.Unix(), parsedUser.Timestamp.Unix())
	})
}

// TestGCSStorage_AppendMessages_ExistingHistory tests appending to existing history.
// ADR: Read-Modify-Write pattern for appending messages
func TestGCSStorage_AppendMessages_ExistingHistory(t *testing.T) {
	t.Run("should append to existing history using read-modify-write", func(t *testing.T) {
		// Given: Existing history
		existingHistory := buildJSONL([]history.Message{
			{Role: "user", Content: "First message", Timestamp: time.Date(2025, 12, 28, 9, 0, 0, 0, time.UTC)},
			{Role: "assistant", Content: "First response", Timestamp: time.Date(2025, 12, 28, 9, 1, 0, 0, time.UTC)},
		})

		mockObj := &mockObjectHandle{
			data:        existingHistory,
			generation:  1,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		newUserMsg := history.Message{Role: "user", Content: "Second message", Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC)}
		newBotMsg := history.Message{Role: "assistant", Content: "Second response", Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC)}

		// When: Append messages
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", newUserMsg, newBotMsg)

		// Then: Should append to existing history
		require.NoError(t, err)

		written := mockObj.writtenData.String()
		lines := strings.Split(strings.TrimSpace(written), "\n")
		assert.Len(t, lines, 4) // 2 existing + 2 new

		// Verify order is preserved
		var msg3 history.Message
		require.NoError(t, json.Unmarshal([]byte(lines[2]), &msg3))
		assert.Equal(t, "Second message", msg3.Content)
	})
}

// TestGCSStorage_AppendMessages_AtomicPair tests that both messages are saved atomically.
// AC-001: ボットの応答が履歴に保存される
func TestGCSStorage_AppendMessages_AtomicPair(t *testing.T) {
	t.Run("should save both user and bot messages in single write", func(t *testing.T) {
		// Given: GCS storage
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{Role: "user", Content: "User message", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Bot response", Timestamp: time.Now()}

		// When: Append messages
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Both messages should be in the same write
		require.NoError(t, err)
		assert.Equal(t, 1, mockObj.writeCount) // Single write operation

		written := mockObj.writtenData.String()
		lines := strings.Split(strings.TrimSpace(written), "\n")
		assert.Len(t, lines, 2) // Both messages present
	})
}

// =============================================================================
// AppendMessages Tests - Error Cases
// =============================================================================

// TestGCSStorage_AppendMessages_WriteError tests write failure handling.
// AC-004: ストレージ障害時の動作 [NFR-002]
func TestGCSStorage_AppendMessages_WriteError(t *testing.T) {
	t.Run("should return StorageWriteError when GCS write fails", func(t *testing.T) {
		// Given: GCS object that fails writes
		mockObj := &mockObjectHandle{
			notFound:   true,
			writeError: errors.New("GCS write permission denied"),
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: Append messages
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Should return StorageWriteError
		require.Error(t, err)

		var writeErr *history.StorageWriteError
		assert.ErrorAs(t, err, &writeErr)
		assert.Contains(t, writeErr.Message, "failed to write")
	})

	t.Run("should return StorageWriteError when read phase fails", func(t *testing.T) {
		// Given: GCS object that fails reads (during read-modify-write)
		mockObj := &mockObjectHandle{
			readError: errors.New("read failed"),
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: Append to existing history
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Should return error (read-modify-write failed at read)
		require.Error(t, err)
		var writeErr *history.StorageWriteError
		assert.ErrorAs(t, err, &writeErr)
	})
}

// TestGCSStorage_AppendMessages_PreconditionFailed tests handling of concurrent writes.
// ADR: Generation preconditions to detect conflicts (handle 412 Precondition Failed)
func TestGCSStorage_AppendMessages_PreconditionFailed(t *testing.T) {
	t.Run("should return StorageWriteError when generation precondition fails", func(t *testing.T) {
		// Given: GCS object that simulates concurrent modification (412)
		mockObj := &mockObjectHandle{
			data:              buildJSONL([]history.Message{{Role: "user", Content: "Existing", Timestamp: time.Now()}}),
			generation:        1,
			preconditionError: errors.New("precondition failed: generation mismatch"),
			writtenData:       &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: Append with outdated generation
		ctx := context.Background()
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Should return StorageWriteError with precondition details
		require.Error(t, err)

		var writeErr *history.StorageWriteError
		assert.ErrorAs(t, err, &writeErr)
		assert.Contains(t, writeErr.Message, "precondition")
	})
}

// TestGCSStorage_AppendMessages_Timeout tests timeout during append.
// NFR-001: 履歴の読み書きはメッセージ処理のレイテンシに大きな影響を与えない（+100ms以内）
func TestGCSStorage_AppendMessages_Timeout(t *testing.T) {
	t.Run("should return StorageTimeoutError when context is cancelled", func(t *testing.T) {
		// Given: Context that is already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		userMsg := history.Message{Role: "user", Content: "Test", Timestamp: time.Now()}
		botMsg := history.Message{Role: "assistant", Content: "Response", Timestamp: time.Now()}

		// When: Append with cancelled context
		err := gcsStorage.AppendMessages(ctx, "U123abc", userMsg, botMsg)

		// Then: Should return StorageTimeoutError
		require.Error(t, err)

		var timeoutErr *history.StorageTimeoutError
		assert.ErrorAs(t, err, &timeoutErr)
	})
}

// =============================================================================
// Close Tests
// =============================================================================

// TestGCSStorage_Close tests the Close method.
// Spec: Close releases storage resources
func TestGCSStorage_Close(t *testing.T) {
	t.Run("should close without error when GCS client is managed externally", func(t *testing.T) {
		// Given: GCS storage
		mockBucket := &mockBucketHandle{}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		// When: Close storage
		ctx := context.Background()
		err := gcsStorage.Close(ctx)

		// Then: Should succeed (GCS client is managed externally, so Close is no-op)
		require.NoError(t, err)
	})

	t.Run("should be callable multiple times", func(t *testing.T) {
		// Given: GCS storage
		mockBucket := &mockBucketHandle{}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()

		// When: Close multiple times
		err1 := gcsStorage.Close(ctx)
		err2 := gcsStorage.Close(ctx)
		err3 := gcsStorage.Close(ctx)

		// Then: All calls should succeed
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})
}

// =============================================================================
// Integration Scenarios
// =============================================================================

// TestGCSStorage_RoundTrip tests writing and reading back messages.
func TestGCSStorage_RoundTrip(t *testing.T) {
	t.Run("should preserve messages through write-read cycle", func(t *testing.T) {
		// Given: Empty storage
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"U123abc.jsonl": mockObj,
			},
		}
		gcsStorage := history.NewGCSStorageWithBucket(mockBucket)

		originalUser := history.Message{
			Role:      "user",
			Content:   "私の名前は太郎です",
			Timestamp: time.Date(2025, 12, 28, 10, 0, 0, 0, time.UTC),
		}
		originalBot := history.Message{
			Role:      "assistant",
			Content:   "太郎さん、こんにちは！",
			Timestamp: time.Date(2025, 12, 28, 10, 1, 0, 0, time.UTC),
		}

		ctx := context.Background()

		// When: Write messages
		err := gcsStorage.AppendMessages(ctx, "U123abc", originalUser, originalBot)
		require.NoError(t, err)

		// Simulate GCS persistence: update the mock object data
		mockObj.data = mockObj.writtenData.String()
		mockObj.notFound = false

		// Then: Read back should return same messages
		messages, err := gcsStorage.GetHistory(ctx, "U123abc")
		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, originalUser.Role, messages[0].Role)
		assert.Equal(t, originalUser.Content, messages[0].Content)
		assert.Equal(t, originalBot.Role, messages[1].Role)
		assert.Equal(t, originalBot.Content, messages[1].Content)
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// buildJSONL creates JSONL string from messages.
// Panics on marshal error (test helper only).
func buildJSONL(messages []history.Message) string {
	var builder strings.Builder
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			panic(err)
		}
		builder.Write(data)
		builder.WriteString("\n")
	}
	return builder.String()
}

// =============================================================================
// Mock Implementations
// =============================================================================

// mockBucketHandle implements history.BucketHandle for testing.
type mockBucketHandle struct {
	objects map[string]*mockObjectHandle
}

func (m *mockBucketHandle) Object(name string) history.ObjectHandle {
	if obj, ok := m.objects[name]; ok {
		obj.name = name
		return obj
	}
	// Return a new mock object that will return not found
	return &mockObjectHandle{name: name, notFound: true}
}

// mockObjectHandle implements history.ObjectHandle for testing.
type mockObjectHandle struct {
	name       string
	data       string
	notFound   bool
	generation int64

	// Error simulation
	readError         error
	writeError        error
	preconditionError error

	// Write tracking
	writtenData *bytes.Buffer
	writeCount  int
}

func (m *mockObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	if m.notFound {
		return nil, storage.ErrObjectNotExist
	}
	if m.readError != nil {
		return nil, m.readError
	}
	return io.NopCloser(strings.NewReader(m.data)), nil
}

func (m *mockObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	if m.writtenData == nil {
		m.writtenData = &bytes.Buffer{}
	}
	return &mockWriter{
		obj:    m,
		buffer: m.writtenData,
	}
}

func (m *mockObjectHandle) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	if m.notFound {
		return nil, storage.ErrObjectNotExist
	}
	return &storage.ObjectAttrs{
		Generation: m.generation,
	}, nil
}

func (m *mockObjectHandle) Generation(gen int64) history.ObjectHandle {
	// Return the same object (generation precondition is checked on write)
	return m
}

func (m *mockObjectHandle) If(conds storage.Conditions) history.ObjectHandle {
	// Return the same object (precondition is checked on write)
	return m
}

// mockWriter implements io.WriteCloser for mock writes.
type mockWriter struct {
	obj    *mockObjectHandle
	buffer *bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (int, error) {
	if w.obj.writeError != nil {
		return 0, w.obj.writeError
	}
	return w.buffer.Write(p)
}

func (w *mockWriter) Close() error {
	w.obj.writeCount++
	if w.obj.preconditionError != nil {
		return w.obj.preconditionError
	}
	if w.obj.writeError != nil {
		return w.obj.writeError
	}
	return nil
}
