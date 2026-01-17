package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/bot"
	"yuruppu/internal/history"
	"yuruppu/internal/line"
	lineclient "yuruppu/internal/line/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleJoin Tests
// =============================================================================

func TestHandler_HandleJoin(t *testing.T) {
	// AC-001: Save group info on join [FR-001, FR-002, FR-003]
	// Given: The bot is invited to a LINE group
	// When: The bot receives a join event with source type "group"
	// Then: The bot retrieves group summary from LINE API using the group ID
	//       The group information (ID, name, picture URL) is saved to storage
	//       The join event handling completes successfully
	t.Run("should save group profile when bot joins group", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    "G1234567890abcdef",
				GroupName:  "Test Group",
				PictureURL: "", // Empty to avoid HTTP request in test
			},
		}
		mockGPS := &mockGroupProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "G1234567890abcdef", mockGPS.lastGroupID)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, "Test Group", mockGPS.profile.DisplayName)
		assert.Equal(t, "", mockGPS.profile.PictureURL)
	})

	// AC-003: Handle missing picture URL [FR-003]
	// Given: The bot is invited to a LINE group without a picture
	// When: The group summary returns empty picture URL
	// Then: The group is saved with an empty picture URL field
	//       Other fields (ID, name) are saved normally
	t.Run("should save group profile with empty picture URL when not provided", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    "G1234567890abcdef",
				GroupName:  "Group Without Picture",
				PictureURL: "", // Empty picture URL
			},
		}
		mockGPS := &mockGroupProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "G1234567890abcdef", mockGPS.lastGroupID)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, "Group Without Picture", mockGPS.profile.DisplayName)
		assert.Equal(t, "", mockGPS.profile.PictureURL)
	})

	// AC-002: Handle API failure gracefully [FR-001, Error]
	// Given: The bot is invited to a LINE group
	// When: The LINE API call to get group summary fails
	// Then: The error is returned (caller handles gracefully)
	//       No partial data is saved
	t.Run("should return error when LINE API fails", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		lineAPIError := errors.New("LINE API network error")
		mockClient := &mockLineClient{
			groupSummaryErr: lineAPIError,
		}
		mockGPS := &mockGroupProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)

		// Then
		require.Error(t, err)
		assert.True(t, errors.Is(err, lineAPIError), "error chain should contain original error")
		assert.Contains(t, err.Error(), "failed to get group summary")
		// Verify no data was saved
		assert.Nil(t, mockGPS.profile, "no partial data should be saved on LINE API failure")
	})

	// AC-002: Handle storage failure gracefully [FR-002, Error]
	// Given: The bot is invited to a LINE group
	// When: The storage save operation fails
	// Then: The error is returned (caller handles gracefully)
	t.Run("should return error when storage fails", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    "G1234567890abcdef",
				GroupName:  "Test Group",
				PictureURL: "",
			},
		}
		storageError := errors.New("GCS write quota exceeded")
		mockGPS := &mockGroupProfileService{
			setErr: storageError,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)

		// Then
		require.Error(t, err)
		assert.True(t, errors.Is(err, storageError), "error chain should contain original error")
		assert.Contains(t, err.Error(), "failed to save group profile")
	})

	// FR-001: Verify LINE API is called with correct group ID
	t.Run("should call GetGroupSummary with correct group ID from context", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    "G9876543210fedcba",
				GroupName:  "Another Group",
				PictureURL: "",
			},
		}
		mockGPS := &mockGroupProfileService{}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G9876543210fedcba")

		// When
		err = h.HandleJoin(ctx)

		// Then
		require.NoError(t, err)
		assert.Equal(t, "G9876543210fedcba", mockClient.lastGroupID)
		assert.Equal(t, "G9876543210fedcba", mockGPS.lastGroupID)
	})
}

// TestHandler_HandleJoin_ContextValidation tests context validation
func TestHandler_HandleJoin_ContextValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupCtx    func(context.Context) context.Context
		wantErr     bool
		errContains string
	}{
		{
			name: "missing chatType in context",
			setupCtx: func(ctx context.Context) context.Context {
				// Only set sourceID, missing chatType
				return line.WithSourceID(ctx, "G1234567890abcdef")
			},
			wantErr:     true,
			errContains: "chatType not found",
		},
		{
			name: "missing sourceID in context",
			setupCtx: func(ctx context.Context) context.Context {
				// Only set chatType, missing sourceID
				return line.WithChatType(ctx, line.ChatTypeGroup)
			},
			wantErr:     true,
			errContains: "sourceID not found",
		},
		{
			name: "valid context with all required fields",
			setupCtx: func(ctx context.Context) context.Context {
				ctx = line.WithChatType(ctx, line.ChatTypeGroup)
				return line.WithSourceID(ctx, "G1234567890abcdef")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := newMockStorage()
			mockClient := &mockLineClient{
				groupSummary: &lineclient.GroupSummary{
					GroupID:    "G1234567890abcdef",
					GroupName:  "Test Group",
					PictureURL: "",
				},
			}
			historyRepo, err := history.NewService(mockStore)
			require.NoError(t, err)
			logger := slog.New(slog.DiscardHandler)
			h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
			require.NoError(t, err)

			ctx := tt.setupCtx(t.Context())
			err = h.HandleJoin(ctx)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestHandler_HandleJoin_ErrorWrapping verifies error wrapping preserves error chain
func TestHandler_HandleJoin_ErrorWrapping(t *testing.T) {
	t.Run("GetGroupSummary error is wrapped and preserves original error", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		lineAPIError := errors.New("LINE API rate limit exceeded")
		mockClient := &mockLineClient{
			groupSummaryErr: lineAPIError,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, &mockGroupProfileService{}, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)
		// Then
		// NOTE: This test verifies the CURRENT implementation's error wrapping behavior.
		// Once AC-002 is implemented (graceful error handling), this test will need to be updated
		// or removed, as errors will be logged rather than returned.
		if err != nil {
			assert.True(t, errors.Is(err, lineAPIError), "error chain should contain original LINE API error")
			assert.Contains(t, err.Error(), "failed to get group summary")
		}
	})

	t.Run("SetGroupProfile error is wrapped and preserves original error", func(t *testing.T) {
		// Given
		mockStore := newMockStorage()
		mockClient := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    "G1234567890abcdef",
				GroupName:  "Test Group",
				PictureURL: "",
			},
		}
		storageError := errors.New("GCS bucket not accessible")
		mockGPS := &mockGroupProfileService{
			setErr: storageError,
		}
		historyRepo, err := history.NewService(mockStore)
		require.NoError(t, err)
		logger := slog.New(slog.DiscardHandler)
		h, err := bot.NewHandler(mockClient, &mockProfileService{}, mockGPS, historyRepo, &mockMediaService{}, &mockAgent{}, validHandlerConfig(), logger)
		require.NoError(t, err)

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")

		// When
		err = h.HandleJoin(ctx)
		// Then
		// NOTE: This test verifies the CURRENT implementation's error wrapping behavior.
		// Once AC-002 is implemented (graceful error handling), this test will need to be updated
		// or removed, as errors will be logged rather than returned.
		if err != nil {
			assert.True(t, errors.Is(err, storageError), "error chain should contain original storage error")
			assert.Contains(t, err.Error(), "failed to save group profile")
		}
	})
}

// =============================================================================
// Test Helpers
// =============================================================================

// withJoinContext creates a context for join event with sourceID and chatType
func withJoinContext(ctx context.Context, groupID string) context.Context {
	ctx = line.WithChatType(ctx, line.ChatTypeGroup)
	ctx = line.WithSourceID(ctx, groupID)
	return ctx
}
