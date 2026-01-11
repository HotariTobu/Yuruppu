# Refactor: Add Key Prefix Support to GCSStorage

## Overview

Add key prefix support to `GCSStorage` initialization, enabling consolidation of multiple GCS buckets into a single bucket.

## Background & Purpose

Currently, the application uses three separate GCS buckets for different data types:
- `HISTORY_BUCKET` - Chat history storage
- `PROFILE_BUCKET` - User profile storage
- `MEDIA_BUCKET` - Media file storage

This refactoring adds a key prefix parameter to `NewGCSStorage`, allowing all data to be stored in a single bucket with prefixes like `history/`, `profile/`, and `media/`. This simplifies infrastructure management by reducing the number of buckets to manage.

**Constraints**:
- No data migration required (no existing data)
- No backward compatibility required

## Breaking Changes

- `NewGCSStorage(client, bucketName)` signature changes to `NewGCSStorage(client, bucketName, keyPrefix)`
- Configuration changes from three bucket environment variables to one:
  - `HISTORY_BUCKET`, `PROFILE_BUCKET`, `MEDIA_BUCKET` → `BUCKET_NAME`
- Prefixes are hardcoded: `history/`, `profile/`, `media/`
- Terraform configuration changes from three buckets to one

## Acceptance Criteria

### AC-001: GCSStorage accepts key prefix parameter

- **Given**: A GCS client and bucket name
- **When**: `NewGCSStorage` is called with a key prefix (e.g., `"history/"`)
- **Then**: The storage instance prepends the prefix to all key operations (simple string concatenation: `prefix + key`)

### AC-002: Key prefix is applied to Read operations

- **Given**: A GCSStorage instance with prefix `"history/"`
- **When**: `Read(ctx, "user123.json")` is called
- **Then**: The actual GCS object path is `"history/user123.json"`

### AC-003: Key prefix is applied to Write operations

- **Given**: A GCSStorage instance with prefix `"history/"`
- **When**: `Write(ctx, "user123.json", ...)` is called
- **Then**: The object is written to `"history/user123.json"`

### AC-004: Key prefix is applied to GetSignedURL operations

- **Given**: A GCSStorage instance with prefix `"media/"`
- **When**: `GetSignedURL(ctx, "image.png", ...)` is called
- **Then**: The signed URL points to `"media/image.png"`

### AC-005: Empty prefix is supported

- **Given**: A GCS client and bucket name
- **When**: `NewGCSStorage` is called with an empty prefix `""`
- **Then**: Keys are used as-is without any prefix

### AC-006: Application uses single bucket with hardcoded prefixes

- **Given**: The application is configured with a single bucket
- **When**: Storage instances are created for history, profile, and media
- **Then**: All three GCSStorage instances share the same bucket with hardcoded prefixes (`history/`, `profile/`, `media/`)

### AC-007: Terraform defines single bucket

- **Given**: Terraform configuration
- **When**: Infrastructure is applied
- **Then**: A single GCS bucket is created (replacing three separate buckets)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-11 | 1.0 | Initial version | - |
