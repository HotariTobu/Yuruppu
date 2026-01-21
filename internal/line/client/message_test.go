package client_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"yuruppu/internal/line/client"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SendFlexReply Tests
// =============================================================================

// TestSendFlexReply_Success tests successful flex message reply sending
func TestSendFlexReply_Success(t *testing.T) {
	t.Parallel()

	// AC-001: Send Flex Message reply using the LINE Messaging API
	t.Run("sends flex message with valid JSON", func(t *testing.T) {
		t.Parallel()

		// Given: A valid flex message JSON
		flexJSON := []byte(`{
			"type": "bubble",
			"body": {
				"type": "box",
				"layout": "vertical",
				"contents": [
					{
						"type": "text",
						"text": "Test Event"
					}
				]
			}
		}`)

		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Test Alt Text", flexJSON)

		// Then: No error is returned
		require.NoError(t, err)

		// And: ReplyMessage API was called once
		assert.Equal(t, 1, mockAPI.replyCallCount)

		// And: Request contains correct reply token
		assert.Equal(t, "test-reply-token", mockAPI.lastReplyRequest.ReplyToken)

		// And: Request contains one message
		require.Len(t, mockAPI.lastReplyRequest.Messages, 1)

		// And: Message is a FlexMessage with correct alt text
		flexMsg, ok := mockAPI.lastReplyRequest.Messages[0].(messaging_api.FlexMessage)
		require.True(t, ok, "message should be a FlexMessage")
		assert.Equal(t, "Test Alt Text", flexMsg.AltText)

		// And: Contents are properly unmarshaled
		require.NotNil(t, flexMsg.Contents)
	})

	t.Run("sends flex message with carousel container", func(t *testing.T) {
		t.Parallel()

		// Given: A valid carousel flex message JSON
		flexJSON := []byte(`{
			"type": "carousel",
			"contents": [
				{
					"type": "bubble",
					"body": {
						"type": "box",
						"layout": "vertical",
						"contents": [
							{
								"type": "text",
								"text": "Event 1"
							}
						]
					}
				},
				{
					"type": "bubble",
					"body": {
						"type": "box",
						"layout": "vertical",
						"contents": [
							{
								"type": "text",
								"text": "Event 2"
							}
						]
					}
				}
			]
		}`)

		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Event List", flexJSON)

		// Then: No error is returned
		require.NoError(t, err)

		// And: Request was sent successfully
		assert.Equal(t, 1, mockAPI.replyCallCount)
		flexMsg, ok := mockAPI.lastReplyRequest.Messages[0].(messaging_api.FlexMessage)
		require.True(t, ok)
		assert.Equal(t, "Event List", flexMsg.AltText)
	})

	t.Run("logs debug info with x-line-request-id on success", func(t *testing.T) {
		t.Parallel()

		// Given: A mock API that returns x-line-request-id
		flexJSON := []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`)
		mockAPI := &mockMessagingAPI{
			replyResponse:  &messaging_api.ReplyMessageResponse{},
			xLineRequestID: "test-request-id-12345",
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", flexJSON)

		// Then: No error is returned
		require.NoError(t, err)

		// Note: Actual logging is tested through integration tests
		// This test verifies the flow completes successfully with request ID
	})
}

// TestSendFlexReply_ErrorHandling tests error cases
func TestSendFlexReply_ErrorHandling(t *testing.T) {
	t.Parallel()

	// AC-002: Return error when API call fails
	t.Run("returns error when LINE API call fails", func(t *testing.T) {
		t.Parallel()

		// Given: A mock API that returns an error
		flexJSON := []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`)
		apiErr := errors.New("LINE API rate limit exceeded")
		mockAPI := &mockMessagingAPI{
			replyErr:       apiErr,
			xLineRequestID: "error-request-id",
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", flexJSON)

		// Then: Error is returned
		require.Error(t, err)

		// And: Error contains x-line-request-id
		assert.Contains(t, err.Error(), "error-request-id")

		// And: Error contains context
		assert.Contains(t, err.Error(), "LINE API reply failed")

		// And: Original error is preserved
		assert.True(t, errors.Is(err, apiErr))
	})

	t.Run("returns error when flex JSON is invalid", func(t *testing.T) {
		t.Parallel()

		// Given: Invalid JSON bytes
		invalidJSON := []byte(`{invalid json}`)
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", invalidJSON)

		// Then: Error is returned
		require.Error(t, err)

		// And: Error indicates JSON unmarshaling failure
		assert.Contains(t, err.Error(), "failed to unmarshal flex container")
	})

	t.Run("passes through unknown type to LINE API", func(t *testing.T) {
		t.Parallel()

		// Given: JSON with unknown type field
		// Note: UnmarshalFlexContainer does not validate type; LINE API will reject it
		unknownTypeJSON := []byte(`{
			"type": "invalid_type",
			"body": {}
		}`)
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", unknownTypeJSON)

		// Then: No error from client (SDK doesn't validate type locally)
		// LINE API would reject this, but that's an API error, not a client error
		require.NoError(t, err)
		assert.Equal(t, 1, mockAPI.replyCallCount)
	})

	t.Run("includes x-line-request-id in error even when API fails", func(t *testing.T) {
		t.Parallel()

		// Given: API returns error but response headers are available
		flexJSON := []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`)
		mockAPI := &mockMessagingAPI{
			replyErr:       errors.New("authentication failed"),
			xLineRequestID: "failed-request-id-999",
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", flexJSON)

		// Then: Error contains the request ID for debugging
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed-request-id-999")
	})
}

// TestSendFlexReply_EdgeCases tests boundary conditions and edge cases
func TestSendFlexReply_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles empty alt text", func(t *testing.T) {
		t.Parallel()

		// Given: Empty alt text (edge case)
		flexJSON := []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`)
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called with empty alt text
		err := c.SendFlexReply("test-reply-token", "", flexJSON)

		// Then: Request is sent (LINE API will validate)
		require.NoError(t, err)
		flexMsg := mockAPI.lastReplyRequest.Messages[0].(messaging_api.FlexMessage)
		assert.Equal(t, "", flexMsg.AltText)
	})

	t.Run("handles empty flex JSON", func(t *testing.T) {
		t.Parallel()

		// Given: Empty JSON bytes
		emptyJSON := []byte(``)
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", emptyJSON)

		// Then: Error is returned (cannot unmarshal empty JSON)
		require.Error(t, err)
	})

	t.Run("handles nil flex JSON", func(t *testing.T) {
		t.Parallel()

		// Given: Nil JSON bytes
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", "Alt Text", nil)

		// Then: Error is returned
		require.Error(t, err)
	})

	t.Run("handles very large flex JSON", func(t *testing.T) {
		t.Parallel()

		// Given: A large carousel with many bubbles
		var bubbles []map[string]any
		for i := range 10 {
			bubbles = append(bubbles, map[string]any{
				"type": "bubble",
				"body": map[string]any{
					"type":   "box",
					"layout": "vertical",
					"contents": []map[string]any{
						{
							"type": "text",
							"text": fmt.Sprintf("Event %d", i+1),
						},
					},
				},
			})
		}
		carousel := map[string]any{
			"type":     "carousel",
			"contents": bubbles,
		}
		largeJSON, err := json.Marshal(carousel)
		require.NoError(t, err)

		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		// When: SendFlexReply is called
		err = c.SendFlexReply("test-reply-token", "Large Event List", largeJSON)

		// Then: No error is returned
		require.NoError(t, err)
		assert.Equal(t, 1, mockAPI.replyCallCount)
	})

	t.Run("handles special characters in alt text", func(t *testing.T) {
		t.Parallel()

		// Given: Alt text with special characters
		flexJSON := []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`)
		mockAPI := &mockMessagingAPI{
			replyResponse: &messaging_api.ReplyMessageResponse{},
		}
		logger := slog.New(slog.DiscardHandler)
		c := newTestClient(mockAPI, logger)

		specialAltText := "イベント一覧 <Test> & \"Special\" 'Chars' 😊"

		// When: SendFlexReply is called
		err := c.SendFlexReply("test-reply-token", specialAltText, flexJSON)

		// Then: No error is returned
		require.NoError(t, err)
		flexMsg := mockAPI.lastReplyRequest.Messages[0].(messaging_api.FlexMessage)
		assert.Equal(t, specialAltText, flexMsg.AltText)
	})
}

// TestSendFlexReply_TableDriven provides comprehensive test coverage with table-driven tests
func TestSendFlexReply_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		replyToken  string
		altText     string
		flexJSON    []byte
		apiErr      error
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid bubble message",
			replyToken: "token-123",
			altText:    "Test Message",
			flexJSON:   []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`),
			apiErr:     nil,
			wantErr:    false,
		},
		{
			name:       "valid carousel message",
			replyToken: "token-456",
			altText:    "Event List",
			flexJSON:   []byte(`{"type": "carousel", "contents": [{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}]}`),
			apiErr:     nil,
			wantErr:    false,
		},
		{
			name:        "API error",
			replyToken:  "token-789",
			altText:     "Test",
			flexJSON:    []byte(`{"type": "bubble", "body": {"type": "box", "layout": "vertical", "contents": []}}`),
			apiErr:      errors.New("network timeout"),
			wantErr:     true,
			errContains: "LINE API reply failed",
		},
		{
			name:        "invalid JSON syntax",
			replyToken:  "token-999",
			altText:     "Test",
			flexJSON:    []byte(`{invalid`),
			wantErr:     true,
			errContains: "failed to unmarshal flex container",
		},
		{
			name:        "nil JSON",
			replyToken:  "token-nil",
			altText:     "Test",
			flexJSON:    nil,
			wantErr:     true,
			errContains: "unexpected end of JSON input",
		},
		{
			name:        "empty JSON",
			replyToken:  "token-empty",
			altText:     "Test",
			flexJSON:    []byte(``),
			wantErr:     true,
			errContains: "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockAPI := &mockMessagingAPI{
				replyResponse:  &messaging_api.ReplyMessageResponse{},
				replyErr:       tt.apiErr,
				xLineRequestID: "test-request-id",
			}
			logger := slog.New(slog.DiscardHandler)
			c := newTestClient(mockAPI, logger)

			err := c.SendFlexReply(tt.replyToken, tt.altText, tt.flexJSON)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.replyToken, mockAPI.lastReplyRequest.ReplyToken)
			}
		})
	}
}

// =============================================================================
// Mock Implementation
// =============================================================================

// mockMessagingAPI mocks the LINE Messaging API for testing
type mockMessagingAPI struct {
	// ReplyMessage tracking
	replyCallCount   int
	lastReplyRequest *messaging_api.ReplyMessageRequest
	replyResponse    *messaging_api.ReplyMessageResponse
	replyErr         error
	xLineRequestID   string
}

// ReplyMessageWithHttpInfo mocks the LINE SDK's ReplyMessageWithHttpInfo method
func (m *mockMessagingAPI) ReplyMessageWithHttpInfo(request *messaging_api.ReplyMessageRequest) (*http.Response, *messaging_api.ReplyMessageResponse, error) {
	m.replyCallCount++
	m.lastReplyRequest = request

	// Create mock HTTP response with headers
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       http.NoBody,
	}

	if m.xLineRequestID != "" {
		resp.Header.Set("X-Line-Request-Id", m.xLineRequestID)
	}

	if m.replyErr != nil {
		return resp, nil, m.replyErr
	}

	return resp, m.replyResponse, nil
}

// ShowLoadingAnimation mocks the LINE SDK's ShowLoadingAnimation method
func (m *mockMessagingAPI) ShowLoadingAnimation(request *messaging_api.ShowLoadingAnimationRequest) (*map[string]any, error) {
	// Not used in current tests, but required by messagingAPI interface
	result := make(map[string]any)
	return &result, nil
}

// GetProfile mocks the LINE SDK's GetProfile method
func (m *mockMessagingAPI) GetProfile(userId string) (*messaging_api.UserProfileResponse, error) {
	// Not used in current tests, but required by messagingAPI interface
	return &messaging_api.UserProfileResponse{}, nil
}

// GetGroupSummary mocks the LINE SDK's GetGroupSummary method
func (m *mockMessagingAPI) GetGroupSummary(groupId string) (*messaging_api.GroupSummaryResponse, error) {
	// Not used in current tests, but required by messagingAPI interface
	return &messaging_api.GroupSummaryResponse{}, nil
}

// GetGroupMemberCount mocks the LINE SDK's GetGroupMemberCount method
func (m *mockMessagingAPI) GetGroupMemberCount(groupId string) (*messaging_api.GroupMemberCountResponse, error) {
	// Not used in current tests, but required by messagingAPI interface
	return &messaging_api.GroupMemberCountResponse{}, nil
}

// =============================================================================
// Test Helpers
// =============================================================================

// newTestClient creates a test client with a mock API for testing.
func newTestClient(mockAPI *mockMessagingAPI, logger *slog.Logger) *client.Client {
	return client.NewClientForTest(mockAPI, nil, logger)
}
