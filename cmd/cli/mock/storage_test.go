package mock_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	"yuruppu/cmd/cli/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStorage(t *testing.T) {
	t.Run("should create storage with data directory", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()

		// When
		storage := mock.NewFileStorage(dataDir, "")

		// Then
		require.NotNil(t, storage)
	})
}

func TestFileStorage_Read(t *testing.T) {
	t.Run("should return nil, 0, nil when key does not exist", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		// When
		data, generation, err := storage.Read(ctx, "profiles/nonexistent.json")

		// Then
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), generation)
	})

	t.Run("should return data and generation when key exists", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		// Create test file
		testData := []byte(`{"userId":"user123","name":"Test User"}`)
		filePath := filepath.Join(dataDir, "profiles", "user123.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, testData, 0o644))

		// Get expected generation (file mtime in unix nanoseconds)
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		expectedGeneration := info.ModTime().UnixNano()

		// When
		data, generation, err := storage.Read(ctx, "profiles/user123.json")

		// Then
		require.NoError(t, err)
		assert.Equal(t, testData, data)
		assert.Equal(t, expectedGeneration, generation)
	})

	t.Run("should handle nested subdirectories", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		testData := []byte("test content")
		filePath := filepath.Join(dataDir, "media", "images", "2024", "01", "image.jpg")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, testData, 0o644))

		// When
		data, _, err := storage.Read(ctx, "media/images/2024/01/image.jpg")

		// Then
		require.NoError(t, err)
		assert.Equal(t, testData, data)
	})
}

func TestFileStorage_Write(t *testing.T) {
	t.Run("should create new file when expectedGeneration is 0", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		testData := []byte(`{"userId":"user456","name":"New User"}`)

		// When
		newGeneration, err := storage.Write(ctx, "profiles/user456.json", "application/json", testData, 0)

		// Then
		require.NoError(t, err)
		assert.Greater(t, newGeneration, int64(0))

		// Verify file was created
		filePath := filepath.Join(dataDir, "profiles", "user456.json")
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, testData, content)

		// Verify generation matches file mtime
		info, statErr := os.Stat(filePath)
		require.NoError(t, statErr)
		assert.Equal(t, info.ModTime().UnixNano(), newGeneration)
	})

	t.Run("should return error when expectedGeneration is 0 but file already exists", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		// Create existing file
		filePath := filepath.Join(dataDir, "profiles", "user789.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, []byte("existing"), 0o644))

		// When
		newGeneration, err := storage.Write(ctx, "profiles/user789.json", "application/json", []byte("new data"), 0)

		// Then
		require.Error(t, err)
		assert.Equal(t, int64(0), newGeneration)
	})

	t.Run("should update file when expectedGeneration matches current generation", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		// Create existing file
		originalData := []byte("original data")
		filePath := filepath.Join(dataDir, "profiles", "user999.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, originalData, 0o644))

		// Get current generation
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		currentGeneration := info.ModTime().UnixNano()

		// Wait a bit to ensure mtime changes
		time.Sleep(10 * time.Millisecond)

		// When
		updatedData := []byte("updated data")
		newGeneration, err := storage.Write(ctx, "profiles/user999.json", "application/json", updatedData, currentGeneration)

		// Then
		require.NoError(t, err)
		assert.Greater(t, newGeneration, currentGeneration)

		// Verify file was updated
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, updatedData, content)
	})

	t.Run("should return error when expectedGeneration does not match", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		// Create existing file
		originalData := []byte("original data")
		filePath := filepath.Join(dataDir, "profiles", "user888.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, originalData, 0o644))

		// When
		wrongGeneration := int64(12345)
		newGeneration, err := storage.Write(ctx, "profiles/user888.json", "application/json", []byte("updated"), wrongGeneration)

		// Then
		require.Error(t, err)
		assert.Equal(t, int64(0), newGeneration)

		// Verify file was not modified
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, originalData, content)
	})

	t.Run("should create subdirectories automatically", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		testData := []byte("test content")

		// When
		_, err := storage.Write(ctx, "media/videos/2024/12/video.mp4", "video/mp4", testData, 0)

		// Then
		require.NoError(t, err)

		// Verify directory structure was created
		filePath := filepath.Join(dataDir, "media", "videos", "2024", "12", "video.mp4")
		_, statErr := os.Stat(filePath)
		require.NoError(t, statErr)
	})

	t.Run("should handle different mimetypes", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		tests := []struct {
			name     string
			key      string
			mimetype string
			data     []byte
		}{
			{
				name:     "JSON",
				key:      "profiles/user.json",
				mimetype: "application/json",
				data:     []byte(`{"key":"value"}`),
			},
			{
				name:     "PNG image",
				key:      "media/image.png",
				mimetype: "image/png",
				data:     []byte{0x89, 0x50, 0x4E, 0x47}, // PNG magic bytes
			},
			{
				name:     "Plain text",
				key:      "history/chat.txt",
				mimetype: "text/plain",
				data:     []byte("Hello, world!"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// When
				_, err := storage.Write(ctx, tt.key, tt.mimetype, tt.data, 0)

				// Then
				require.NoError(t, err)

				// Verify content
				filePath := filepath.Join(dataDir, tt.key)
				content, readErr := os.ReadFile(filePath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.data, content)
			})
		}
	})
}

func TestFileStorage_GetSignedURL(t *testing.T) {
	t.Run("should return file:// URL with absolute path", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "profiles/user123.json"

		// When
		url, err := storage.GetSignedURL(ctx, key, "GET", 1*time.Hour)

		// Then
		require.NoError(t, err)
		expectedPath := filepath.Join(dataDir, key)
		expectedURL := "file://" + expectedPath
		assert.Equal(t, expectedURL, url)
	})

	t.Run("should work for different HTTP methods", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()

		methods := []string{"GET", "PUT", "POST", "DELETE"}

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				// When
				url, err := storage.GetSignedURL(ctx, "test/file.txt", method, 1*time.Hour)

				// Then
				require.NoError(t, err)
				assert.Contains(t, url, "file://")
			})
		}
	})

	t.Run("should work for nested paths", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "media/images/2024/01/15/photo.jpg"

		// When
		url, err := storage.GetSignedURL(ctx, key, "GET", 1*time.Hour)

		// Then
		require.NoError(t, err)
		expectedPath := filepath.Join(dataDir, key)
		expectedURL := "file://" + expectedPath
		assert.Equal(t, expectedURL, url)
	})

	t.Run("should ignore TTL parameter for local filesystem", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "test.txt"

		// When
		url1, err1 := storage.GetSignedURL(ctx, key, "GET", 1*time.Minute)
		url2, err2 := storage.GetSignedURL(ctx, key, "GET", 24*time.Hour)

		// Then
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, url1, url2) // URLs should be identical regardless of TTL
	})
}

func TestFileStorage_KeyPrefix_Read(t *testing.T) {
	// AC-002: Key prefix is applied to Read operations
	t.Run("should prepend prefix to read path", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "history/")
		ctx := context.Background()

		// Create test file with prefix applied
		testData := []byte(`{"userId":"user123","messages":["hello"]}`)
		filePath := filepath.Join(dataDir, "history", "user123.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
		require.NoError(t, os.WriteFile(filePath, testData, 0o644))

		// When
		data, generation, err := storage.Read(ctx, "user123.json")

		// Then
		require.NoError(t, err)
		assert.Equal(t, testData, data)
		assert.Greater(t, generation, int64(0))
	})

	t.Run("should handle different prefixes", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		ctx := context.Background()

		tests := []struct {
			name      string
			keyPrefix string
			key       string
			data      []byte
		}{
			{
				name:      "history prefix",
				keyPrefix: "history/",
				key:       "user123.json",
				data:      []byte(`{"history":"data"}`),
			},
			{
				name:      "profile prefix",
				keyPrefix: "profile/",
				key:       "user456.json",
				data:      []byte(`{"profile":"data"}`),
			},
			{
				name:      "media prefix",
				keyPrefix: "media/",
				key:       "image.png",
				data:      []byte{0x89, 0x50, 0x4E, 0x47},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given
				storage := mock.NewFileStorage(dataDir, tt.keyPrefix)

				// Create test file with prefix applied
				filePath := filepath.Join(dataDir, tt.keyPrefix+tt.key)
				require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
				require.NoError(t, os.WriteFile(filePath, tt.data, 0o644))

				// When
				data, _, err := storage.Read(ctx, tt.key)

				// Then
				require.NoError(t, err)
				assert.Equal(t, tt.data, data)
			})
		}
	})

	t.Run("should return nil when key with prefix does not exist", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "history/")
		ctx := context.Background()

		// When
		data, generation, err := storage.Read(ctx, "nonexistent.json")

		// Then
		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), generation)
	})
}

func TestFileStorage_KeyPrefix_Write(t *testing.T) {
	// AC-003: Key prefix is applied to Write operations
	t.Run("should prepend prefix to write path", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "history/")
		ctx := context.Background()
		testData := []byte(`{"userId":"user123","messages":["hello"]}`)

		// When
		generation, err := storage.Write(ctx, "user123.json", "application/json", testData, 0)

		// Then
		require.NoError(t, err)
		assert.Greater(t, generation, int64(0))

		// Verify file was created at correct path with prefix
		filePath := filepath.Join(dataDir, "history", "user123.json")
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, testData, content)
	})

	t.Run("should handle different prefixes", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		ctx := context.Background()

		tests := []struct {
			name      string
			keyPrefix string
			key       string
			data      []byte
			mimetype  string
		}{
			{
				name:      "history prefix",
				keyPrefix: "history/",
				key:       "user123.json",
				data:      []byte(`{"history":"data"}`),
				mimetype:  "application/json",
			},
			{
				name:      "profile prefix",
				keyPrefix: "profile/",
				key:       "user456.json",
				data:      []byte(`{"profile":"data"}`),
				mimetype:  "application/json",
			},
			{
				name:      "media prefix",
				keyPrefix: "media/",
				key:       "image.png",
				data:      []byte{0x89, 0x50, 0x4E, 0x47},
				mimetype:  "image/png",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given
				storage := mock.NewFileStorage(dataDir, tt.keyPrefix)

				// When
				_, err := storage.Write(ctx, tt.key, tt.mimetype, tt.data, 0)

				// Then
				require.NoError(t, err)

				// Verify file was created at correct path with prefix
				filePath := filepath.Join(dataDir, tt.keyPrefix+tt.key)
				content, readErr := os.ReadFile(filePath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.data, content)
			})
		}
	})

	t.Run("should create subdirectories with prefix automatically", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "media/")
		ctx := context.Background()
		testData := []byte("video content")

		// When
		_, err := storage.Write(ctx, "videos/2024/12/video.mp4", "video/mp4", testData, 0)

		// Then
		require.NoError(t, err)

		// Verify directory structure with prefix was created
		filePath := filepath.Join(dataDir, "media", "videos", "2024", "12", "video.mp4")
		_, statErr := os.Stat(filePath)
		require.NoError(t, statErr)
	})

	t.Run("should support generation checking with prefix", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "history/")
		ctx := context.Background()
		key := "user999.json"

		// Create initial file
		initialData := []byte("version 1")
		gen1, err := storage.Write(ctx, key, "application/json", initialData, 0)
		require.NoError(t, err)

		// Wait to ensure mtime changes
		time.Sleep(10 * time.Millisecond)

		// When - update with correct generation
		updatedData := []byte("version 2")
		gen2, err := storage.Write(ctx, key, "application/json", updatedData, gen1)

		// Then
		require.NoError(t, err)
		assert.Greater(t, gen2, gen1)

		// Verify file at prefixed path was updated
		filePath := filepath.Join(dataDir, "history", key)
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, updatedData, content)
	})
}

func TestFileStorage_KeyPrefix_GetSignedURL(t *testing.T) {
	// AC-004: Key prefix is applied to GetSignedURL operations
	t.Run("should prepend prefix to URL path", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "media/")
		ctx := context.Background()
		key := "image.png"

		// When
		url, err := storage.GetSignedURL(ctx, key, "GET", 1*time.Hour)

		// Then
		require.NoError(t, err)
		expectedPath := filepath.Join(dataDir, "media", "image.png")
		expectedURL := "file://" + expectedPath
		assert.Equal(t, expectedURL, url)
	})

	t.Run("should handle different prefixes", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		ctx := context.Background()

		tests := []struct {
			name      string
			keyPrefix string
			key       string
		}{
			{
				name:      "history prefix",
				keyPrefix: "history/",
				key:       "user123.json",
			},
			{
				name:      "profile prefix",
				keyPrefix: "profile/",
				key:       "user456.json",
			},
			{
				name:      "media prefix",
				keyPrefix: "media/",
				key:       "image.png",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given
				storage := mock.NewFileStorage(dataDir, tt.keyPrefix)

				// When
				url, err := storage.GetSignedURL(ctx, tt.key, "GET", 1*time.Hour)

				// Then
				require.NoError(t, err)
				expectedPath := filepath.Join(dataDir, tt.keyPrefix+tt.key)
				expectedURL := "file://" + expectedPath
				assert.Equal(t, expectedURL, url)
			})
		}
	})

	t.Run("should work with nested paths and prefix", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "media/")
		ctx := context.Background()
		key := "images/2024/01/15/photo.jpg"

		// When
		url, err := storage.GetSignedURL(ctx, key, "GET", 1*time.Hour)

		// Then
		require.NoError(t, err)
		expectedPath := filepath.Join(dataDir, "media", "images", "2024", "01", "15", "photo.jpg")
		expectedURL := "file://" + expectedPath
		assert.Equal(t, expectedURL, url)
	})
}

func TestFileStorage_KeyPrefix_EmptyPrefix(t *testing.T) {
	// AC-005: Empty prefix is supported
	t.Run("should use keys as-is when prefix is empty", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "user123.json"
		testData := []byte(`{"test":"data"}`)

		// When - write with empty prefix
		generation, err := storage.Write(ctx, key, "application/json", testData, 0)
		require.NoError(t, err)

		// Then - file should be at root of dataDir without any prefix
		filePath := filepath.Join(dataDir, key)
		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, testData, content)

		// When - read with empty prefix
		data, gen, err := storage.Read(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, testData, data)
		assert.Equal(t, generation, gen)

		// When - get signed URL with empty prefix
		url, err := storage.GetSignedURL(ctx, key, "GET", 1*time.Hour)
		require.NoError(t, err)
		expectedURL := "file://" + filePath
		assert.Equal(t, expectedURL, url)
	})

	t.Run("should not add extra slashes when prefix is empty", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "profiles/user456.json"

		// When
		_, err := storage.Write(ctx, key, "application/json", []byte("data"), 0)

		// Then - file should be at profiles/user456.json, not /profiles/user456.json
		require.NoError(t, err)
		filePath := filepath.Join(dataDir, "profiles", "user456.json")
		_, statErr := os.Stat(filePath)
		require.NoError(t, statErr)

		// Verify no extra directory was created
		wrongPath := filepath.Join(dataDir, "", "profiles", "user456.json")
		assert.Equal(t, filePath, wrongPath) // Should resolve to same path
	})
}

func TestFileStorage_IntegrationScenario(t *testing.T) {
	t.Run("should support read-modify-write cycle with generation checking", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "profiles/user777.json"

		// Create initial file
		initialData := []byte(`{"userId":"user777","score":100}`)
		gen1, err := storage.Write(ctx, key, "application/json", initialData, 0)
		require.NoError(t, err)
		require.Greater(t, gen1, int64(0))

		// Read the file
		data, gen2, err := storage.Read(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, initialData, data)
		assert.Equal(t, gen1, gen2)

		// Wait to ensure mtime changes
		time.Sleep(10 * time.Millisecond)

		// Update with correct generation
		updatedData := []byte(`{"userId":"user777","score":200}`)
		gen3, err := storage.Write(ctx, key, "application/json", updatedData, gen2)
		require.NoError(t, err)
		assert.Greater(t, gen3, gen2)

		// Verify update
		finalData, gen4, err := storage.Read(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, updatedData, finalData)
		assert.Equal(t, gen3, gen4)

		// Try to update with stale generation (should fail)
		_, err = storage.Write(ctx, key, "application/json", []byte("stale update"), gen2)
		require.Error(t, err)
	})

	t.Run("should handle concurrent-like updates with generation mismatch", func(t *testing.T) {
		// Given
		dataDir := t.TempDir()
		storage := mock.NewFileStorage(dataDir, "")
		ctx := context.Background()
		key := "profiles/concurrent.json"

		// Writer A: Create initial file
		gen1, err := storage.Write(ctx, key, "application/json", []byte("version 1"), 0)
		require.NoError(t, err)

		// Writer B: Read current state
		_, genB, err := storage.Read(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, gen1, genB)

		// Wait to ensure mtime changes
		time.Sleep(10 * time.Millisecond)

		// Writer A: Update (succeeds)
		gen2, err := storage.Write(ctx, key, "application/json", []byte("version 2"), gen1)
		require.NoError(t, err)
		assert.Greater(t, gen2, gen1)

		// Writer B: Try to update with stale generation (fails)
		_, err = storage.Write(ctx, key, "application/json", []byte("version B"), genB)
		require.Error(t, err)

		// Verify file contains Writer A's update
		data, _, err := storage.Read(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, []byte("version 2"), data)
	})
}
