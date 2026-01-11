# Design: storage-key-prefix

## Overview

Add key prefix support to GCSStorage and FileStorage, consolidate three GCS buckets into one.

## File Structure

| File | Purpose |
|------|---------|
| `internal/storage/gcs.go` | Add `keyPrefix` field, prepend to all operations |
| `cmd/cli/mock/storage.go` | Add `keyPrefix` field, use `filepath.Join(dataDir, prefix, key)` |
| `main.go` | Replace 3 bucket config fields with `BucketName` |
| `cmd/cli/main.go` | Update FileStorage calls |
| `infra/main.tf` | Single bucket, single env var |

## Interfaces

### GCSStorage

```go
type GCSStorage struct {
	bucket    *storage.BucketHandle
	keyPrefix string
}

func NewGCSStorage(client *storage.Client, bucketName, keyPrefix string) (*GCSStorage, error)
```

Key operations use string concatenation: `s.keyPrefix + key`

### FileStorage

```go
type FileStorage struct {
	dataDir string
}

func NewFileStorage(baseDataDir, keyPrefix string) *FileStorage {
	return &FileStorage{
		dataDir: filepath.Join(baseDataDir, keyPrefix),
	}
}
```

Key operations use: `filepath.Join(fs.dataDir, key)`

### Config

```go
type Config struct {
	// ... other fields ...
	BucketName string
}
```

Replaces `ProfileBucket`, `HistoryBucket`, `MediaBucket`.

## Data Flow

1. Application loads `BUCKET_NAME` from environment
2. Creates storage instances with bucket name and prefix
3. Storage prepends prefix to all key operations
4. GCS objects stored at `prefix/key` path

## Related

- [ADR: Comment Guidelines](../../adr/20260111-comment-guidelines.md)
