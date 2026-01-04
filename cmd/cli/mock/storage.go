package mock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileStorage implements storage.Storage interface using local filesystem.
// It uses file modification time (UnixNano) as the generation number.
type FileStorage struct {
	dataDir string
}

// NewFileStorage creates a new FileStorage instance with the given data directory.
func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{
		dataDir: dataDir,
	}
}

// Read retrieves data for a key from the filesystem.
// Returns nil, 0, nil if the key doesn't exist.
func (fs *FileStorage) Read(_ context.Context, key string) ([]byte, int64, error) {
	filePath := filepath.Join(fs.dataDir, key)

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return nil, 0, nil as per spec
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file content
	data, err := os.ReadFile(filePath) //nolint:gosec // CLI tool reads user-specified files by design
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Use file modification time as generation
	return data, info.ModTime().UnixNano(), nil
}

// Write stores data for a key with optional generation precondition.
// If expectedGeneration is 0, creates new object (fails if exists).
// If expectedGeneration > 0, updates only if generation matches (fails if mismatch).
// Returns the new generation number of the written object.
func (fs *FileStorage) Write(_ context.Context, key, _ string, data []byte, expectedGeneration int64) (int64, error) {
	filePath := filepath.Join(fs.dataDir, key)

	// Check if file exists
	info, statErr := os.Stat(filePath)
	fileExists := statErr == nil

	if expectedGeneration == 0 {
		// Creating new file - must not exist
		if fileExists {
			return 0, errors.New("file already exists")
		}
	} else {
		// Updating existing file - must exist and generation must match
		if !fileExists {
			return 0, errors.New("file does not exist")
		}

		currentGeneration := info.ModTime().UnixNano()
		if currentGeneration != expectedGeneration {
			return 0, errors.New("generation mismatch")
		}
	}

	// Create subdirectories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, fmt.Errorf("failed to create directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	// Get new generation (file mtime)
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file after write: %w", err)
	}

	return info.ModTime().UnixNano(), nil
}

// GetSignedURL generates a file:// URL for accessing the object.
// The method and ttl parameters are ignored for local filesystem.
func (fs *FileStorage) GetSignedURL(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	filePath := filepath.Join(fs.dataDir, key)
	return "file://" + filePath, nil
}

// Close releases storage resources. This is a no-op for FileStorage.
func (fs *FileStorage) Close(_ context.Context) error {
	return nil
}
