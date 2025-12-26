# Fix: Integration Test Issues

> Fix multiple issues with integration test setup and tracking.

## Overview

Fix integrity issues in the integration test system: false progress tracking, spec violations, and test performance problems.

## Current Behavior (Bug)

1. **Missing LINE integration tests**: `progress.json` for `20251226-refact-integration-test-strategy` claims SC-002 (LINE API integration tests) is complete, but `internal/line/line_integration_test.go` does not exist.

2. **Spec violation in test code**: `vertexai_integration_test.go` uses hardcoded default region `us-central1`, violating `20251226-fix-env-config-cleanup` spec which states "No default values for GCP config".

3. **Slow CI due to real timers**: `TestGetRegion_Timeout` and `TestGetProjectID_Timeout` use real time delays (~15+ seconds total), running in every CI build.

## Expected Behavior

1. LINE integration tests exist and progress tracking is accurate.
2. Test code follows the same spec requirements as production code.
3. CI runs efficiently with fake timers.

## Root Cause

1. **Missing tests**: LINE integration tests were never implemented despite progress.json claiming completion.
2. **Spec violation**: Integration test was written without checking config spec requirements.
3. **Real timers**: Tests use `time.Sleep` instead of fake timers.

## Proposed Fix

- [ ] FX-001: Create `internal/line/line_integration_test.go` with LINE API integration tests (note: original spec incorrectly specified `internal/bot/` which does not exist)
- [ ] FX-002: Remove default region fallback and skip logic from `vertexai_integration_test.go`, fail with error when `GCP_PROJECT_ID` or `GCP_REGION` is not set
- [ ] FX-003: Replace real timers with fake timers in `TestGetRegion_Timeout` and `TestGetProjectID_Timeout` using `github.com/benbjohnson/clock`

## Acceptance Criteria

### AC-001: LINE integration tests exist [Linked to FX-001]

- **Given**: `internal/line/line_integration_test.go` exists
- **When**: Running `make test-integration` with `LINE_CHANNEL_SECRET` and `LINE_CHANNEL_ACCESS_TOKEN` set
- **Then**:
  - LINE API integration tests execute
  - Tests verify actual LINE API connectivity

### AC-002: Integration test fails without credentials [Linked to FX-002]

- **Given**: `GCP_PROJECT_ID` or `GCP_REGION` is not set
- **When**: Running `make test-integration`
- **Then**:
  - Test output contains error message indicating missing environment variable
  - Test exits with non-zero status (not skip)

**Code verification**:
- `vertexai_integration_test.go` does not contain `t.Skip()` calls for missing credentials
- `vertexai_integration_test.go` does not contain fallback region logic
- The string `us-central1` does not appear in `vertexai_integration_test.go`

### AC-003: Timeout tests use fake timers [Linked to FX-003]

- **Given**: Running `make test` (regular CI)
- **When**: Tests complete
- **Then**:
  - `TestGetRegion_Timeout` completes in under 1 second (total for all subtests)
  - `TestGetProjectID_Timeout` completes in under 1 second (total for all subtests)
  - Test behavior is deterministic (no flaky timing issues)

## Implementation Notes

- FX-001: Refer to `20251226-refact-integration-test-strategy` spec for LINE integration test requirements (AC-004: `GetBotInfo()` returns bot information)
- FX-003: Inject `clock.Clock` interface into functions that use `time.After` or `time.Sleep`

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-27 | 1.0 | Initial version | - |
| 2025-12-27 | 1.1 | Revised based on user feedback: implement missing tests instead of updating progress, fail instead of skip, no -v flag, use fake timers | - |
| 2025-12-27 | 1.2 | FX-002/AC-002: Also fail for missing GCP_PROJECT_ID (not just GCP_REGION) | - |
