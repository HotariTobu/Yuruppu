package setup_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"yuruppu/cmd/cli/setup"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureDataDir_DirectoryExists tests that no prompt is shown when directory exists
// AC-008: Directory already exists - should return nil without prompting
func TestEnsureDataDir_DirectoryExists(t *testing.T) {
	t.Run("should return nil without prompting when directory exists", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		stdin := strings.NewReader("")
		stderr := &bytes.Buffer{}

		// When
		err := setup.EnsureDataDir(dataDir, stdin, stderr)

		// Then
		require.NoError(t, err)
		assert.Empty(t, stderr.String(), "should not write to stderr when directory exists")
	})
}

// TestEnsureDataDir_UserConfirms tests that directory is created when user enters "y"
// AC-008: If user enters "y": directory is created and CLI continues
func TestEnsureDataDir_UserConfirms(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "lowercase y",
			input: "y\n",
		},
		{
			name:  "uppercase Y",
			input: "Y\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			dataDir := filepath.Join(t.TempDir(), "new-dir")
			stdin := strings.NewReader(tt.input)
			stderr := &bytes.Buffer{}

			// Ensure directory does not exist
			_, err := os.Stat(dataDir)
			require.True(t, os.IsNotExist(err), "directory should not exist before test")

			// When
			err = setup.EnsureDataDir(dataDir, stdin, stderr)

			// Then
			require.NoError(t, err)
			assert.Contains(t, stderr.String(), "Directory "+dataDir+" does not exist. Create it? [y/N]",
				"should prompt user for confirmation")

			// Verify directory was created
			info, statErr := os.Stat(dataDir)
			require.NoError(t, statErr, "directory should exist after confirmation")
			assert.True(t, info.IsDir(), "created path should be a directory")
		})
	}
}

// TestEnsureDataDir_UserDeclines tests that CLI exits when user declines
// AC-008: If user enters anything else: CLI exits
func TestEnsureDataDir_UserDeclines(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "lowercase n",
			input: "n\n",
		},
		{
			name:  "uppercase N",
			input: "N\n",
		},
		{
			name:  "empty input (just Enter - default is N)",
			input: "\n",
		},
		{
			name:  "random text",
			input: "maybe\n",
		},
		{
			name:  "whitespace only",
			input: "  \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			dataDir := filepath.Join(t.TempDir(), "declined-dir")
			stdin := strings.NewReader(tt.input)
			stderr := &bytes.Buffer{}

			// Ensure directory does not exist
			_, err := os.Stat(dataDir)
			require.True(t, os.IsNotExist(err), "directory should not exist before test")

			// When
			err = setup.EnsureDataDir(dataDir, stdin, stderr)

			// Then
			require.Error(t, err, "should return error when user declines")
			assert.Contains(t, stderr.String(), "Directory "+dataDir+" does not exist. Create it? [y/N]",
				"should prompt user for confirmation")

			// Verify directory was NOT created
			_, statErr := os.Stat(dataDir)
			assert.True(t, os.IsNotExist(statErr), "directory should not be created when user declines")
		})
	}
}

// TestEnsureDataDir_ParentDirectoryDoesNotExist tests error when parent directory doesn't exist
// Note: os.Mkdir doesn't create parent directories, only the final directory
func TestEnsureDataDir_ParentDirectoryDoesNotExist(t *testing.T) {
	t.Run("should fail when parent directory does not exist", func(t *testing.T) {
		// Given
		tempDir := t.TempDir()
		dataDir := filepath.Join(tempDir, "nonexistent", "parent", "new-dir")
		stdin := strings.NewReader("y\n")
		stderr := &bytes.Buffer{}

		// Ensure parent directory does not exist
		parentDir := filepath.Dir(dataDir)
		_, err := os.Stat(parentDir)
		require.True(t, os.IsNotExist(err), "parent directory should not exist")

		// When
		err = setup.EnsureDataDir(dataDir, stdin, stderr)

		// Then
		require.Error(t, err, "should fail when parent directory does not exist")

		// Verify directory was not created
		_, statErr := os.Stat(dataDir)
		assert.True(t, os.IsNotExist(statErr), "directory should not be created when parent doesn't exist")
	})
}

// TestEnsureDataDir_EmptyPath tests behavior with empty path
func TestEnsureDataDir_EmptyPath(t *testing.T) {
	t.Run("should return error for empty path", func(t *testing.T) {
		// Given
		dataDir := ""
		stdin := strings.NewReader("")
		stderr := &bytes.Buffer{}

		// When
		err := setup.EnsureDataDir(dataDir, stdin, stderr)

		// Then
		require.Error(t, err, "should return error for empty path")
	})
}

// TestEnsureDataDir_PathIsFile tests error when path exists but is a file, not a directory
func TestEnsureDataDir_PathIsFile(t *testing.T) {
	t.Run("should return error when path exists but is a file", func(t *testing.T) {
		// Given
		tempDir := t.TempDir()
		dataDir := filepath.Join(tempDir, "existing-file")

		// Create a file at the path
		require.NoError(t, os.WriteFile(dataDir, []byte("test"), 0o644))

		stdin := strings.NewReader("")
		stderr := &bytes.Buffer{}

		// When
		err := setup.EnsureDataDir(dataDir, stdin, stderr)

		// Then
		require.Error(t, err, "should return error when path is a file")
	})
}
