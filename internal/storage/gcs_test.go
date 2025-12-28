//go:build !integration

package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	yuruppu_storage "yuruppu/internal/storage"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GCS Storage Constructor Tests
// =============================================================================

func TestNewGCSStorageWithBucket(t *testing.T) {
	t.Run("should create GCS storage with valid bucket", func(t *testing.T) {
		mockBucket := &mockBucketHandle{}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)
		assert.NotNil(t, gcsStorage)
	})
}

// =============================================================================
// Read Tests - Happy Path
// =============================================================================

func TestGCSStorage_Read_NotFound(t *testing.T) {
	t.Run("should return nil and zero generation when key does not exist", func(t *testing.T) {
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"test-key": {notFound: true},
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		data, generation, err := gcsStorage.Read(ctx, "test-key")

		require.NoError(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), generation)
	})
}

func TestGCSStorage_Read_ValidData(t *testing.T) {
	t.Run("should return data and generation when key exists", func(t *testing.T) {
		expectedData := "test data content"
		expectedGeneration := int64(12345)
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"test-key": {
					data:       expectedData,
					generation: expectedGeneration,
				},
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		data, generation, err := gcsStorage.Read(ctx, "test-key")

		require.NoError(t, err)
		assert.Equal(t, []byte(expectedData), data)
		assert.Equal(t, expectedGeneration, generation)
	})

	t.Run("should handle empty data with generation", func(t *testing.T) {
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"empty-key": {
					data:       "",
					generation: 100,
				},
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		data, generation, err := gcsStorage.Read(ctx, "empty-key")

		require.NoError(t, err)
		assert.Equal(t, []byte{}, data)
		assert.Equal(t, int64(100), generation)
	})
}

// =============================================================================
// Read Tests - Error Cases
// =============================================================================

func TestGCSStorage_Read_Error(t *testing.T) {
	t.Run("should return error when GCS attrs fails", func(t *testing.T) {
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"error-key": {
					attrsError: errors.New("permission denied"),
				},
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		data, generation, err := gcsStorage.Read(ctx, "error-key")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), generation)
		assert.Contains(t, err.Error(), "failed to get attrs")
	})

	t.Run("should return error when GCS read fails", func(t *testing.T) {
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"error-key": {
					generation: 1,
					readError:  errors.New("read failed"),
				},
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		data, generation, err := gcsStorage.Read(ctx, "error-key")

		require.Error(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), generation)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

// =============================================================================
// Write Tests - Happy Path
// =============================================================================

func TestGCSStorage_Write_NewKey(t *testing.T) {
	t.Run("should create new object with DoesNotExist precondition when generation is 0", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"new-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		err := gcsStorage.Write(ctx, "new-key", []byte("new data"), 0)

		require.NoError(t, err)
		assert.Equal(t, "new data", mockObj.writtenData.String())
		assert.True(t, mockObj.usedDoesNotExist, "should use DoesNotExist precondition")
	})

	t.Run("should update with GenerationMatch precondition when generation > 0", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			data:        "old data",
			generation:  12345,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"existing-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		err := gcsStorage.Write(ctx, "existing-key", []byte("new data"), 12345)

		require.NoError(t, err)
		assert.Equal(t, "new data", mockObj.writtenData.String())
		assert.Equal(t, int64(12345), mockObj.usedGenerationMatch, "should use GenerationMatch precondition")
	})

	t.Run("should overwrite unconditionally when generation < 0", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			data:        "old data",
			generation:  999,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"overwrite-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		err := gcsStorage.Write(ctx, "overwrite-key", []byte("new data"), -1)

		require.NoError(t, err)
		assert.Equal(t, "new data", mockObj.writtenData.String())
		assert.False(t, mockObj.usedDoesNotExist)
		assert.Equal(t, int64(0), mockObj.usedGenerationMatch)
	})
}

// =============================================================================
// Write Tests - Error Cases
// =============================================================================

func TestGCSStorage_Write_Error(t *testing.T) {
	t.Run("should return error when GCS write fails", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			writeError: errors.New("write permission denied"),
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"error-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		err := gcsStorage.Write(ctx, "error-key", []byte("data"), -1)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write")
	})

	t.Run("should return error when precondition fails", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			generation:        100,
			preconditionError: errors.New("precondition failed: generation mismatch"),
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"conflict-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		// Try to write with wrong generation
		err := gcsStorage.Write(ctx, "conflict-key", []byte("data"), 50)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "precondition")
	})
}

// =============================================================================
// Close Tests
// =============================================================================

func TestGCSStorage_Close(t *testing.T) {
	t.Run("should close without error", func(t *testing.T) {
		mockBucket := &mockBucketHandle{}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		ctx := context.Background()
		err := gcsStorage.Close(ctx)

		require.NoError(t, err)
	})
}

// =============================================================================
// Round Trip Tests
// =============================================================================

func TestGCSStorage_RoundTrip(t *testing.T) {
	t.Run("should preserve data through write-read cycle with generation", func(t *testing.T) {
		mockObj := &mockObjectHandle{
			notFound:    true,
			writtenData: &bytes.Buffer{},
		}
		mockBucket := &mockBucketHandle{
			objects: map[string]*mockObjectHandle{
				"test-key": mockObj,
			},
		}
		gcsStorage := yuruppu_storage.NewGCSStorageWithBucket(mockBucket)

		originalData := []byte("test data for round trip")
		ctx := context.Background()

		// Write data (new object)
		err := gcsStorage.Write(ctx, "test-key", originalData, 0)
		require.NoError(t, err)

		// Simulate GCS persistence
		mockObj.data = mockObj.writtenData.String()
		mockObj.notFound = false
		mockObj.generation = 1

		// Read back should return same data with generation
		readData, generation, err := gcsStorage.Read(ctx, "test-key")
		require.NoError(t, err)
		assert.Equal(t, originalData, readData)
		assert.Equal(t, int64(1), generation)
	})
}

// =============================================================================
// Mock Implementations
// =============================================================================

type mockBucketHandle struct {
	objects map[string]*mockObjectHandle
}

func (m *mockBucketHandle) Object(name string) yuruppu_storage.ObjectHandle {
	if obj, ok := m.objects[name]; ok {
		obj.name = name
		return obj
	}
	return &mockObjectHandle{name: name, notFound: true}
}

type mockObjectHandle struct {
	name       string
	data       string
	notFound   bool
	generation int64

	// Error simulation
	attrsError        error
	readError         error
	writeError        error
	closeError        error
	preconditionError error

	// Write tracking
	writtenData         *bytes.Buffer
	usedDoesNotExist    bool
	usedGenerationMatch int64
}

func (m *mockObjectHandle) NewReader(ctx context.Context) (io.ReadCloser, error) {
	if m.notFound {
		return nil, storage.ErrObjectNotExist
	}
	if m.readError != nil {
		return nil, m.readError
	}
	return io.NopCloser(strings.NewReader(m.data)), nil
}

func (m *mockObjectHandle) NewWriter(ctx context.Context) io.WriteCloser {
	if m.writtenData == nil {
		m.writtenData = &bytes.Buffer{}
	}
	return &mockWriter{
		obj:    m,
		buffer: m.writtenData,
	}
}

func (m *mockObjectHandle) Attrs(ctx context.Context) (*storage.ObjectAttrs, error) {
	if m.notFound {
		return nil, storage.ErrObjectNotExist
	}
	if m.attrsError != nil {
		return nil, m.attrsError
	}
	return &storage.ObjectAttrs{
		Generation: m.generation,
	}, nil
}

func (m *mockObjectHandle) Generation(gen int64) yuruppu_storage.ObjectHandle {
	return m
}

func (m *mockObjectHandle) If(conds storage.Conditions) yuruppu_storage.ObjectHandle {
	if conds.DoesNotExist {
		m.usedDoesNotExist = true
	}
	if conds.GenerationMatch != 0 {
		m.usedGenerationMatch = conds.GenerationMatch
	}
	return m
}

type mockWriter struct {
	obj    *mockObjectHandle
	buffer *bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (int, error) {
	if w.obj.writeError != nil {
		return 0, w.obj.writeError
	}
	return w.buffer.Write(p)
}

func (w *mockWriter) Close() error {
	if w.obj.closeError != nil {
		return w.obj.closeError
	}
	if w.obj.preconditionError != nil {
		return w.obj.preconditionError
	}
	if w.obj.writeError != nil {
		return w.obj.writeError
	}
	return nil
}
