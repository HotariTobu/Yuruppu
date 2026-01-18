package bot_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"yuruppu/internal/bot"
	"yuruppu/internal/groupprofile"
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

// =============================================================================
// HandleJoin Tests - FR-001: Retrieve group member count when bot joins
// =============================================================================

func TestHandleJoin_MemberCount(t *testing.T) {
	// AC-001: Member count retrieved on join [FR-001, FR-004]
	t.Run("should retrieve and save member count when bot joins group", func(t *testing.T) {
		// Given: The bot is invited to a LINE group
		ctx := context.Background()
		groupID := "G-member-count-test"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Engineering Team",
				PictureURL: "",
			},
			groupMemberCount: 10,
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: The bot receives a join event
		err = handler.HandleJoin(ctx)

		// Then: The member count is retrieved from LINE API
		require.NoError(t, err)
		assert.Equal(t, groupID, mockLine.lastGroupID, "GetGroupSummary should be called with correct groupID")

		// Then: The count is saved to storage with the group information
		require.NotNil(t, mockGroupProfile.profile, "Group profile should be saved")
		assert.Equal(t, "Engineering Team", mockGroupProfile.profile.DisplayName)
		assert.Equal(t, 10, mockGroupProfile.profile.UserCount, "Member count should be saved")
		assert.Equal(t, groupID, mockGroupProfile.lastGroupID, "Profile should be saved with correct groupID")
	})

	// AC-006: Handle API failure on join [FR-001]
	t.Run("should use fallback count when GetGroupMemberCount fails", func(t *testing.T) {
		// Given: The bot is invited to a group
		ctx := context.Background()
		groupID := "G-api-error-test"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Sales Team",
				PictureURL: "",
			},
			groupMemberCountErr: errors.New("LINE API error: rate limit exceeded"),
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: The LINE API call to get member count fails
		err = handler.HandleJoin(ctx)

		// Then: The error is logged (no crash)
		require.NoError(t, err, "HandleJoin should complete without error")

		// Then: Join event handling completes without crashing
		// Then: Group is saved without member count (fallback to 1)
		require.NotNil(t, mockGroupProfile.profile, "Group profile should still be saved")
		assert.Equal(t, "Sales Team", mockGroupProfile.profile.DisplayName)
		assert.Equal(t, 1, mockGroupProfile.profile.UserCount, "Should use fallback count of 1 when API fails")
		assert.Equal(t, groupID, mockGroupProfile.lastGroupID)
	})

	// AC-006: Verify error is logged but doesn't prevent join
	t.Run("should log warning but continue when member count API fails", func(t *testing.T) {
		// Given: Bot joins a group where GetGroupMemberCount will fail
		ctx := context.Background()
		groupID := "G-log-warning-test"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Community",
				PictureURL: "",
			},
			groupMemberCountErr: errors.New("network timeout"),
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleJoin is called
		err = handler.HandleJoin(ctx)

		// Then: No error is returned (error is logged internally)
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 1, mockGroupProfile.profile.UserCount, "Fallback value used")
	})
}

func TestHandleJoin_MemberCount_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		groupID          string
		groupName        string
		memberCount      int
		memberCountErr   error
		wantUserCount    int
		wantErr          bool
		wantDisplayName  string
		wantProfileSaved bool
	}{
		{
			name:             "small group with 3 members",
			groupID:          "G-small-001",
			groupName:        "Family Chat",
			memberCount:      3,
			wantUserCount:    3,
			wantDisplayName:  "Family Chat",
			wantProfileSaved: true,
		},
		{
			name:             "medium group with 15 members",
			groupID:          "G-medium-001",
			groupName:        "Project Team",
			memberCount:      15,
			wantUserCount:    15,
			wantDisplayName:  "Project Team",
			wantProfileSaved: true,
		},
		{
			name:             "large group with 100 members",
			groupID:          "G-large-001",
			groupName:        "Company All-Hands",
			memberCount:      100,
			wantUserCount:    100,
			wantDisplayName:  "Company All-Hands",
			wantProfileSaved: true,
		},
		{
			name:             "single member group",
			groupID:          "G-single-001",
			groupName:        "Private Group",
			memberCount:      1,
			wantUserCount:    1,
			wantDisplayName:  "Private Group",
			wantProfileSaved: true,
		},
		{
			name:             "API error uses fallback",
			groupID:          "G-error-001",
			groupName:        "Error Test Group",
			memberCountErr:   errors.New("API timeout"),
			wantUserCount:    1,
			wantDisplayName:  "Error Test Group",
			wantProfileSaved: true,
		},
		{
			name:             "rate limit error uses fallback",
			groupID:          "G-ratelimit-001",
			groupName:        "Rate Limited Group",
			memberCountErr:   errors.New("rate limit exceeded"),
			wantUserCount:    1,
			wantDisplayName:  "Rate Limited Group",
			wantProfileSaved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Bot is invited to a group
			ctx := context.Background()
			ctx = line.WithChatType(ctx, line.ChatTypeGroup)
			ctx = line.WithSourceID(ctx, tt.groupID)

			mockLine := &mockLineClient{
				groupSummary: &lineclient.GroupSummary{
					GroupID:    tt.groupID,
					GroupName:  tt.groupName,
					PictureURL: "",
				},
				groupMemberCount:    tt.memberCount,
				groupMemberCountErr: tt.memberCountErr,
			}
			mockGroupProfile := &mockGroupProfileService{}
			historyRepo, err := history.NewService(newMockStorage())
			require.NoError(t, err)

			handler, err := bot.NewHandler(
				mockLine,
				&mockProfileService{},
				mockGroupProfile,
				historyRepo,
				&mockMediaService{},
				&mockAgent{},
				validHandlerConfig(),
				slog.New(slog.DiscardHandler),
			)
			require.NoError(t, err)

			// When: HandleJoin is called
			err = handler.HandleJoin(ctx)

			// Then: Check error expectation
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Then: Verify profile was saved correctly
			if tt.wantProfileSaved {
				require.NotNil(t, mockGroupProfile.profile, "Profile should be saved")
				assert.Equal(t, tt.wantUserCount, mockGroupProfile.profile.UserCount, "UserCount should match expected")
				assert.Equal(t, tt.wantDisplayName, mockGroupProfile.profile.DisplayName, "DisplayName should match")
				assert.Equal(t, tt.groupID, mockGroupProfile.lastGroupID, "Should save to correct groupID")
			}
		})
	}
}

func TestHandleJoin_MemberCount_EdgeCases(t *testing.T) {
	t.Run("should handle zero member count from API", func(t *testing.T) {
		// Given: API returns 0 members (edge case)
		ctx := context.Background()
		groupID := "G-zero-members"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Empty Group",
				PictureURL: "",
			},
			groupMemberCount: 0,
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleJoin is called
		err = handler.HandleJoin(ctx)

		// Then: Should save 0 without crashing
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 0, mockGroupProfile.profile.UserCount, "Should save 0 if API returns 0")
	})

	t.Run("should preserve member count when both API call and profile save succeed", func(t *testing.T) {
		// Given: Everything works correctly
		ctx := context.Background()
		groupID := "G-success-path"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		expectedCount := 42
		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Success Group",
				PictureURL: "",
			},
			groupMemberCount: expectedCount,
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleJoin is called
		err = handler.HandleJoin(ctx)

		// Then: Member count matches API response exactly
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, expectedCount, mockGroupProfile.profile.UserCount)
	})

	t.Run("should verify GroupProfile struct has UserCount field", func(t *testing.T) {
		// This test ensures GroupProfile struct has UserCount field (compile-time check)
		profile := &groupprofile.GroupProfile{
			DisplayName: "Test",
			PictureURL:  "",
			UserCount:   5,
		}

		// Verify field exists and is accessible
		assert.Equal(t, 5, profile.UserCount, "GroupProfile should have UserCount field")
	})
}

func TestHandleJoin_MemberCount_Integration(t *testing.T) {
	// Integration test: Verify entire flow from join to storage
	t.Run("should complete full join flow with member count", func(t *testing.T) {
		// Given: Complete setup with all dependencies
		ctx := context.Background()
		groupID := "G-integration-test"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		mockLine := &mockLineClient{
			groupSummary: &lineclient.GroupSummary{
				GroupID:    groupID,
				GroupName:  "Integration Test Group",
				PictureURL: "",
			},
			groupMemberCount: 25,
		}
		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			mockLine,
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Full join flow executes
		err = handler.HandleJoin(ctx)

		// Then: Verify all steps completed successfully
		require.NoError(t, err)

		// Verify GetGroupSummary was called
		assert.Equal(t, groupID, mockLine.lastGroupID)

		// Verify profile was saved with complete data
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, groupID, mockGroupProfile.lastGroupID)
		assert.Equal(t, "Integration Test Group", mockGroupProfile.profile.DisplayName)
		assert.Equal(t, 25, mockGroupProfile.profile.UserCount)
	})
}
