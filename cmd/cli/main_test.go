package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"yuruppu/cmd/cli/groupsim"
	"yuruppu/cmd/cli/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_UserIDValidation tests user ID pattern validation
// AC-006: Invalid user ID rejection [FR-004]
func TestRun_UserIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid user ID with lowercase letters",
			userID:  "user",
			wantErr: false,
		},
		{
			name:    "valid user ID with numbers",
			userID:  "user123",
			wantErr: false,
		},
		{
			name:    "valid user ID with underscore",
			userID:  "user_123",
			wantErr: false,
		},
		{
			name:    "valid user ID with only numbers",
			userID:  "123456",
			wantErr: false,
		},
		{
			name:    "valid user ID with only underscores",
			userID:  "___",
			wantErr: false,
		},
		{
			name:    "default user ID",
			userID:  "default",
			wantErr: false,
		},
		{
			name:    "invalid user ID with uppercase letters",
			userID:  "User123",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "invalid user ID with special character @",
			userID:  "User@123",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "invalid user ID with hyphen",
			userID:  "user-123",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "invalid user ID with space",
			userID:  "user 123",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "invalid user ID with dot",
			userID:  "user.123",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
		{
			name:    "empty user ID",
			userID:  "",
			wantErr: true,
			errMsg:  "invalid user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Set minimal required environment variables
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			// Setup test data directory
			dataDir := t.TempDir()

			args := []string{
				"yuruppu-cli",
				"--user-id", tt.userID,
				"--data-dir", dataDir,
				"--message", "test message",
			}
			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			err := run(args, stdin, stdout, stderr)

			// Then
			if tt.wantErr {
				require.Error(t, err, "should return error for invalid user ID")
				assert.Contains(t, err.Error(), tt.errMsg,
					"error message should indicate invalid user ID")
			} else if err != nil {
				// For valid user IDs, the test may fail due to missing dependencies
				// but it should NOT fail on user ID validation
				assert.NotContains(t, err.Error(), "invalid user ID",
					"should not fail on user ID validation for valid user ID")
			}
		})
	}
}

// TestRun_UserIDValidation_ExitCode tests that invalid user ID exits with non-zero status
// AC-006: CLI exits with non-zero status [FR-004]
func TestRun_UserIDValidation_ExitCode(t *testing.T) {
	t.Run("should return error for invalid user ID", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "Invalid@User",
			"--data-dir", dataDir,
			"--message", "test",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		require.Error(t, err, "should return error for invalid user ID")
	})
}

// TestRun_MinimalConfiguration tests that only LLM env vars are required
// AC-010: Minimal configuration [NFR-001]
func TestRun_MinimalConfiguration(t *testing.T) {
	t.Run("should start with only LLM environment variables", func(t *testing.T) {
		// Given: Only LLM environment variables are set
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		// Ensure LINE and GCS credentials are NOT set
		os.Unsetenv("LINE_CHANNEL_SECRET")
		os.Unsetenv("LINE_CHANNEL_ACCESS_TOKEN")
		os.Unsetenv("PROFILE_BUCKET")
		os.Unsetenv("HISTORY_BUCKET")
		os.Unsetenv("MEDIA_BUCKET")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "testuser",
			"--data-dir", dataDir,
			"--message", "Hello",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)
		// Then: The function may fail due to missing Agent/Handler implementation,
		// but it should NOT complain about missing LINE/GCS credentials.
		if err != nil {
			stderrOutput := stderr.String()
			assert.NotContains(t, stderrOutput, "LINE_CHANNEL_SECRET",
				"should not reference missing LINE_CHANNEL_SECRET")
			assert.NotContains(t, stderrOutput, "LINE_CHANNEL_ACCESS_TOKEN",
				"should not reference missing LINE_CHANNEL_ACCESS_TOKEN")
			assert.NotContains(t, stderrOutput, "PROFILE_BUCKET",
				"should not reference missing PROFILE_BUCKET")
			assert.NotContains(t, stderrOutput, "HISTORY_BUCKET",
				"should not reference missing HISTORY_BUCKET")
			assert.NotContains(t, stderrOutput, "MEDIA_BUCKET",
				"should not reference missing MEDIA_BUCKET")
		}
	})
}

// TestRun_MinimalConfiguration_MissingLLMVars tests error when LLM env vars are missing
// NFR-001: CLI requires only LLM-related environment variables
func TestRun_MinimalConfiguration_MissingLLMVars(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func(*testing.T)
		wantErrMsg string
	}{
		{
			name: "missing GCP_PROJECT_ID",
			setupEnv: func(t *testing.T) {
				os.Unsetenv("GCP_PROJECT_ID")
				t.Setenv("GCP_REGION", "test-region")
				t.Setenv("LLM_MODEL", "test-model")
			},
			wantErrMsg: "GCP_PROJECT_ID",
		},
		{
			name: "missing GCP_REGION",
			setupEnv: func(t *testing.T) {
				t.Setenv("GCP_PROJECT_ID", "test-project")
				os.Unsetenv("GCP_REGION")
				t.Setenv("LLM_MODEL", "test-model")
			},
			wantErrMsg: "GCP_REGION",
		},
		{
			name: "missing LLM_MODEL",
			setupEnv: func(t *testing.T) {
				t.Setenv("GCP_PROJECT_ID", "test-project")
				t.Setenv("GCP_REGION", "test-region")
				os.Unsetenv("LLM_MODEL")
			},
			wantErrMsg: "LLM_MODEL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			tt.setupEnv(t)

			dataDir := t.TempDir()

			args := []string{
				"yuruppu-cli",
				"--user-id", "testuser",
				"--data-dir", dataDir,
				"--message", "test",
			}
			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			err := run(args, stdin, stdout, stderr)

			// Then
			require.Error(t, err, "should return error when LLM env var is missing")
			errOutput := err.Error()
			assert.Contains(t, errOutput, tt.wantErrMsg,
				"error message should mention missing environment variable")
		})
	}
}

// TestRun_FlagParsing tests flag parsing and default values
// FR-002, FR-004, FR-008: CLI supports flags
func TestRun_FlagParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedUserID  string
		expectedDataDir string
		expectMessage   bool
	}{
		{
			name:           "default user ID when not specified",
			args:           []string{"yuruppu-cli"},
			expectedUserID: "default",
		},
		{
			name:           "custom user ID via flag",
			args:           []string{"yuruppu-cli", "--user-id", "custom123"},
			expectedUserID: "custom123",
		},
		{
			name:            "custom data dir via flag",
			args:            []string{"yuruppu-cli", "--data-dir", "/tmp/custom"},
			expectedDataDir: "/tmp/custom",
		},
		{
			name:          "message flag for single-turn mode",
			args:          []string{"yuruppu-cli", "--message", "Hello"},
			expectMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates that flag parsing logic exists
			// The actual implementation will determine behavior

			// Given: Minimal setup to allow flag parsing
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			// Note: This will fail because implementation doesn't exist yet (TDD)
			err := run(tt.args, stdin, stdout, stderr)

			// Then
			// We can't make assertions about success since the implementation
			// doesn't exist yet. This test will pass once main.go is implemented.
			_ = err
		})
	}
}

// TestRun_SingleTurnMode tests single-turn message mode
// AC-002: Single-turn message mode [FR-002]
func TestRun_SingleTurnMode(t *testing.T) {
	t.Run("should send message and exit when --message flag is provided", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "testuser",
			"--data-dir", dataDir,
			"--message", "Hello, bot!",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented, it should:
		// 1. Not enter REPL mode
		// 2. Send the message
		// 3. Print response to stdout
		// 4. Exit
		_ = err
	})
}

// TestRun_REPLMode tests REPL mode when no --message flag
// AC-001: Interactive REPL mode [FR-001, FR-003]
func TestRun_REPLMode(t *testing.T) {
	t.Run("should enter REPL mode when --message flag is not provided", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "testuser",
			"--data-dir", dataDir,
		}
		// Simulate immediate /quit to exit REPL
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented, it should:
		// 1. Enter REPL mode
		// 2. Display "> " prompt
		// 3. Wait for user input
		// 4. Process /quit and exit
		_ = err
	})
}

// TestRun_DataDirDefault tests default data directory
// FR-008: Storage directory is configurable via --data-dir flag (default: .yuruppu/)
func TestRun_DataDirDefault(t *testing.T) {
	t.Run("should use .yuruppu/ as default data directory", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		args := []string{
			"yuruppu-cli",
			"--user-id", "testuser",
			"--message", "test",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented, it should use ".yuruppu/" as default data directory
		_ = err
	})
}

// TestRun_VerboseLogging tests verbose logging to stderr
// FR-011: CLI outputs verbose logs to stderr
func TestRun_VerboseLogging(t *testing.T) {
	t.Run("should output verbose logs to stderr", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "testuser",
			"--data-dir", dataDir,
			"--message", "test message",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented, stderr should contain verbose logs about:
		// - Tool calls
		// - LLM processing
		// - Storage operations
		_ = err

		// Once implemented, verify logging:
		// stderrOutput := stderr.String()
		// assert.NotEmpty(t, stderrOutput, "should output logs to stderr")
	})
}

// TestRun_ExistingUserProfile tests loading existing profile
// AC-003: Existing user profile [FR-004, FR-006]
func TestRun_ExistingUserProfile(t *testing.T) {
	t.Run("should load existing profile without prompting", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		// Pre-create a profile file
		// (This will be implemented once FileStorage is available)

		args := []string{
			"yuruppu-cli",
			"--user-id", "existinguser",
			"--data-dir", dataDir,
			"--message", "test",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. Profile should be loaded from local filesystem
		// 2. No name input prompt should be shown
		// 3. Conversation should start immediately
		_ = err
	})
}

// TestRun_NewUserProfile tests new user profile creation
// AC-004: New user profile creation [FR-004, FR-005]
func TestRun_NewUserProfile(t *testing.T) {
	t.Run("should prompt for profile when user is new", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "newuser",
			"--data-dir", dataDir,
			"--message", "test",
		}
		// Simulate profile input: display name, skip picture URL, skip status
		stdin := strings.NewReader("New User\n\n\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. CLI should prompt for display name (required)
		// 2. CLI should prompt for picture URL (optional)
		// 3. CLI should prompt for status message (optional)
		// 4. Profile should be saved to local filesystem
		// 5. Conversation should start after profile creation
		_ = err
	})
}

// TestRun_InvalidArgs tests handling of invalid command-line arguments
func TestRun_InvalidArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErrMsg string
	}{
		{
			name:       "unknown flag",
			args:       []string{"yuruppu-cli", "--unknown-flag", "value"},
			wantErrMsg: "flag",
		},
		{
			name:       "flag without value",
			args:       []string{"yuruppu-cli", "--user-id"},
			wantErrMsg: "flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			err := run(tt.args, stdin, stdout, stderr)

			// Then
			require.Error(t, err, "should return error for invalid arguments")
			assert.Contains(t, err.Error(), tt.wantErrMsg,
				"error message should mention flag issue")
		})
	}
}

// TestRun_NilIO tests error handling for nil I/O
func TestRun_NilIO(t *testing.T) {
	t.Run("nil stdin", func(t *testing.T) {
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		args := []string{"yuruppu-cli", "--message", "test"}
		err := run(args, nil, &bytes.Buffer{}, &bytes.Buffer{})
		require.Error(t, err, "should return error for nil stdin")
	})

	t.Run("nil stdout", func(t *testing.T) {
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		args := []string{"yuruppu-cli", "--message", "test"}
		err := run(args, strings.NewReader(""), nil, &bytes.Buffer{})
		require.Error(t, err, "should return error for nil stdout")
	})

	t.Run("nil stderr", func(t *testing.T) {
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		args := []string{"yuruppu-cli", "--message", "test"}
		err := run(args, strings.NewReader(""), &bytes.Buffer{}, nil)
		require.Error(t, err, "should return error for nil stderr")
	})
}

// TestRun_GroupID_FlagParsing tests group-id flag parsing
// FR-001: CLI accepts optional -group-id flag
func TestRun_GroupID_FlagParsing(t *testing.T) {
	t.Run("group-id flag is parsed successfully", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()
		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
			"--message", "test",
		}

		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)
		// Then: The run will fail due to Gemini API not available in tests,
		// but it should NOT fail on flag parsing
		if err != nil {
			assert.NotContains(t, err.Error(), "flag",
				"should not fail on flag parsing for valid flags")
		}
	})

	t.Run("no group-id flag means 1-on-1 mode", func(t *testing.T) {
		// Given
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()
		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--data-dir", dataDir,
			"--message", "test",
		}

		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)
		// Then: The run will fail due to Gemini API not available in tests,
		// but it should NOT fail on flag parsing
		if err != nil {
			assert.NotContains(t, err.Error(), "flag",
				"should not fail on flag parsing for valid flags")
		}
	})
}

// TestRun_GroupID_CreateNewGroup tests creating a new group when it doesn't exist
// AC-001: Create new group [FR-001, FR-002]
func TestRun_GroupID_CreateNewGroup(t *testing.T) {
	t.Run("should create new group when group-id is specified and group does not exist", func(t *testing.T) {
		// Given: CLI is invoked with -user-id alice -group-id mygroup
		// And: Group "mygroup" does not exist
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
			"--message", "Hello in group",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When: The CLI starts
		err := run(args, stdin, stdout, stderr)

		// Then:
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented, it should:
		// 1. Group "mygroup" is created
		// 2. "alice" is added as the first member
		// 3. REPL starts in group chat mode (or processes single message)
		_ = err

		// Once implemented, verify group was created:
		// groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
		// groupService, _ := groupsim.NewService(groupStorage)
		// exists, _ := groupService.Exists(context.Background(), "mygroup")
		// assert.True(t, exists, "group should be created")
		// isMember, _ := groupService.IsMember(context.Background(), "mygroup", "alice")
		// assert.True(t, isMember, "alice should be first member")
	})
}

// TestRun_GroupID_JoinExistingGroup tests joining an existing group as a member
// AC-002: Join existing group [FR-001, FR-003]
func TestRun_GroupID_JoinExistingGroup(t *testing.T) {
	t.Run("should start in group chat mode when user is already a member", func(t *testing.T) {
		// Given: Group "mygroup" exists with members ["alice", "bob"]
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		// Pre-create group with alice and bob as members
		// groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
		// groupService, _ := groupsim.NewService(groupStorage)
		// ctx := context.Background()
		// _ = groupService.Create(ctx, "mygroup", "alice")
		// _ = groupService.AddMember(ctx, "mygroup", "bob")

		// When: CLI is invoked with -user-id alice -group-id mygroup
		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
			"--message", "Hello in group",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		err := run(args, stdin, stdout, stderr)

		// Then:
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. REPL starts in group chat mode
		// 2. "alice" is the active user
		// 3. Messages are sent with group context
		_ = err
	})
}

// TestRun_GroupID_RejectNonMember tests rejection of non-members
// AC-003: Reject non-member [FR-004]
func TestRun_GroupID_RejectNonMember(t *testing.T) {
	t.Run("should reject user who is not a member of existing group", func(t *testing.T) {
		// Given: Group "mygroup" exists with members ["alice", "bob"]
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		// Pre-create group with alice and bob as members
		groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
		groupService, err := groupsim.NewService(groupStorage)
		require.NoError(t, err, "failed to create group service")
		ctx := context.Background()
		err = groupService.Create(ctx, "mygroup", "alice")
		require.NoError(t, err, "failed to create group")
		err = groupService.AddMember(ctx, "mygroup", "bob")
		require.NoError(t, err, "failed to add member")

		// When: CLI is invoked with -user-id charlie -group-id mygroup
		args := []string{
			"yuruppu-cli",
			"--user-id", "charlie",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
			"--message", "Hello",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		err = run(args, stdin, stdout, stderr)

		// Then:
		// Error message to stderr: "user 'charlie' is not a member of group 'mygroup'"
		// CLI exits with non-zero status
		require.Error(t, err, "should return error for non-member access")
		assert.Contains(t, err.Error(), "charlie", "error should mention the user")
		assert.Contains(t, err.Error(), "mygroup", "error should mention the group")
		assert.Contains(t, err.Error(), "not a member", "error should indicate membership issue")
	})
}

// TestRun_GroupID_NoGroupID_OneOnOneMode tests 1-on-1 mode when no group-id is specified
// AC-004: No group-id means 1-on-1 [FR-005]
func TestRun_GroupID_NoGroupID_OneOnOneMode(t *testing.T) {
	t.Run("should use 1-on-1 chat mode when group-id is not specified", func(t *testing.T) {
		// Given: CLI is invoked with -user-id alice (no -group-id)
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--data-dir", dataDir,
			"--message", "Hello",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When: User sends a message
		err := run(args, stdin, stdout, stderr)

		// Then:
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. Chat type is "1-on-1"
		// 2. Source ID equals user ID ("alice")
		_ = err

		// Once implemented, verify 1-on-1 mode:
		// (Check that group context is NOT set, source ID = user ID)
	})
}

// TestRun_GroupID_TableDriven tests various group-id scenarios
// FR-001, FR-002, FR-003, FR-004, FR-005
func TestRun_GroupID_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		groupID     string
		setupGroup  func(dataDir string) error
		wantErr     bool
		errContains string
	}{
		{
			name:    "create new group - user is first member",
			userID:  "alice",
			groupID: "newgroup",
			setupGroup: func(dataDir string) error {
				// No setup - group doesn't exist
				return nil
			},
			wantErr: false,
		},
		{
			name:    "join existing group as member",
			userID:  "alice",
			groupID: "existinggroup",
			setupGroup: func(dataDir string) error {
				// Pre-create group with alice as member
				groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
				groupService, err := groupsim.NewService(groupStorage)
				if err != nil {
					return err
				}
				return groupService.Create(context.Background(), "existinggroup", "alice")
			},
			wantErr: false,
		},
		{
			name:    "reject non-member of existing group",
			userID:  "charlie",
			groupID: "alicebobgroup",
			setupGroup: func(dataDir string) error {
				// Pre-create group with alice and bob, but not charlie
				groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
				groupService, err := groupsim.NewService(groupStorage)
				if err != nil {
					return err
				}
				ctx := context.Background()
				if err := groupService.Create(ctx, "alicebobgroup", "alice"); err != nil {
					return err
				}
				return groupService.AddMember(ctx, "alicebobgroup", "bob")
			},
			wantErr:     true,
			errContains: "not a member",
		},
		{
			name:    "no group-id - 1-on-1 mode",
			userID:  "alice",
			groupID: "", // Empty string means no group
			setupGroup: func(dataDir string) error {
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			dataDir := t.TempDir()

			// Setup group if needed
			if tt.setupGroup != nil {
				err := tt.setupGroup(dataDir)
				require.NoError(t, err, "group setup should not fail")
			}

			// Build args
			args := []string{
				"yuruppu-cli",
				"--user-id", tt.userID,
				"--data-dir", dataDir,
				"--message", "test",
			}
			if tt.groupID != "" {
				args = append(args, "--group-id", tt.groupID)
			}

			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			err := run(args, stdin, stdout, stderr)

			// Then
			if tt.wantErr {
				require.Error(t, err, "should return error")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"error should contain expected message")
				}
			}
			// Note: Success cases will fail until implementation is complete (TDD)
		})
	}
}

// TestRun_GroupID_SingleTurnMode tests single-turn mode with group-id
// AC-018: Single-turn mode with group [FR-001, FR-006]
func TestRun_GroupID_SingleTurnMode(t *testing.T) {
	t.Run("should process message in group context for single-turn mode", func(t *testing.T) {
		// Given: Group "mygroup" exists with members ["alice", "bob"] and bot
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		// Pre-create group with alice, bob, and bot
		// groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
		// groupService, _ := groupsim.NewService(groupStorage)
		// ctx := context.Background()
		// _ = groupService.Create(ctx, "mygroup", "alice")
		// _ = groupService.AddMember(ctx, "mygroup", "bob")
		// _ = groupService.AddBot(ctx, "mygroup")

		// When: CLI is invoked with -user-id alice -group-id mygroup -message "Hello"
		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
			"--message", "Hello",
		}
		stdin := strings.NewReader("")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		err := run(args, stdin, stdout, stderr)

		// Then:
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. Message is processed as "alice" speaking in group "mygroup"
		// 2. Chat type is "group", source ID is "mygroup", user ID is "alice"
		// 3. Bot response is displayed
		// 4. CLI exits (no REPL)
		_ = err
	})
}

// TestRun_GroupID_REPLMode tests REPL mode with group-id
// FR-006, FR-007: Group chat REPL with proper context
func TestRun_GroupID_REPLMode(t *testing.T) {
	t.Run("should enter REPL in group chat mode", func(t *testing.T) {
		// Given: Group "mygroup" exists with members ["alice"]
		t.Setenv("GCP_PROJECT_ID", "test-project")
		t.Setenv("GCP_REGION", "test-region")
		t.Setenv("LLM_MODEL", "test-model")

		dataDir := t.TempDir()

		// Pre-create group with alice
		// groupStorage := mock.NewFileStorage(dataDir, "groupsim/")
		// groupService, _ := groupsim.NewService(groupStorage)
		// _ = groupService.Create(context.Background(), "mygroup", "alice")

		args := []string{
			"yuruppu-cli",
			"--user-id", "alice",
			"--group-id", "mygroup",
			"--data-dir", dataDir,
		}
		// Simulate immediate /quit to exit REPL
		stdin := strings.NewReader("/quit\n")
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		// When
		err := run(args, stdin, stdout, stderr)

		// Then:
		// The function will fail because implementation doesn't exist yet (TDD)
		// Once implemented:
		// 1. REPL starts in group chat mode
		// 2. Prompt shows user info (e.g., "Alice(alice)> " or "(alice)> ")
		// 3. Messages are sent with group context (type="group", sourceID="mygroup")
		_ = err
	})
}

// TestRun_GroupID_Validation tests group-id validation
// Edge cases for group-id format
func TestRun_GroupID_Validation(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid lowercase group-id",
			groupID: "mygroup",
			wantErr: false,
		},
		{
			name:    "valid group-id with numbers",
			groupID: "group123",
			wantErr: false,
		},
		{
			name:    "valid group-id with underscore",
			groupID: "my_group",
			wantErr: false,
		},
		{
			name:    "empty group-id treated as no group (1-on-1 mode)",
			groupID: "",
			wantErr: false,
		},
		// Note: Group ID validation rules depend on implementation
		// Add more validation tests as needed based on spec requirements
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			dataDir := t.TempDir()

			args := []string{
				"yuruppu-cli",
				"--user-id", "testuser",
				"--data-dir", dataDir,
				"--message", "test",
			}
			if tt.groupID != "" {
				args = append(args, "--group-id", tt.groupID)
			}

			stdin := strings.NewReader("")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			// When
			err := run(args, stdin, stdout, stderr)

			// Then
			if tt.wantErr {
				require.Error(t, err, "should return error for invalid group-id")
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg,
						"error message should indicate invalid group-id")
				}
			}
			// Note: Success cases will fail until implementation is complete (TDD)
		})
	}
}
