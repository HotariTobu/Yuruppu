package mock

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// FileStorage implements storage.Storage interface using local filesystem.
type FileStorage struct {
	baseDir string
}

// NewFileStorage creates a new FileStorage with the given base directory.
func NewFileStorage(baseDir string) *FileStorage {
	return &FileStorage{baseDir: baseDir}
}

// Read reads data from a file. Generation is derived from file modification time.
func (s *FileStorage) Read(ctx context.Context, key string) ([]byte, int64, error) {
	path := filepath.Join(s.baseDir, key)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, err
		}
		return nil, 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	generation := info.ModTime().UnixNano()
	return data, generation, nil
}

// Write writes data to a file. Uses file modification time as generation.
func (s *FileStorage) Write(ctx context.Context, key, mimetype string, data []byte, expectedGeneration int64) (int64, error) {
	path := filepath.Join(s.baseDir, key)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	// Check generation if expectedGeneration > 0
	if expectedGeneration > 0 {
		info, err := os.Stat(path)
		if err == nil {
			currentGen := info.ModTime().UnixNano()
			if currentGen != expectedGeneration {
				return 0, errors.New("generation mismatch")
			}
		}
	}

	// Write to temp file and rename for atomicity
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return 0, err
	}

	// Get new generation
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.ModTime().UnixNano(), nil
}

// GetSignedURL returns a file:// URL for local files.
func (s *FileStorage) GetSignedURL(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	path := filepath.Join(s.baseDir, key)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return "file://" + absPath, nil
}

// Close is a no-op for file storage.
func (s *FileStorage) Close(ctx context.Context) error {
	return nil
}

// Helper to encode int64 to bytes (not used but kept for reference)
func int64ToBytes(n int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(n))
	return b
}

// Helper to decode bytes to int64 (not used but kept for reference)
func bytesToInt64(b []byte) int64 {
	if len(b) < 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(b))
}
