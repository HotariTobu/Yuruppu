# Fix: Terraform Missing Vertex AI API Configuration

## Overview

Terraform configuration does not include Vertex AI API enablement and IAM roles, even though the application uses Vertex AI for LLM functionality. This causes infrastructure inconsistency and requires manual setup.

## Current Behavior (Bug)

- `aiplatform.googleapis.com` is not enabled in Terraform's `google_project_service.apis`
- Cloud Run service account lacks `roles/aiplatform.user` role
- Vertex AI must be manually enabled and configured outside of Terraform

## Expected Behavior

- Vertex AI API should be enabled via Terraform alongside other required APIs
- Cloud Run service account should have Vertex AI access permissions
- All infrastructure should be managed through Terraform for reproducibility

## Root Cause

Vertex AI integration was added after the initial Terraform configuration was created (see `docs/adr/20251224-llm-provider.md`), but Terraform was not updated to include the new service.

## Proposed Fix

- [x] FX-001: Add `aiplatform.googleapis.com` to `google_project_service.apis`
- [x] FX-002: Add `roles/aiplatform.user` IAM role to Cloud Run service account

## Acceptance Criteria

### AC-001: Vertex AI API Enabled [Linked to FX-001]

- **Given**: A fresh GCP project with Terraform applied
- **When**: `tofu apply` is executed
- **Then**:
  - `aiplatform.googleapis.com` API is enabled
  - No manual API enablement is required

### AC-002: Cloud Run Has Vertex AI Access [Linked to FX-002]

- **Given**: Terraform has been applied
- **When**: Cloud Run service invokes Vertex AI Gemini API
- **Then**:
  - Request succeeds with valid credentials
  - No permission denied errors occur

### AC-003: Existing Infrastructure Unchanged [Regression]

- **Given**: Existing deployed infrastructure
- **When**: Terraform is applied with these changes
- **Then**:
  - Existing Cloud Run, Secret Manager, and other resources remain intact
  - Only new API and IAM resources are added

## Implementation Notes

- API enable resource: `google_project_service`
- IAM role resource: `google_project_iam_member`
- Related ADR: `docs/adr/20251224-llm-provider.md`

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
