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
// HandleMemberLeft Tests - FR-003: Decrement member count when members leave
// =============================================================================

func TestHandleMemberLeft(t *testing.T) {
	// AC-003: Member count decremented on member leave [FR-003, FR-004]
	t.Run("should decrement member count when multiple members leave", func(t *testing.T) {
		groupID := "G-member-left-test"
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
		leftUserIDs := []string{"U-user-001", "U-user-002", "U-user-003"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		assert.Equal(t, groupID, mockGPS.lastGroupID, "Profile should be updated for correct groupID")
		require.NotNil(t, mockGPS.profile, "Profile should be saved")
		assert.Equal(t, 7, mockGPS.profile.UserCount, "UserCount should be 10 - 3 = 7")
	})

	// AC-003: Single member leave
	t.Run("should decrement member count by 1 when single member leaves", func(t *testing.T) {
		groupID := "G-single-leave"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Small Team",
			UserCount:   5,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		leftUserIDs := []string{"U-leaving-user"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 4, mockGPS.profile.UserCount, "UserCount should be 5 - 1 = 4")
	})

	// AC-003: Error handling - GetGroupProfile fails
	t.Run("should log warning and return nil when GetGroupProfile fails", func(t *testing.T) {
		groupID := "G-get-error"
		mockGPS := &mockGroupProfileService{getErr: errors.New("storage unavailable")}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		leftUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		assert.NoError(t, err, "Should not return error, only log warning")
		assert.Nil(t, mockGPS.profile, "Profile should not be updated on GetGroupProfile failure")
	})

	// AC-003: Error handling - SetGroupProfile fails
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
		leftUserIDs := []string{"U-user-001", "U-user-002"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		assert.NoError(t, err, "Should not return error, only log warning")
	})

	// Context validation
	t.Run("should return error when chatType is missing from context", func(t *testing.T) {
		handler := newTestHandler(t).Build()
		ctx := line.WithSourceID(context.Background(), "G-no-chattype")

		leftUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatType not found")
	})

	t.Run("should return error when sourceID is missing from context", func(t *testing.T) {
		handler := newTestHandler(t).Build()
		ctx := line.WithChatType(context.Background(), line.ChatTypeGroup)

		leftUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "sourceID not found")
	})
}

func TestHandleMemberLeft_TableDriven(t *testing.T) {
	tests := []struct {
		name              string
		initialUserCount  int
		leftUserIDs       []string
		expectedUserCount int
		getProfileErr     error
		setProfileErr     error
		wantErr           bool
		wantErrContains   string
		expectUpdate      bool
	}{
		{
			name:              "decrement by 1 when single member leaves",
			initialUserCount:  10,
			leftUserIDs:       []string{"U-single"},
			expectedUserCount: 9,
			expectUpdate:      true,
		},
		{
			name:              "decrement by 3 when multiple members leave",
			initialUserCount:  10,
			leftUserIDs:       []string{"U-user-001", "U-user-002", "U-user-003"},
			expectedUserCount: 7,
			expectUpdate:      true,
		},
		{
			name:              "decrement by 5 when five members leave",
			initialUserCount:  20,
			leftUserIDs:       []string{"U-1", "U-2", "U-3", "U-4", "U-5"},
			expectedUserCount: 15,
			expectUpdate:      true,
		},
		{
			name:              "handle small group shrinking from 2 to 1",
			initialUserCount:  2,
			leftUserIDs:       []string{"U-leaving-member"},
			expectedUserCount: 1,
			expectUpdate:      true,
		},
		{
			name:              "handle large group shrinking from 100 to 90",
			initialUserCount:  100,
			leftUserIDs:       []string{"U-1", "U-2", "U-3", "U-4", "U-5", "U-6", "U-7", "U-8", "U-9", "U-10"},
			expectedUserCount: 90,
			expectUpdate:      true,
		},
		{
			name:             "GetGroupProfile error returns nil without update",
			initialUserCount: 10,
			leftUserIDs:      []string{"U-user-001"},
			getProfileErr:    errors.New("storage read failed"),
			wantErr:          false,
			expectUpdate:     false,
		},
		{
			name:              "SetGroupProfile error still returns nil",
			initialUserCount:  10,
			leftUserIDs:       []string{"U-user-001"},
			expectedUserCount: 9,
			setProfileErr:     errors.New("storage write failed"),
			wantErr:           false,
			expectUpdate:      true,
		},
		{
			name:              "empty leftUserIDs array decrements by 0",
			initialUserCount:  15,
			leftUserIDs:       []string{},
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
			err := handler.HandleMemberLeft(ctx, tt.leftUserIDs)

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

func TestHandleMemberLeft_EdgeCases(t *testing.T) {
	t.Run("should handle decrement to zero", func(t *testing.T) {
		groupID := "G-to-zero"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Shrinking Group",
			UserCount:   2,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		leftUserIDs := []string{"U-first", "U-second"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 0, mockGPS.profile.UserCount, "Should decrement to 0")
	})

	t.Run("should handle decrement below zero (no minimum bound per spec)", func(t *testing.T) {
		// Per spec: "Handling member count going below 0 (cannot happen)" is out of scope
		// This means we don't need to guard against it, but the code should handle it gracefully
		groupID := "G-below-zero"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Edge Case Group",
			UserCount:   1,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		leftUserIDs := []string{"U-first", "U-second", "U-third"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, -2, mockGPS.profile.UserCount, "No minimum bound check per spec")
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
		leftUserIDs := []string{"U-user-001"}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 7, mockGPS.profile.UserCount, "UserCount should be updated")
		assert.Equal(t, "Complete Profile Group", mockGPS.profile.DisplayName, "DisplayName should be preserved")
		assert.Equal(t, "https://example.com/picture.jpg", mockGPS.profile.PictureURL, "PictureURL should be preserved")
		assert.Equal(t, "image/jpeg", mockGPS.profile.PictureMIMEType, "PictureMIMEType should be preserved")
	})

	t.Run("should handle very large member count decrement", func(t *testing.T) {
		groupID := "G-large-decrement"
		initialProfile := &groupprofile.GroupProfile{
			DisplayName: "Large Group",
			UserCount:   500,
		}
		mockGPS := &mockGroupProfileService{profile: initialProfile}
		handler := newTestHandler(t).
			WithGroupProfile(mockGPS).
			Build()

		ctx := withJoinContext(t.Context(), groupID)
		leftUserIDs := make([]string, 50)
		for i := range leftUserIDs {
			leftUserIDs[i] = fmt.Sprintf("U-user-%03d", i+1)
		}
		err := handler.HandleMemberLeft(ctx, leftUserIDs)

		require.NoError(t, err)
		require.NotNil(t, mockGPS.profile)
		assert.Equal(t, 450, mockGPS.profile.UserCount, "Should handle large decrement: 500 - 50 = 450")
	})
}
