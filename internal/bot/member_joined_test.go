package bot_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"yuruppu/internal/groupprofile"
	"yuruppu/internal/line"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// HandleMemberJoined Tests - FR-002: Increment member count when members join
// =============================================================================

func TestHandleMemberJoined(t *testing.T) {
	// AC-002: Member count incremented on member join [FR-002, FR-004]
	t.Run("should increment member count when multiple members join", func(t *testing.T) {
		groupID := "G-member-join-test"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Engineering Team",
			PictureURL:  "",
			UserCount:   10,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-user-001", "U-user-002", "U-user-003"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.NoError(t, err)
		assert.Equal(t, groupID, mockGPS.lastGroupID, "Profile should be updated for correct groupID")
		require.NotNil(t, mockGPS.profile, "Profile should be saved")
		assert.Equal(t, 13, mockGPS.profile.UserCount, "UserCount should be 10 + 3 = 13")
	})

	// AC-002: Single member join
	t.Run("should increment member count by 1 when single member joins", func(t *testing.T) {
		groupID := "G-single-join"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Small Team",
			UserCount:   5,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-new-user"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 6, mockGPS.profile.UserCount, "UserCount should be 5 + 1 = 6")
	})

	// AC-002: Error handling - GetGroupProfile fails
	t.Run("should log warning and return nil when GetGroupProfile fails", func(t *testing.T) {
		groupID := "G-get-error"
		mockGPS := &mockGroupProfileService{getErr: errors.New("storage unavailable")}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		assert.NoError(t, err, "Should not return error, only log warning")
		assert.Nil(t, mockGPS.profile, "Profile should not be updated on GetGroupProfile failure")
	})

	// AC-002: Error handling - SetGroupProfile fails
	t.Run("should log warning and continue when SetGroupProfile fails", func(t *testing.T) {
		groupID := "G-set-error"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Test Group",
			UserCount:   20,
		}
		mockGPS := &mockGroupProfileService{
			profile: initialProfile,
			setErr:  errors.New("write permission denied"),
		}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-user-001", "U-user-002"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		assert.NoError(t, err, "Should not return error, only log warning")
	})

	// Context validation
	t.Run("should return error when chatType is missing from context", func(t *testing.T) {
		handler := newTestHandler(t).Build()
		ctx := line.WithSourceID(context.Background(), "G-no-chattype")

		joinedUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatType not found")
	})

	t.Run("should return error when sourceID is missing from context", func(t *testing.T) {
		handler := newTestHandler(t).Build()
		ctx := line.WithChatType(context.Background(), line.ChatTypeGroup)

		joinedUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

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
			groupID := "G-table-test"
			initialProfile := &groupprofile.GroupProfile{
				DisplayName: "Table Test Group",
				UserCount:   tt.initialUserCount,
			}
			mockGPS := &mockGroupProfileService{
				profile: initialProfile,
				getErr:  tt.getProfileErr,
				setErr:  tt.setProfileErr,
			}
			handler := newTestHandler(t).
				WithGroupProfile(mockGPS).
				Build()

			ctx := withJoinContext(t.Context(), groupID)
			err := handler.HandleMemberJoined(ctx, tt.joinedUserIDs)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
				return
			}
			require.NoError(t, err)

			if tt.expectUpdate && tt.getProfileErr == nil {
				require.NotNil(t, mockGPS.profile, "Profile should be updated")
				assert.Equal(t, tt.expectedUserCount, mockGPS.profile.UserCount, "UserCount should match expected value")
				assert.Equal(t, groupID, mockGPS.lastGroupID, "Should update correct groupID")
			}
		})
	}
}

func TestHandleMemberJoined_EdgeCases(t *testing.T) {
	t.Run("should handle zero initial count", func(t *testing.T) {
		groupID := "G-zero-count"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Empty Group",
			UserCount:   0,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-first", "U-second"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 2, mockGPS.profile.UserCount, "Should increment from 0 to 2")
	})

	t.Run("should preserve other profile fields when updating count", func(t *testing.T) {
		groupID := "G-preserve-fields"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName:     "Complete Profile Group",
			PictureURL:      "https://example.com/picture.jpg",
			PictureMIMEType: "image/jpeg",
			UserCount:       8,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 9, mockGPS.profile.UserCount, "UserCount should be updated")
		assert.Equal(t, "Complete Profile Group", mockGPS.profile.DisplayName, "DisplayName should be preserved")
		assert.Equal(t, "https://example.com/picture.jpg", mockGPS.profile.PictureURL, "PictureURL should be preserved")
		assert.Equal(t, "image/jpeg", mockGPS.profile.PictureMIMEType, "PictureMIMEType should be preserved")
	})

	t.Run("should handle very large member count increment", func(t *testing.T) {
		groupID := "G-large-increment"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Large Group",
			UserCount:   500,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		joinedUserIDs := make([]string, 50)
		for i := range joinedUserIDs {
			joinedUserIDs[i] = fmt.Sprintf("U-user-%03d", i+1)
		}
		err := handler.HandleMemberJoined(ctx, joinedUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 550, mockGPS.profile.UserCount, "Should handle large increment: 500 + 50 = 550")
	})
}
