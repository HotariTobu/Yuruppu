# Fix: [Bug Description]

> Template for bug fixes.
> Filename: `yyyymmdd-fix-name.md`

## Overview

<!-- What bug is being fixed? (1-2 sentences) -->

## Current Behavior (Bug)

<!-- What is happening incorrectly? -->

## Expected Behavior

<!-- What should happen instead? -->

## Root Cause

<!-- What is causing the bug? -->

## Proposed Fix

<!-- How will the bug be fixed? Each fix item must have a unique ID for progress tracking -->

- [ ] FX-001: Fix description 1
- [ ] FX-002: Fix description 2

## Acceptance Criteria

<!--
Define acceptance criteria using Given-When-Then (GWT) format.
Each criterion must have a unique ID (AC-XXX) linked to a fix item (FX-XXX).
-->

### AC-001: [Linked to FX-001]

- **Given**: [Condition that previously caused the bug]
- **When**: [Action that triggered the bug]
- **Then**:
  - [Expected correct behavior]
  - Bug no longer occurs

### AC-002: [Linked to FX-001, Regression]

- **Given**: [Normal usage condition]
- **When**: [Related actions are performed]
- **Then**:
  - Existing functionality remains intact
  - No new bugs are introduced

## Implementation Notes

<!-- Notes for implementation, references, etc. -->

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| YYYY-MM-DD | 1.0 | Initial version | - |
