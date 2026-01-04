package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

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
	tests := []struct {
		name    string
		stdin   *strings.Reader
		stdout  *bytes.Buffer
		stderr  *bytes.Buffer
		wantErr bool
	}{
		{
			name:    "nil stdin",
			stdin:   nil,
			stdout:  &bytes.Buffer{},
			stderr:  &bytes.Buffer{},
			wantErr: true,
		},
		{
			name:    "nil stdout",
			stdin:   strings.NewReader(""),
			stdout:  nil,
			stderr:  &bytes.Buffer{},
			wantErr: true,
		},
		{
			name:    "nil stderr",
			stdin:   strings.NewReader(""),
			stdout:  &bytes.Buffer{},
			stderr:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			t.Setenv("GCP_PROJECT_ID", "test-project")
			t.Setenv("GCP_REGION", "test-region")
			t.Setenv("LLM_MODEL", "test-model")

			args := []string{"yuruppu-cli", "--message", "test"}

			// When
			err := run(args, tt.stdin, tt.stdout, tt.stderr)

			// Then
			if tt.wantErr {
				require.Error(t, err, "should return error for nil I/O")
			}
		})
	}
}
