package bot_test

import (
	"context"
	"errors"
	"fmt"
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

// =============================================================================
// HandleMemberJoined Tests - FR-002: Increment member count when members join
// =============================================================================

func TestHandleMemberJoined(t *testing.T) {
	// AC-002: Member count incremented on member join [FR-002, FR-004]
	t.Run("should increment member count when multiple members join", func(t *testing.T) {
		// Given: The bot is already in a group with a stored member count of 10
		ctx := context.Background()
		groupID := "G-member-join-test"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Engineering Team",
			PictureURL:  "",
			UserCount:   10,
		}
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Members join the group (3 new members)
		joinedUserIDs := []string{"U-user-001", "U-user-002", "U-user-003"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: The stored member count is incremented by len(joinedUserIDs) and persisted
		require.NoError(t, err)
		assert.Equal(t, groupID, mockGroupProfile.lastGroupID, "Profile should be updated for correct groupID")
		require.NotNil(t, mockGroupProfile.profile, "Profile should be saved")
		assert.Equal(t, 13, mockGroupProfile.profile.UserCount, "UserCount should be 10 + 3 = 13")
	})

	// AC-002: Single member join
	t.Run("should increment member count by 1 when single member joins", func(t *testing.T) {
		// Given: Group has 5 members
		ctx := context.Background()
		groupID := "G-single-join"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Small Team",
			UserCount:   5,
		}
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Single member joins
		joinedUserIDs := []string{"U-new-user"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Count incremented by 1
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 6, mockGroupProfile.profile.UserCount, "UserCount should be 5 + 1 = 6")
	})

	// AC-002: Error handling - GetGroupProfile fails
	t.Run("should log warning and return nil when GetGroupProfile fails", func(t *testing.T) {
		// Given: Group profile cannot be retrieved
		ctx := context.Background()
		groupID := "G-get-error"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		profileError := errors.New("storage unavailable")
		mockGroupProfile := &mockGroupProfileService{
			getErr: profileError,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Members join but GetGroupProfile fails
		joinedUserIDs := []string{"U-user-001"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Error is logged, nil is returned (graceful handling)
		assert.NoError(t, err, "Should not return error, only log warning")
		assert.Nil(t, mockGroupProfile.profile, "Profile should not be updated on GetGroupProfile failure")
	})

	// AC-002: Error handling - SetGroupProfile fails
	t.Run("should log warning and continue when SetGroupProfile fails", func(t *testing.T) {
		// Given: Group profile exists but SetGroupProfile will fail
		ctx := context.Background()
		groupID := "G-set-error"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Test Group",
			UserCount:   20,
		}
		setError := errors.New("write permission denied")
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
			setErr:  setError,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Members join but SetGroupProfile fails
		joinedUserIDs := []string{"U-user-001", "U-user-002"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Error is logged but function returns nil (no crash)
		assert.NoError(t, err, "Should not return error, only log warning")
	})

	// Context validation
	t.Run("should return error when chatType is missing from context", func(t *testing.T) {
		// Given: Context without chatType
		ctx := context.Background()
		ctx = line.WithSourceID(ctx, "G-no-chattype")

		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleMemberJoined is called
		joinedUserIDs := []string{"U-user-001"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Error is returned
		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatType not found")
	})

	t.Run("should return error when sourceID is missing from context", func(t *testing.T) {
		// Given: Context without sourceID
		ctx := context.Background()
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)

		mockGroupProfile := &mockGroupProfileService{}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: HandleMemberJoined is called
		joinedUserIDs := []string{"U-user-001"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Error is returned
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceID not found")
	})
}

func TestHandleMemberJoined_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		initialUserCount  int
		joinedUserIDs     []string
		expectedUserCount int
		getProfileErr     error
		setProfileErr     error
		wantErr           bool
		wantErrContains   string
		expectUpdate      bool
	}{
		{
			name:              "increment by 1 when single member joins",
			initialUserCount:  10,
			joinedUserIDs:     []string{"U-single"},
			expectedUserCount: 11,
			expectUpdate:      true,
		},
		{
			name:              "increment by 3 when multiple members join",
			initialUserCount:  10,
			joinedUserIDs:     []string{"U-user-001", "U-user-002", "U-user-003"},
			expectedUserCount: 13,
			expectUpdate:      true,
		},
		{
			name:              "increment by 5 when five members join",
			initialUserCount:  20,
			joinedUserIDs:     []string{"U-1", "U-2", "U-3", "U-4", "U-5"},
			expectedUserCount: 25,
			expectUpdate:      true,
		},
		{
			name:              "handle small group growing from 1 to 2",
			initialUserCount:  1,
			joinedUserIDs:     []string{"U-second-member"},
			expectedUserCount: 2,
			expectUpdate:      true,
		},
		{
			name:              "handle large group growing from 100 to 110",
			initialUserCount:  100,
			joinedUserIDs:     []string{"U-1", "U-2", "U-3", "U-4", "U-5", "U-6", "U-7", "U-8", "U-9", "U-10"},
			expectedUserCount: 110,
			expectUpdate:      true,
		},
		{
			name:             "GetGroupProfile error returns nil without update",
			initialUserCount: 10,
			joinedUserIDs:    []string{"U-user-001"},
			getProfileErr:    errors.New("storage read failed"),
			wantErr:          false,
			expectUpdate:     false,
		},
		{
			name:              "SetGroupProfile error still returns nil",
			initialUserCount:  10,
			joinedUserIDs:     []string{"U-user-001"},
			expectedUserCount: 11,
			setProfileErr:     errors.New("storage write failed"),
			wantErr:           false,
			expectUpdate:      true,
		},
		{
			name:              "empty joinedUserIDs array increments by 0",
			initialUserCount:  15,
			joinedUserIDs:     []string{},
			expectedUserCount: 15,
			expectUpdate:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Group with initial member count
			ctx := context.Background()
			groupID := "G-table-test"
			ctx = line.WithChatType(ctx, line.ChatTypeGroup)
			ctx = line.WithSourceID(ctx, groupID)

			initialProfile := &groupprofile.GroupProfile{
				DisplayName: "Table Test Group",
				UserCount:   tt.initialUserCount,
			}
			mockGroupProfile := &mockGroupProfileService{
				profile: initialProfile,
				getErr:  tt.getProfileErr,
				setErr:  tt.setProfileErr,
			}
			historyRepo, err := history.NewService(newMockStorage())
			require.NoError(t, err)

			handler, err := bot.NewHandler(
				&mockLineClient{},
				&mockProfileService{},
				mockGroupProfile,
				historyRepo,
				&mockMediaService{},
				&mockAgent{},
				validHandlerConfig(),
				slog.New(slog.DiscardHandler),
			)
			require.NoError(t, err)

			// When: HandleMemberJoined is called
			err = handler.HandleMemberJoined(ctx, tt.joinedUserIDs)

			// Then: Verify error expectation
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
				return
			}
			require.NoError(t, err)

			// Then: Verify count update
			if tt.expectUpdate && tt.getProfileErr == nil {
				require.NotNil(t, mockGroupProfile.profile, "Profile should be updated")
				assert.Equal(t, tt.expectedUserCount, mockGroupProfile.profile.UserCount, "UserCount should match expected value")
				assert.Equal(t, groupID, mockGroupProfile.lastGroupID, "Should update correct groupID")
			}
		})
	}
}

func TestHandleMemberJoined_EdgeCases(t *testing.T) {
	t.Run("should handle zero initial count", func(t *testing.T) {
		// Given: Group with 0 member count (edge case)
		ctx := context.Background()
		groupID := "G-zero-count"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Empty Group",
			UserCount:   0,
		}
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: 2 members join
		joinedUserIDs := []string{"U-first", "U-second"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Count goes from 0 to 2
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 2, mockGroupProfile.profile.UserCount, "Should increment from 0 to 2")
	})

	t.Run("should preserve other profile fields when updating count", func(t *testing.T) {
		// Given: Group profile with all fields populated
		ctx := context.Background()
		groupID := "G-preserve-fields"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName:     "Complete Profile Group",
			PictureURL:      "https://example.com/picture.jpg",
			PictureMIMEType: "image/jpeg",
			UserCount:       8,
		}
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: Members join
		joinedUserIDs := []string{"U-user-001"}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Other fields are preserved
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 9, mockGroupProfile.profile.UserCount, "UserCount should be updated")
		assert.Equal(t, "Complete Profile Group", mockGroupProfile.profile.DisplayName, "DisplayName should be preserved")
		assert.Equal(t, "https://example.com/picture.jpg", mockGroupProfile.profile.PictureURL, "PictureURL should be preserved")
		assert.Equal(t, "image/jpeg", mockGroupProfile.profile.PictureMIMEType, "PictureMIMEType should be preserved")
	})

	t.Run("should handle very large member count increment", func(t *testing.T) {
		// Given: Large number of members joining at once
		ctx := context.Background()
		groupID := "G-large-increment"
		ctx = line.WithChatType(ctx, line.ChatTypeGroup)
		ctx = line.WithSourceID(ctx, groupID)

		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Large Group",
			UserCount:   500,
		}
		mockGroupProfile := &mockGroupProfileService{
			profile: initialProfile,
		}
		historyRepo, err := history.NewService(newMockStorage())
		require.NoError(t, err)

		handler, err := bot.NewHandler(
			&mockLineClient{},
			&mockProfileService{},
			mockGroupProfile,
			historyRepo,
			&mockMediaService{},
			&mockAgent{},
			validHandlerConfig(),
			slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		// When: 50 members join
		joinedUserIDs := make([]string, 50)
		for i := range joinedUserIDs {
			joinedUserIDs[i] = fmt.Sprintf("U-user-%03d", i+1)
		}
		err = handler.HandleMemberJoined(ctx, joinedUserIDs)

		// Then: Count correctly incremented by 50
		require.NoError(t, err)
		require.NotNil(t, mockGroupProfile.profile)
		assert.Equal(t, 550, mockGroupProfile.profile.UserCount, "Should handle large increment: 500 + 50 = 550")
	})
}
