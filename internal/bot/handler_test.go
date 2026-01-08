package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"yuruppu/internal/agent"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	lineclient "yuruppu/internal/line/client"
	lineserver "yuruppu/internal/line/server"
	"yuruppu/internal/profile"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks
var (
	_ agent.Agent        = (*mockAgent)(nil)
	_ lineserver.Handler = (*bot.Handler)(nil)
)

// =============================================================================
// NewHandler Tests
// =============================================================================

func TestNewHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		mockAg := &mockAgent{}
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)

		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)

		require.NoError(t, err)
		require.NotNil(t, h)
	})
}

func TestNewHandler_NilDependencies(t *testing.T) {
	validConfig := bot.HandlerConfig{
		TypingIndicatorDelay:   3 * time.Second,
		TypingIndicatorTimeout: 30 * time.Second,
	}

	t.Run("returns error when lineClient is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(nil, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "lineClient is required")
	})

	t.Run("returns error when profileService is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, nil, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "profileService is required")
	})

	t.Run("returns error when historySvc is nil", func(t *testing.T) {
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, nil, &mockMediaService{}, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "historySvc is required")
	})

	t.Run("returns error when mediaSvc is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, nil, &mockAgent{}, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "mediaSvc is required")
	})

	t.Run("returns error when agent is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, nil, validConfig, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "agent is required")
	})

	t.Run("returns error when logger is nil", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validConfig, nil)

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "logger is required")
	})
}

// =============================================================================
// HandlerConfig Validation Tests (AC-007, FR-005)
// =============================================================================

func TestNewHandler_ConfigValidation(t *testing.T) {
	// AC-007: Application should error on startup if timeout is outside valid range (5-60s)

	t.Run("returns error when TypingIndicatorTimeout is below minimum (FR-005)", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   3 * time.Second,
			TypingIndicatorTimeout: 4 * time.Second, // Below minimum of 5s
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "TypingIndicatorTimeout must be between 5s and 60s")
	})

	t.Run("returns error when TypingIndicatorTimeout is above maximum (FR-005)", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   3 * time.Second,
			TypingIndicatorTimeout: 61 * time.Second, // Above maximum of 60s
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "TypingIndicatorTimeout must be between 5s and 60s")
	})

	t.Run("returns error when TypingIndicatorTimeout is zero", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   3 * time.Second,
			TypingIndicatorTimeout: 0, // Zero is below minimum
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "TypingIndicatorTimeout must be between 5s and 60s")
	})

	t.Run("returns error when TypingIndicatorDelay is negative", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   -1 * time.Second, // Negative delay
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.Error(t, err)
		assert.Nil(t, h)
		assert.Contains(t, err.Error(), "TypingIndicatorDelay must be non-negative")
	})

	t.Run("accepts minimum valid timeout (5s)", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   3 * time.Second,
			TypingIndicatorTimeout: 5 * time.Second, // Exact minimum
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		assert.NotNil(t, h)
	})

	t.Run("accepts maximum valid timeout (60s)", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   3 * time.Second,
			TypingIndicatorTimeout: 60 * time.Second, // Exact maximum
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		assert.NotNil(t, h)
	})

	t.Run("accepts zero delay (immediate indicator)", func(t *testing.T) {
		historyRepo, err := history.NewService(&mockStorage{})
		require.NoError(t, err)
		config := bot.HandlerConfig{
			TypingIndicatorDelay:   0, // Zero delay is valid
			TypingIndicatorTimeout: 30 * time.Second,
		}
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, config, slog.New(slog.DiscardHandler))

		require.NoError(t, err)
		assert.NotNil(t, h)
	})
}

// =============================================================================
// Handle* Method Tests
// =============================================================================

// withLineContext creates a context with LINE-specific values
func withLineContext(ctx context.Context, replyToken, sourceID, userID string) context.Context {
	ctx = line.WithReplyToken(ctx, replyToken)
	ctx = line.WithSourceID(ctx, sourceID)
	ctx = line.WithUserID(ctx, userID)
	return ctx
}

// validHandlerConfig returns a valid HandlerConfig for tests
func validHandlerConfig() bot.HandlerConfig {
	return bot.HandlerConfig{
		TypingIndicatorDelay:   3 * time.Second,
		TypingIndicatorTimeout: 30 * time.Second,
	}
}

func TestHandler_HandleText(t *testing.T) {
	t.Run("success - generates response", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.NoError(t, err)
	})

	t.Run("agent error - returns error", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "LLM failed")
	})
}

func TestHandler_HandleSticker(t *testing.T) {
	t.Run("converts sticker to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice sticker!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleSticker(ctx, "pkg-1", "stk-2")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a sticker]", mockAg.lastUserMessageText)
	})
}

func TestHandler_HandleVideo(t *testing.T) {
	t.Run("converts video to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I see a video!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleVideo(ctx, "msg-789")

		require.NoError(t, err)
		assert.Equal(t, "[User sent a video]", mockAg.lastUserMessageText)
	})
}

func TestHandler_HandleAudio(t *testing.T) {
	t.Run("converts audio to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I hear audio!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleAudio(ctx, "msg-101")

		require.NoError(t, err)
		assert.Equal(t, "[User sent an audio]", mockAg.lastUserMessageText)
	})
}

func TestHandler_HandleLocation(t *testing.T) {
	t.Run("converts location to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Nice place!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleLocation(ctx, 35.6762, 139.6503)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a location]", mockAg.lastUserMessageText)
	})
}

func TestHandler_HandleUnknown(t *testing.T) {
	t.Run("converts unknown message to text placeholder", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "I got your message!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleUnknown(ctx)

		require.NoError(t, err)
		assert.Equal(t, "[User sent a message]", mockAg.lastUserMessageText)
	})
}

// =============================================================================
// HandleFollow Tests
// =============================================================================

func TestHandler_HandleFollow(t *testing.T) {
	t.Run("fetches profile from LINE and stores it", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profile: &lineclient.UserProfile{
				DisplayName:   "Alice",
				PictureURL:    "",
				StatusMessage: "Hello!",
			},
		}
		mockPS := &mockProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, mockPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.NoError(t, err)
		assert.Equal(t, "user-123", mockPS.lastUserID)
		require.NotNil(t, mockPS.profile)
		assert.Equal(t, "Alice", mockPS.profile.DisplayName)
		assert.Equal(t, "Hello!", mockPS.profile.StatusMessage)
	})

	t.Run("returns error when userID not in context", func(t *testing.T) {
		mockStore := newMockStorage()
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := t.Context() // No userID in context
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "userID not found")
	})

	t.Run("returns error when GetProfile fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profileErr: errors.New("LINE API error"),
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch profile")
	})

	t.Run("returns error when SetUserProfile fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			profile: &lineclient.UserProfile{DisplayName: "Alice"},
		}
		mockPS := &mockProfileService{
			setErr: errors.New("storage error"),
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, mockPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "", "", "user-123")
		err = h.HandleFollow(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to store profile")
	})
}

// =============================================================================
// History Integration Tests
// =============================================================================

func TestHandler_HistoryIntegration(t *testing.T) {
	t.Run("saves user message to history", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.NoError(t, err)
		// Verify storage was called once (user message only, assistant message is saved by reply tool)
		require.Equal(t, 1, mockStore.writeCallCount)
	})

	t.Run("does not respond when history read fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.readErr = errors.New("GCS read failed")
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read history")
	})

	t.Run("does not respond when user message storage fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockStore.writeResults = []writeResult{{gen: 0, err: errors.New("GCS failed")}}
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write history")
	})

	t.Run("saves only user message when agent fails", func(t *testing.T) {
		mockStore := newMockStorage()
		mockAg := &mockAgent{err: errors.New("LLM failed")}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// User message is saved before agent is called
		assert.Equal(t, 1, mockStore.writeCallCount)
	})
}

// =============================================================================
// Error Chain Tests (errors.Is verification)
// =============================================================================

func TestHandler_ErrorChain(t *testing.T) {
	t.Run("agent error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		agentErr := errors.New("LLM generation failed")
		mockAg := &mockAgent{err: agentErr}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, agentErr), "error chain should contain original agent error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to generate response")
	})

	t.Run("storage read error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS bucket not found")
		mockStore.readErr = storageErr
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to load history")
	})

	t.Run("storage write error is wrapped and preserves original error", func(t *testing.T) {
		mockStore := newMockStorage()
		storageErr := errors.New("GCS write quota exceeded")
		mockStore.writeResults = []writeResult{{gen: 0, err: storageErr}}
		mockAg := &mockAgent{response: "Hello!"}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(&mockLineClient{}, &mockProfileService{}, historyRepo, &mockMediaService{}, mockAg, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withLineContext(t.Context(), "reply-token", "user-123", "user-123")
		err = h.HandleText(ctx, "Hi")

		require.Error(t, err)
		// Verify error chain preserves original error
		assert.True(t, errors.Is(err, storageErr), "error chain should contain original storage error")
		// Verify wrapping context is present
		assert.Contains(t, err.Error(), "failed to save user message to history")
	})
}

// =============================================================================
// Mocks
// =============================================================================

type mockAgent struct {
	response            string
	err                 error
	lastUserMessageText string
	processDelay        time.Duration // Delay to simulate slow processing
}

func (m *mockAgent) Generate(ctx context.Context, hist []agent.Message) (*agent.AssistantMessage, error) {
	// Extract text from last user message in history for testing
	// Parts[0] is the header, Parts[1] is the actual message content
	if len(hist) > 0 {
		if userMsg, ok := hist[len(hist)-1].(*agent.UserMessage); ok && len(userMsg.Parts) > 1 {
			if textPart, ok := userMsg.Parts[1].(*agent.UserTextPart); ok {
				m.lastUserMessageText = textPart.Text
			}
		}
	}
	// Simulate processing delay for testing delayed loading indicator
	if m.processDelay > 0 {
		select {
		case <-time.After(m.processDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return &agent.AssistantMessage{
		Parts: []agent.AssistantPart{&agent.AssistantTextPart{Text: m.response}},
	}, nil
}

func (m *mockAgent) Close(ctx context.Context) error {
	return nil
}

type mockLineClient struct {
	data          []byte
	mimeType      string
	err           error
	lastMessageID string
	profile       *lineclient.UserProfile
	profileErr    error
	// ShowLoadingAnimation tracking
	showLoadingCalled  bool
	showLoadingChatID  string
	showLoadingTimeout time.Duration
	showLoadingDelay   time.Duration // Delay to simulate slow API call
	showLoadingErr     error
}

func (m *mockLineClient) GetMessageContent(messageID string) ([]byte, string, error) {
	m.lastMessageID = messageID
	if m.err != nil {
		return nil, "", m.err
	}
	return m.data, m.mimeType, nil
}

func (m *mockLineClient) GetProfile(ctx context.Context, userID string) (*lineclient.UserProfile, error) {
	if m.profileErr != nil {
		return nil, m.profileErr
	}
	if m.profile != nil {
		return m.profile, nil
	}
	return &lineclient.UserProfile{
		DisplayName:   "Test User",
		PictureURL:    "",
		StatusMessage: "",
	}, nil
}

func (m *mockLineClient) ShowLoadingAnimation(ctx context.Context, chatID string, timeout time.Duration) error {
	m.showLoadingCalled = true
	m.showLoadingChatID = chatID
	m.showLoadingTimeout = timeout

	// Simulate API delay if configured
	if m.showLoadingDelay > 0 {
		select {
		case <-time.After(m.showLoadingDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return m.showLoadingErr
}

type mockProfileService struct {
	profile    *profile.UserProfile
	getErr     error
	setErr     error
	lastUserID string
}

func (m *mockProfileService) GetUserProfile(ctx context.Context, userID string) (*profile.UserProfile, error) {
	m.lastUserID = userID
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.profile != nil {
		return m.profile, nil
	}
	return &profile.UserProfile{
		DisplayName:   "Test User",
		PictureURL:    "",
		StatusMessage: "",
	}, nil
}

func (m *mockProfileService) SetUserProfile(ctx context.Context, userID string, p *profile.UserProfile) error {
	m.lastUserID = userID
	m.profile = p
	return m.setErr
}

// writeResult represents a single Write call result
type writeResult struct {
	gen int64
	err error
}

// writeRecord represents a recorded Write call
type writeRecord struct {
	key      string
	mimeType string
	data     []byte
}

// mockStorage implements storage.Storage interface
type mockStorage struct {
	// Read behavior
	data          map[string][]byte
	generation    map[string]int64
	readErr       error
	readCallCount int

	// Write behavior
	writeResults         []writeResult
	writeCallCount       int
	writes               []writeRecord
	lastWriteKey         string
	lastWriteMIMEType    string
	lastWriteData        []byte
	lastWriteExpectedGen int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:       make(map[string][]byte),
		generation: make(map[string]int64),
	}
}

func (m *mockStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	m.readCallCount++
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	data, exists := m.data[key]
	if !exists {
		return nil, 0, nil
	}
	return data, m.generation[key], nil
}

func (m *mockStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	m.writeCallCount++
	m.writes = append(m.writes, writeRecord{key: key, mimeType: mimetype, data: data})
	m.lastWriteKey = key
	m.lastWriteMIMEType = mimetype
	m.lastWriteData = data
	m.lastWriteExpectedGen = expectedGeneration

	if len(m.writeResults) > 0 {
		r := m.writeResults[0]
		m.writeResults = m.writeResults[1:]
		if r.err != nil {
			return 0, r.err
		}
		m.data[key] = data
		m.generation[key] = r.gen
		return r.gen, nil
	}
	m.data[key] = data
	newGen := expectedGeneration + 1
	m.generation[key] = newGen
	return newGen, nil
}

func (m *mockStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + key, nil
}

func (m *mockStorage) Close(ctx context.Context) error {
	return nil
}

// mockMediaService implements bot.MediaService interface
type mockMediaService struct {
	storeErr     error
	storeKey     string
	lastSourceID string
	lastData     []byte
	lastMIMEType string
}

func (m *mockMediaService) Store(ctx context.Context, sourceID string, data []byte, mimeType string) (string, error) {
	m.lastSourceID = sourceID
	m.lastData = data
	m.lastMIMEType = mimeType
	if m.storeErr != nil {
		return "", m.storeErr
	}
	if m.storeKey != "" {
		return m.storeKey, nil
	}
	return sourceID + "/test-uuid", nil
}

func (m *mockMediaService) GetSignedURL(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	return "https://example.com/signed/" + storageKey, nil
}
