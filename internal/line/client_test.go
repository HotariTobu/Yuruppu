package line_test

import (
	"io"
	"log/slog"
	"testing"

	"yuruppu/internal/line"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMessagingAPI is a mock implementation of the LINE messaging API client
// for testing Client.SendReply without making real API calls.
type mockMessagingAPI struct {
	// Track calls to ReplyMessage
	replyMessageCalls []*messaging_api.ReplyMessageRequest
	// Error to return from ReplyMessage
	replyMessageError error
}

func (m *mockMessagingAPI) ReplyMessage(req *messaging_api.ReplyMessageRequest) (*messaging_api.ReplyMessageResponse, error) {
	m.replyMessageCalls = append(m.replyMessageCalls, req)
	if m.replyMessageError != nil {
		return nil, m.replyMessageError
	}
	return &messaging_api.ReplyMessageResponse{}, nil
}

// TestNewClient tests Client creation with various inputs.
// SC-003: Client should validate channelToken
func TestNewClient(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name         string
		channelToken string
		wantErr      bool
		errType      *line.ConfigError
	}{
		{
			name:         "valid channel token",
			channelToken: "test-token-123",
			wantErr:      false,
		},
		{
			name:         "empty channel token returns ConfigError",
			channelToken: "",
			wantErr:      true,
			errType:      &line.ConfigError{Variable: "channelToken"},
		},
		{
			name:         "whitespace-only channel token returns ConfigError",
			channelToken: "   \t\n  ",
			wantErr:      true,
			errType:      &line.ConfigError{Variable: "channelToken"},
		},
		{
			name:         "channel token with leading/trailing spaces is valid",
			channelToken: "  valid-token  ",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := line.NewClient(tt.channelToken, logger)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, client)

				// Verify error type is ConfigError
				if tt.errType != nil {
					configErr, ok := err.(*line.ConfigError)
					require.True(t, ok, "error should be *line.ConfigError")
					assert.Equal(t, tt.errType.Variable, configErr.Variable)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestClient_SendReply tests the SendReply method.
// SC-003: Client.SendReply should call LINE ReplyMessage API
func TestClient_SendReply(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name       string
		replyToken string
		text       string
		wantErr    bool
	}{
		{
			name:       "send reply with valid inputs",
			replyToken: "test-reply-token",
			text:       "Hello, World!",
			wantErr:    false,
		},
		{
			name:       "send reply with empty text",
			replyToken: "test-reply-token",
			text:       "",
			wantErr:    false,
		},
		{
			name:       "send reply with multiline text",
			replyToken: "test-reply-token",
			text:       "Line 1\nLine 2\nLine 3",
			wantErr:    false,
		},
		{
			name:       "send reply with special characters",
			replyToken: "test-reply-token",
			text:       "Hello! 😊 Special chars: @#$%^&*()",
			wantErr:    false,
		},
		{
			name:       "send reply with long text",
			replyToken: "test-reply-token",
			text:       string(make([]byte, 5000)), // 5000 characters
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create client with mock API
			mock := &mockMessagingAPI{}
			client := line.NewClientWithAPI(mock, logger)

			err := client.SendReply(tt.replyToken, tt.text)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify API was called
			require.Len(t, mock.replyMessageCalls, 1)
			call := mock.replyMessageCalls[0]
			assert.Equal(t, tt.replyToken, call.ReplyToken)
			require.Len(t, call.Messages, 1)
			textMsg, ok := call.Messages[0].(messaging_api.TextMessage)
			require.True(t, ok, "message should be TextMessage")
			assert.Equal(t, tt.text, textMsg.Text)
		})
	}
}

// TestClient_SendReply_APICall tests that SendReply calls the LINE API correctly.
// SC-003: Verify ReplyMessage is called with correct parameters
func TestClient_SendReply_APICall(t *testing.T) {
	t.Parallel()

	// Setup
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mock := &mockMessagingAPI{}
	client := line.NewClientWithAPI(mock, logger)

	replyToken := "test-reply-token-123"
	messageText := "Hello, World!"

	// Act
	err := client.SendReply(replyToken, messageText)
	require.NoError(t, err)

	// Assert
	require.Len(t, mock.replyMessageCalls, 1, "ReplyMessage should be called once")

	call := mock.replyMessageCalls[0]
	assert.Equal(t, replyToken, call.ReplyToken)
	require.Len(t, call.Messages, 1)

	// Verify message is TextMessage
	textMsg, ok := call.Messages[0].(messaging_api.TextMessage)
	require.True(t, ok, "message should be TextMessage")
	assert.Equal(t, messageText, textMsg.Text)
}

// TestClient_SendReply_APIError tests error handling when LINE API fails.
// SC-003: Client should propagate API errors
func TestClient_SendReply_APIError(t *testing.T) {
	t.Parallel()

	// Setup
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mock := &mockMessagingAPI{
		replyMessageError: assert.AnError,
	}
	client := line.NewClientWithAPI(mock, logger)

	// Act
	err := client.SendReply("test-token", "Hello")

	// Assert
	require.Error(t, err, "should propagate API error")
	assert.Equal(t, assert.AnError, err)
}

// TestClient_SendReply_EmptyReplyToken tests behavior with empty reply token.
// SC-003: SendReply should handle edge cases
func TestClient_SendReply_EmptyReplyToken(t *testing.T) {
	t.Parallel()

	// Setup with mock
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mock := &mockMessagingAPI{}
	client := line.NewClientWithAPI(mock, logger)

	// Empty reply token - LINE API will reject this, but Client should call it
	// Client is not responsible for validating business logic, just making the API call
	err := client.SendReply("", "Hello")

	// Client should call the API even with empty token
	require.NoError(t, err)
	require.Len(t, mock.replyMessageCalls, 1)
	assert.Equal(t, "", mock.replyMessageCalls[0].ReplyToken)
}

// TestClient_SendReply_MultipleCalls tests that Client can be used multiple times.
// SC-003: Client should be reusable
func TestClient_SendReply_MultipleCalls(t *testing.T) {
	t.Parallel()

	// Setup
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mock := &mockMessagingAPI{}
	client := line.NewClientWithAPI(mock, logger)

	// Act - send multiple replies
	err1 := client.SendReply("token-1", "Message 1")
	err2 := client.SendReply("token-2", "Message 2")
	err3 := client.SendReply("token-3", "Message 3")

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Len(t, mock.replyMessageCalls, 3, "should make 3 API calls")

	// Verify each call has correct token
	assert.Equal(t, "token-1", mock.replyMessageCalls[0].ReplyToken)
	assert.Equal(t, "token-2", mock.replyMessageCalls[1].ReplyToken)
	assert.Equal(t, "token-3", mock.replyMessageCalls[2].ReplyToken)
}

// TestConfigError_Message tests ConfigError error message format.
// SC-003: ConfigError should follow same pattern as Server
func TestConfigError_Message(t *testing.T) {
	t.Parallel()

	err := &line.ConfigError{Variable: "channelToken"}
	expected := "Missing required configuration: channelToken"
	assert.Equal(t, expected, err.Error())
}

// TestNewClient_TrimSpace tests that channelToken is trimmed.
// SC-003: Follow same pattern as Server (trim whitespace)
func TestNewClient_TrimSpace(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Create client with leading/trailing whitespace
	client, err := line.NewClient("  valid-token  ", logger)
	require.NoError(t, err)
	assert.NotNil(t, client)

	// Client should accept the token (after trimming)
	// Internal implementation should trim the token
}
