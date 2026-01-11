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
	bucket := os.Getenv("BUCKET_NAME")
	if bucket == "" {
		t.Fatal("BUCKET_NAME environment variable is not set")
	}
	return bucket
}

func TestGCSStorage_Integration_ReadWrite(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	key := "test-integration-" + time.Now().Format("20060102-150405") + ".txt"

	// Read non-existent key returns nil
	data, gen, err := s.Read(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, data)
	assert.Equal(t, int64(0), gen)

	// Write new object
	content := []byte("hello world")
	_, err = s.Write(ctx, key, "text/plain", content, 0)
	require.NoError(t, err)

	// Read returns written data
	data, gen, err = s.Read(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, content, data)
	assert.Greater(t, gen, int64(0))

	// Write with correct generation succeeds
	newContent := []byte("updated content")
	_, err = s.Write(ctx, key, "text/plain", newContent, gen)
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

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	key := "test-precondition-" + time.Now().Format("20060102-150405") + ".txt"

	// Create object
	_, err = s.Write(ctx, key, "text/plain", []byte("initial"), 0)
	require.NoError(t, err)

	// Write with wrong generation fails
	_, err = s.Write(ctx, key, "text/plain", []byte("should fail"), 99999)
	require.Error(t, err)

	// Cleanup
	err = client.Bucket(bucket).Object(key).Delete(ctx)
	require.NoError(t, err)
}

func TestGCSStorage_Integration_GetSignedURL(t *testing.T) {
	// Signed URLs require service_account, external_account, or impersonated_service_account credentials
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && os.Getenv("CI") == "" {
		t.Skip("Skipping: requires service account credentials (set GOOGLE_APPLICATION_CREDENTIALS or run in CI)")
	}

	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	key := "test-signedurl-" + time.Now().Format("20060102-150405") + ".txt"

	// Create test object
	content := []byte("signed url test content")
	_, err = s.Write(ctx, key, "text/plain", content, 0)
	require.NoError(t, err)

	// Generate signed URL for GET
	url, err := s.GetSignedURL(ctx, key, "GET", 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, bucket)
	assert.Contains(t, url, key)

	// Cleanup
	err = client.Bucket(bucket).Object(key).Delete(ctx)
	require.NoError(t, err)
}

func TestGCSStorage_Integration_NegativeGeneration(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	// Write with negative generation should fail
	_, err = s.Write(ctx, "test-key", "text/plain", []byte("data"), -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid expectedGeneration")
}

func TestGCSStorage_Integration_ConcurrentWrites(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	key := "test-concurrent-" + time.Now().Format("20060102-150405") + ".txt"

	// Create initial object
	gen, err := s.Write(ctx, key, "text/plain", []byte("initial"), 0)
	require.NoError(t, err)

	// Simulate concurrent writes with same expected generation
	// First write succeeds
	_, err1 := s.Write(ctx, key, "text/plain", []byte("update1"), gen)
	// Second write with same (now stale) generation fails
	_, err2 := s.Write(ctx, key, "text/plain", []byte("update2"), gen)

	require.NoError(t, err1)
	require.Error(t, err2) // Precondition failed

	// Cleanup
	err = client.Bucket(bucket).Object(key).Delete(ctx)
	require.NoError(t, err)
}

func TestGCSStorage_Integration_EmptyKey(t *testing.T) {
	bucket := requireGCSCredentials(t)
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	require.NoError(t, err)
	defer client.Close()

	s, err := yuruppu_storage.NewGCSStorage(client, bucket, "")
	require.NoError(t, err)

	// Write with empty key should fail (GCS rejects it)
	_, err = s.Write(ctx, "", "text/plain", []byte("data"), 0)
	require.Error(t, err)

	// Read with empty key should fail
	_, _, err = s.Read(ctx, "")
	require.Error(t, err)
}
