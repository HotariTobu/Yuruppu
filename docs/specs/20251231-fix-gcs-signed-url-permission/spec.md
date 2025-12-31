# Fix: GCS Signed URL Permission Error

## Overview

Fix the `iam.serviceAccounts.signBlob` permission error when generating signed URLs in Cloud Run environment.

## Background

The application uses signed URLs to provide the Gemini API with temporary access to user-uploaded images stored in GCS. When running in Cloud Run, the current implementation fails because it requires IAM permissions that are not granted by default.

## Current Behavior (Bug)

When the application generates signed URLs for GCS objects in Cloud Run:
1. The `storage.SignedURL()` method attempts to sign the URL using IAM credentials
2. Cloud Run's default service account lacks `iam.serviceAccounts.signBlob` permission
3. Error occurs: `Permission 'iam.serviceAccounts.signBlob' denied on resource`
4. Handler fails with "failed to get signed URLs for history"

## Expected Behavior

The application should generate signed URLs without requiring `iam.serviceAccounts.signBlob` permission by using the metadata server's signing mechanism available in Cloud Run.

## Root Cause

The current implementation uses `bucket.SignedURL()` without specifying `GoogleAccessID` and `SignBytes`. In this case, the GCS client attempts to use the IAM signBlob API, which requires additional permissions.

**Solution**: Use `storage.SignedURLOptions` with `SigningScheme: SigningSchemeV4`. When running on Google Cloud infrastructure (Cloud Run, GCE, etc.), the client can auto-detect credentials and use the metadata server for signing without requiring `iam.serviceAccounts.signBlob` permission.

## Acceptance Criteria

### AC-001: Signed URL Generation Works in Cloud Run

- **Given**: The application is running in Cloud Run with default service account
- **When**: A signed URL is requested for a GCS object
- **Then**:
  - `GetSignedURL` returns a URL string without error
  - The returned URL contains V4 signature parameters (`X-Goog-Algorithm`, `X-Goog-Credential`, `X-Goog-Signature`)
  - The URL can be used to access the object via HTTP (verified manually after deployment)

### AC-002: Signed URL Generation Works Locally with Service Account

- **Given**: The application is running locally with `GOOGLE_APPLICATION_CREDENTIALS` set to a service account key
- **When**: A signed URL is requested for a GCS object
- **Then**:
  - `GetSignedURL` returns a URL string without error
  - The existing integration test `TestGCSStorage_Integration_GetSignedURL` passes

### AC-003: No Regression in Storage Operations

- **Given**: The storage module is used for read/write operations
- **When**: Read, Write, or Close operations are performed
- **Then**:
  - All tests in `internal/storage/gcs_integration_test.go` pass without modification
  - No changes to read/write behavior

### AC-004: Error Handling for Non-Existent Objects

- **Given**: A signed URL is requested for a key that doesn't exist in GCS
- **When**: `GetSignedURL` is called
- **Then**:
  - The signed URL is still generated successfully (existence check is not performed during URL generation)
  - Access to the URL will return 404 from GCS (this is expected behavior)

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-31 | 1.0 | Initial version | - |
| 2025-12-31 | 1.1 | Added specific solution approach, testable criteria, and AC-004 | - |
