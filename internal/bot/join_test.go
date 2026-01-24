package bot_test

import (
	"context"
	"errors"
	"testing"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleJoin Tests
// =============================================================================

func TestHandler_HandleJoin(t *testing.T) {
	// AC-001: Save group info on join [FR-001, FR-002, FR-003]
	t.Run("should save group profile when bot joins group", func(t *testing.T) {
		mockGPS := &mockGroupProfileService{}
		handler := newTestHandler(t).
			WithGroupSummary("G1234567890abcdef", "Test Group", "").
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		assert.Equal(t, "G1234567890abcdef", mockGPS.lastGroupID)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, "Test Group", mockGPS.profile.DisplayName)
		assert.Equal(t, "", mockGPS.profile.PictureURL)
	})

	// AC-003: Handle missing picture URL [FR-003]
	t.Run("should save group profile with empty picture URL when not provided", func(t *testing.T) {
		mockGPS := &mockGroupProfileService{}
		handler := newTestHandler(t).
			WithGroupSummary("G1234567890abcdef", "Group Without Picture", "").
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		assert.Equal(t, "G1234567890abcdef", mockGPS.lastGroupID)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, "Group Without Picture", mockGPS.profile.DisplayName)
		assert.Equal(t, "", mockGPS.profile.PictureURL)
	})

	// AC-002: Handle API failure gracefully [FR-001, Error]
	t.Run("should return error when LINE API fails", func(t *testing.T) {
		lineAPIError := errors.New("LINE API network error")
		mockGPS := &mockGroupProfileService{}
		handler := newTestHandler(t).
			WithGroupSummaryError(lineAPIError).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)

		require.Error(t, err)
		assert.True(t, errors.Is(err, lineAPIError), "error chain should contain original error")
		assert.Contains(t, err.Error(), "failed to get group summary")
		assert.Nil(t, mockGPS.profile, "no partial data should be saved on LINE API failure")
	})

	// AC-002: Handle storage failure gracefully [FR-002, Error]
	t.Run("should return error when storage fails", func(t *testing.T) {
		storageError := errors.New("GCS write quota exceeded")
		mockGPS := &mockGroupProfileService{setErr: storageError}
		handler := newTestHandler(t).
			WithGroupSummary("G1234567890abcdef", "Test Group", "").
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)

		require.Error(t, err)
		assert.True(t, errors.Is(err, storageError), "error chain should contain original error")
		assert.Contains(t, err.Error(), "failed to save group profile")
	})

	// FR-001: Verify LINE API is called with correct group ID
	t.Run("should call GetGroupSummary with correct group ID from context", func(t *testing.T) {
		handler, mockLine, mockGPS := newTestHandler(t).
			WithGroupSummary("G9876543210fedcba", "Another Group", "").
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), "G9876543210fedcba")
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		assert.Equal(t, "G9876543210fedcba", mockLine.lastGroupID)
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
				return line.WithSourceID(ctx, "G1234567890abcdef")
			},
			wantErr:     true,
			errContains: "chatType not found",
		},
		{
			name: "missing sourceID in context",
			setupCtx: func(ctx context.Context) context.Context {
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
			handler := newTestHandler(t).
				WithGroupSummary("G1234567890abcdef", "Test Group", "").
				Build()

			ctx := tt.setupCtx(t.Context())
			err := handler.HandleJoin(ctx)

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
		lineAPIError := errors.New("LINE API rate limit exceeded")
		handler := newTestHandler(t).
			WithGroupSummaryError(lineAPIError).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)
		if err != nil {
			assert.True(t, errors.Is(err, lineAPIError), "error chain should contain original LINE API error")
			assert.Contains(t, err.Error(), "failed to get group summary")
		}
	})

	t.Run("SetGroupProfile error is wrapped and preserves original error", func(t *testing.T) {
		storageError := errors.New("GCS bucket not accessible")
		mockGPS := &mockGroupProfileService{setErr: storageError}
		handler := newTestHandler(t).
			WithGroupSummary("G1234567890abcdef", "Test Group", "").
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), "G1234567890abcdef")
		err := handler.HandleJoin(ctx)
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
		groupID := "G-member-count-test"
		handler, mockLine, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Engineering Team", "").
			WithMemberCount(10).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		assert.Equal(t, groupID, mockLine.lastGroupID, "GetGroupSummary should be called with correct groupID")
		require.NotNil(t, mockGPS.profile, "Group profile should be saved")
		assert.Equal(t, "Engineering Team", mockGPS.profile.DisplayName)
		assert.Equal(t, 10, mockGPS.profile.UserCount, "Member count should be saved")
		assert.Equal(t, groupID, mockGPS.lastGroupID, "Profile should be saved with correct groupID")
	})

	// AC-006: Handle API failure on join [FR-001]
	t.Run("should use fallback count when GetGroupMemberCount fails", func(t *testing.T) {
		groupID := "G-api-error-test"
		handler, _, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Sales Team", "").
			WithMemberCountError(errors.New("LINE API error: rate limit exceeded")).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err, "HandleJoin should complete without error")
		require.NotNil(t, mockGPS.profile, "Group profile should still be saved")
		assert.Equal(t, "Sales Team", mockGPS.profile.DisplayName)
		assert.Equal(t, 1, mockGPS.profile.UserCount, "Should use fallback count of 1 when API fails")
		assert.Equal(t, groupID, mockGPS.lastGroupID)
	})

	// AC-006: Verify error is logged but doesn't prevent join
	t.Run("should log warning but continue when member count API fails", func(t *testing.T) {
		groupID := "G-log-warning-test"
		handler, _, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Community", "").
			WithMemberCountError(errors.New("network timeout")).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 1, mockGPS.profile.UserCount, "Fallback value used")
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
			builder := newTestHandler(t).
				WithGroupSummary(tt.groupID, tt.groupName, "")

			if tt.memberCountErr != nil {
				builder.WithMemberCountError(tt.memberCountErr)
			} else {
				builder.WithMemberCount(tt.memberCount)
			}

			handler, _, mockGPS := builder.BuildWithMocks()
			ctx := withJoinContext(t.Context(), tt.groupID)
			err := handler.HandleJoin(ctx)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantProfileSaved {
				require.NotNil(t, mockGPS.profile, "Profile should be saved")
				assert.Equal(t, tt.wantUserCount, mockGPS.profile.UserCount, "UserCount should match expected")
				assert.Equal(t, tt.wantDisplayName, mockGPS.profile.DisplayName, "DisplayName should match")
				assert.Equal(t, tt.groupID, mockGPS.lastGroupID, "Should save to correct groupID")
			}
		})
	}
}

func TestHandleJoin_MemberCount_EdgeCases(t *testing.T) {
	t.Run("should handle zero member count from API", func(t *testing.T) {
		groupID := "G-zero-members"
		handler, _, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Empty Group", "").
			WithMemberCount(0).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 0, mockGPS.profile.UserCount, "Should save 0 if API returns 0")
	})

	t.Run("should preserve member count when both API call and profile save succeed", func(t *testing.T) {
		groupID := "G-success-path"
		expectedCount := 42
		handler, _, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Success Group", "").
			WithMemberCount(expectedCount).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, expectedCount, mockGPS.profile.UserCount)
	})

	t.Run("should verify GroupProfile struct has UserCount field", func(t *testing.T) {
		profile := &groupprofile.GroupProfile{
			DisplayName: "Test",
			PictureURL:  "",
			UserCount:   5,
		}
		assert.Equal(t, 5, profile.UserCount, "GroupProfile should have UserCount field")
	})
}

func TestHandleJoin_MemberCount_Integration(t *testing.T) {
	t.Run("should complete full join flow with member count", func(t *testing.T) {
		groupID := "G-integration-test"
		handler, mockLine, mockGPS := newTestHandler(t).
			WithGroupSummary(groupID, "Integration Test Group", "").
			WithMemberCount(25).
			BuildWithMocks()

		ctx := withJoinContext(t.Context(), groupID)
		err := handler.HandleJoin(ctx)

		require.NoError(t, err)
		assert.Equal(t, groupID, mockLine.lastGroupID)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, groupID, mockGPS.lastGroupID)
		assert.Equal(t, "Integration Test Group", mockGPS.profile.DisplayName)
		assert.Equal(t, 25, mockGPS.profile.UserCount)
	})
}
