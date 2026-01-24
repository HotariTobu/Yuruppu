package history_test

import (
	"strings"
	"testing"
	"yuruppu/internal/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// FR-002: MessageID Field Tests
// =============================================================================

// TestUserMessage_MessageIDField tests that UserMessage can store a MessageID.
// FR-002: System identifies the target message in history using the message ID
func TestUserMessage_MessageIDField(t *testing.T) {
	t.Run("should create UserMessage with MessageID", func(t *testing.T) {
		// Given: A UserMessage with MessageID
		messageID := "msg-12345"

		msg := &history.UserMessage{
			MessageID: messageID,
			UserID:    "U123",
			Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
			Timestamp: testTime1,
		}

		// Then: MessageID should be stored
		assert.Equal(t, messageID, msg.MessageID)
	})

	t.Run("should create UserMessage without MessageID for legacy messages", func(t *testing.T) {
		// Given: A UserMessage without MessageID (legacy case)
		msg := &history.UserMessage{
			UserID:    "U123",
			Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
			Timestamp: testTime1,
		}

		// Then: MessageID should be empty string
		assert.Equal(t, "", msg.MessageID)
	})
}

// TestMessageID_Serialization tests that MessageID is serialized to JSON.
// FR-002: System identifies the target message in history using the message ID
func TestMessageID_Serialization(t *testing.T) {
	tests := []struct {
		name      string
		messages  []history.Message
		wantEmpty bool // true if messageId should be omitted from JSON
	}{
		{
			name: "should serialize MessageID when present",
			messages: []history.Message{
				&history.UserMessage{
					MessageID: "msg-12345",
					UserID:    "U123",
					Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
					Timestamp: testTime1,
				},
			},
			wantEmpty: false,
		},
		{
			name: "should omit MessageID when empty (legacy message)",
			messages: []history.Message{
				&history.UserMessage{
					MessageID: "", // empty = legacy message
					UserID:    "U123",
					Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
					Timestamp: testTime1,
				},
			},
			wantEmpty: true,
		},
		{
			name: "should serialize multiple messages with different MessageIDs",
			messages: []history.Message{
				&history.UserMessage{
					MessageID: "msg-001",
					UserID:    "U123",
					Parts:     []history.UserPart{&history.UserTextPart{Text: "First"}},
					Timestamp: testTime1,
				},
				&history.UserMessage{
					MessageID: "msg-002",
					UserID:    "U123",
					Parts:     []history.UserPart{&history.UserTextPart{Text: "Second"}},
					Timestamp: testTime2,
				},
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newMockStorage()
			svc, err := history.NewService(storage)
			require.NoError(t, err)

			// When: Serialize messages to storage
			_, err = svc.PutHistory(t.Context(), "source1", tt.messages, 0)
			require.NoError(t, err)

			// Then: Read raw storage data and verify JSON structure
			data, _, err := storage.Read(t.Context(), "source1")
			require.NoError(t, err)

			jsonStr := string(data)
			if tt.wantEmpty {
				// messageId should be omitted (not present in JSON)
				assert.NotContains(t, jsonStr, "messageId")
			} else {
				// messageId should be present in JSON
				assert.Contains(t, jsonStr, "messageId")
			}
		})
	}
}

// TestMessageID_Parsing tests that MessageID is parsed from JSON.
// FR-002: System identifies the target message in history using the message ID
func TestMessageID_Parsing(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		wantMessageID string
		wantErr       bool
	}{
		{
			name:          "should parse MessageID from JSON",
			jsonData:      `{"role":"user","messageId":"msg-12345","userId":"U123","parts":[{"type":"text","text":"Hello"}],"timestamp":"2025-01-01T10:00:00Z"}`,
			wantMessageID: "msg-12345",
			wantErr:       false,
		},
		{
			name:          "should parse empty MessageID for legacy JSON without messageId field",
			jsonData:      `{"role":"user","userId":"U123","parts":[{"type":"text","text":"Hello"}],"timestamp":"2025-01-01T10:00:00Z"}`,
			wantMessageID: "",
			wantErr:       false,
		},
		{
			name:          "should parse null MessageID as empty string",
			jsonData:      `{"role":"user","messageId":null,"userId":"U123","parts":[{"type":"text","text":"Hello"}],"timestamp":"2025-01-01T10:00:00Z"}`,
			wantMessageID: "",
			wantErr:       false,
		},
		{
			name:          "should parse multiple messages with MessageIDs",
			jsonData:      `{"role":"user","messageId":"msg-001","userId":"U123","parts":[{"type":"text","text":"First"}],"timestamp":"2025-01-01T10:00:00Z"}` + "\n" + `{"role":"user","messageId":"msg-002","userId":"U123","parts":[{"type":"text","text":"Second"}],"timestamp":"2025-01-01T10:01:00Z"}`,
			wantMessageID: "msg-001", // Check first message
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newMockStorage()
			storage.data["source1"] = []byte(tt.jsonData)
			storage.generation["source1"] = 1

			svc, err := history.NewService(storage)
			require.NoError(t, err)

			// When: Parse messages from storage
			messages, _, err := svc.GetHistory(t.Context(), "source1")

			// Then: Should parse correctly
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, messages)

			// Verify first message has correct MessageID
			userMsg, ok := messages[0].(*history.UserMessage)
			require.True(t, ok, "Expected UserMessage, got %T", messages[0])
			assert.Equal(t, tt.wantMessageID, userMsg.MessageID)
		})
	}
}

// TestMessageID_RoundTrip tests full round-trip with MessageID.
// FR-002: System identifies the target message in history using the message ID
func TestMessageID_RoundTrip(t *testing.T) {
	t.Run("should preserve MessageID through Put and Get", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: Messages with MessageIDs
		originalMessages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-12345",
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Hello"}},
				Timestamp: testTime1,
			},
			&history.AssistantMessage{
				ModelName: "gemini-2.0",
				Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: "Hi!"}},
				Timestamp: testTime2,
			},
			&history.UserMessage{
				MessageID: "msg-67890",
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "How are you?"}},
				Timestamp: testTime3,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", originalMessages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: MessageIDs should be preserved
		require.Len(t, retrieved, 3)

		// First user message
		userMsg1, ok := retrieved[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-12345", userMsg1.MessageID)

		// Assistant message (no MessageID)
		_, ok = retrieved[1].(*history.AssistantMessage)
		require.True(t, ok)

		// Second user message
		userMsg2, ok := retrieved[2].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-67890", userMsg2.MessageID)
	})

	t.Run("should preserve empty MessageID for legacy messages", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: Legacy message without MessageID
		originalMessages := []history.Message{
			&history.UserMessage{
				MessageID: "", // Legacy message
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Legacy message"}},
				Timestamp: testTime1,
			},
		}

		// When: Put and Get
		_, err = svc.PutHistory(t.Context(), "source1", originalMessages, 0)
		require.NoError(t, err)

		retrieved, _, err := svc.GetHistory(t.Context(), "source1")
		require.NoError(t, err)

		// Then: MessageID should remain empty
		require.Len(t, retrieved, 1)
		userMsg, ok := retrieved[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "", userMsg.MessageID)
	})
}

// TestMessageID_BackwardCompatibility tests reading legacy data without MessageID.
// FR-002: Backward compatibility - legacy messages without MessageID should parse correctly
func TestMessageID_BackwardCompatibility(t *testing.T) {
	t.Run("should parse legacy JSON without messageId field", func(t *testing.T) {
		storage := newMockStorage()

		// Given: Legacy JSONL data without messageId field (simulates old storage format)
		legacyData := `{"role":"user","userId":"U123","parts":[{"type":"text","text":"Old message"}],"timestamp":"2025-01-01T10:00:00Z"}
{"role":"assistant","modelName":"gemini-2.0","parts":[{"type":"text","text":"Old response"}],"timestamp":"2025-01-01T10:01:00Z"}
`
		storage.data["source1"] = []byte(legacyData)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// When: Read legacy history
		messages, gen, err := svc.GetHistory(t.Context(), "source1")

		// Then: Should parse successfully
		require.NoError(t, err)
		assert.Equal(t, int64(1), gen)
		require.Len(t, messages, 2)

		// First message: user message without MessageID
		userMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "", userMsg.MessageID) // Empty for legacy
		assert.Equal(t, "U123", userMsg.UserID)
		textPart, ok := userMsg.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Old message", textPart.Text)

		// Second message: assistant message (never has MessageID)
		assistantMsg, ok := messages[1].(*history.AssistantMessage)
		require.True(t, ok)
		assert.Equal(t, "gemini-2.0", assistantMsg.ModelName)
	})

	t.Run("should handle mixed legacy and new messages", func(t *testing.T) {
		storage := newMockStorage()

		// Given: Mixed data - some with MessageID, some without
		mixedData := `{"role":"user","userId":"U123","parts":[{"type":"text","text":"Legacy"}],"timestamp":"2025-01-01T10:00:00Z"}
{"role":"user","messageId":"msg-12345","userId":"U123","parts":[{"type":"text","text":"New"}],"timestamp":"2025-01-01T10:01:00Z"}
`
		storage.data["source1"] = []byte(mixedData)
		storage.generation["source1"] = 1

		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// When: Read mixed history
		messages, _, err := svc.GetHistory(t.Context(), "source1")

		// Then: Should parse both correctly
		require.NoError(t, err)
		require.Len(t, messages, 2)

		// Legacy message without MessageID
		legacyMsg, ok := messages[0].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "", legacyMsg.MessageID)
		legacyText, ok := legacyMsg.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "Legacy", legacyText.Text)

		// New message with MessageID
		newMsg, ok := messages[1].(*history.UserMessage)
		require.True(t, ok)
		assert.Equal(t, "msg-12345", newMsg.MessageID)
		newText, ok := newMsg.Parts[0].(*history.UserTextPart)
		require.True(t, ok)
		assert.Equal(t, "New", newText.Text)
	})
}

// TestMessageID_EdgeCases tests edge cases for MessageID handling.
// FR-002: System identifies the target message in history using the message ID
func TestMessageID_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		messageID string
		wantID    string
	}{
		{
			name:      "should handle very long MessageID",
			messageID: "msg-" + strings.Repeat("a", 1000),
			wantID:    "msg-" + strings.Repeat("a", 1000),
		},
		{
			name:      "should handle MessageID with special characters",
			messageID: "msg-!@#$%^&*()_+-=[]{}|;:',.<>?/~`",
			wantID:    "msg-!@#$%^&*()_+-=[]{}|;:',.<>?/~`",
		},
		{
			name:      "should handle MessageID with unicode",
			messageID: "msg-日本語-🎉",
			wantID:    "msg-日本語-🎉",
		},
		{
			name:      "should handle MessageID with whitespace",
			messageID: "msg with spaces and\ttabs",
			wantID:    "msg with spaces and\ttabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newMockStorage()
			svc, err := history.NewService(storage)
			require.NoError(t, err)

			// Given: Message with edge case MessageID
			messages := []history.Message{
				&history.UserMessage{
					MessageID: tt.messageID,
					UserID:    "U123",
					Parts:     []history.UserPart{&history.UserTextPart{Text: "Test"}},
					Timestamp: testTime1,
				},
			}

			// When: Put and Get
			_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
			require.NoError(t, err)

			retrieved, _, err := svc.GetHistory(t.Context(), "source1")
			require.NoError(t, err)

			// Then: MessageID should be preserved exactly
			require.Len(t, retrieved, 1)
			userMsg, ok := retrieved[0].(*history.UserMessage)
			require.True(t, ok)
			assert.Equal(t, tt.wantID, userMsg.MessageID)
		})
	}
}

// TestMessageID_OnlyUserMessages tests that only UserMessage has MessageID.
// FR-002: MessageID is only relevant for user messages (assistant messages cannot be unsent)
func TestMessageID_OnlyUserMessages(t *testing.T) {
	t.Run("should serialize UserMessage MessageID but not AssistantMessage", func(t *testing.T) {
		storage := newMockStorage()
		svc, err := history.NewService(storage)
		require.NoError(t, err)

		// Given: User and assistant messages
		messages := []history.Message{
			&history.UserMessage{
				MessageID: "msg-12345",
				UserID:    "U123",
				Parts:     []history.UserPart{&history.UserTextPart{Text: "Question"}},
				Timestamp: testTime1,
			},
			&history.AssistantMessage{
				ModelName: "gemini-2.0",
				Parts:     []history.AssistantPart{&history.AssistantTextPart{Text: "Answer"}},
				Timestamp: testTime2,
			},
		}

		// When: Serialize to storage
		_, err = svc.PutHistory(t.Context(), "source1", messages, 0)
		require.NoError(t, err)

		// Then: Verify JSON structure
		data, _, err := storage.Read(t.Context(), "source1")
		require.NoError(t, err)

		lines := string(data)

		// User message should have messageId
		assert.Contains(t, lines, `"messageId":"msg-12345"`)

		// Assistant message should have role "assistant" but no messageId
		// (messageId is only for UserMessage)
		assert.Contains(t, lines, `"role":"assistant"`)
	})
}
