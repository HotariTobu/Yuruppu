//go:build integration

package storage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	yuruppu_storage "yuruppu/internal/storage"
)

// requireGCSCredentials fails the test if required GCS credentials are not available.
func requireGCSCredentials(t *testing.T) string {
	t.Helper()
	bucket := os.Getenv("HISTORY_BUCKET")
	if bucket == "" {
		t.Fatal("HISTORY_BUCKET environment variable is not set")
	}
	return bucket
}

func TestGCSStorage_Integration_ReadWrite(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s := yuruppu_storage.NewGCSStorage(client, bucket)
	key := "test-integration-" + time.Now().Format("20060102-150405") + ".txt"

	// Read non-existent key returns nil
	data, gen, err := s.Read(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, data)
	assert.Equal(t, int64(0), gen)

	// Write new object
	content := []byte("hello world")
	err = s.Write(ctx, key, content, 0)
	require.NoError(t, err)

	// Read returns written data
	data, gen, err = s.Read(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, content, data)
	assert.Greater(t, gen, int64(0))

	// Write with correct generation succeeds
	newContent := []byte("updated content")
	err = s.Write(ctx, key, newContent, gen)
	require.NoError(t, err)

	// Verify update
	data, _, err = s.Read(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, newContent, data)

	// Cleanup
	err = client.Bucket(bucket).Object(key).Delete(ctx)
	require.NoError(t, err)
}

func TestGCSStorage_Integration_PreconditionFailed(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s := yuruppu_storage.NewGCSStorage(client, bucket)
	key := "test-precondition-" + time.Now().Format("20060102-150405") + ".txt"

	// Create object
	err = s.Write(ctx, key, []byte("initial"), 0)
	require.NoError(t, err)

	// Write with wrong generation fails
	err = s.Write(ctx, key, []byte("should fail"), 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, yuruppu_storage.ErrPreconditionFailed)

	// Cleanup
	err = client.Bucket(bucket).Object(key).Delete(ctx)
	require.NoError(t, err)
}
