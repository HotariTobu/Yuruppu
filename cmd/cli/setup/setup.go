package setup

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"yuruppu/cmd/cli/groupsim"
	"yuruppu/cmd/cli/mock"
)

// EnsureDataDir checks if dataDir exists. If not, prompts user for confirmation.
// Returns error if user declines or directory creation fails.
//
// FR-009: If storage directory does not exist, CLI prompts user for confirmation before creating it
// AC-008: Directory creation prompt and handling
func EnsureDataDir(dataDir string, stdin io.Reader, stderr io.Writer) error {
	// Check if dataDir is empty
	if dataDir == "" {
		return errors.New("data directory path cannot be empty")
	}

	// Check if path exists
	info, err := os.Stat(dataDir)
	if err == nil {
		// Path exists - check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path %s exists but is not a directory", dataDir)
		}
		// Directory exists - return nil without prompting
		return nil
	}

	// If error is not "not exist", return the error
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	// Directory does not exist - prompt user for confirmation
	_, _ = fmt.Fprintf(stderr, "Directory %s does not exist. Create it? [y/N] ", dataDir)

	// Read user input
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("read user input: %w", err)
		}
		return errors.New("read user input: unexpected EOF")
	}

	response := strings.TrimSpace(scanner.Text())

	// Check if user confirmed
	if response == "y" || response == "Y" {
		// Create directory with mode 0o750 (owner read/write/execute, group read/execute)
		if err := os.Mkdir(dataDir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil
	}

	// User declined
	return fmt.Errorf("user declined to create directory: %s", dataDir)
}

// EnsureGroup handles group creation and membership validation.
// Precondition: groupID must not be empty.
func EnsureGroup(ctx context.Context, dataDir, groupID, userID string) (*groupsim.Service, error) {
	groupSimStorage := mock.NewFileStorage(dataDir, "groupsim/")
	groupService, err := groupsim.NewService(groupSimStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to create group service: %w", err)
	}

	exists, err := groupService.Exists(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to check group existence: %w", err)
	}

	if !exists {
		if err := groupService.Create(ctx, groupID, userID); err != nil {
			return nil, fmt.Errorf("failed to create group: %w", err)
		}
		return groupService, nil
	}

	isMember, err := groupService.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check group membership: %w", err)
	}

	if !isMember {
		return nil, fmt.Errorf("user '%s' is not a member of group '%s'", userID, groupID)
	}

	return groupService, nil
}
