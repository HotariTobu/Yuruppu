# ADR: Comment Guidelines

> Date: 2026-01-11
> Status: **Adopted**

## Context

Comments that enumerate specific items (e.g., listing storage types, feature names) become maintenance burdens. When the code changes, these comments must also be updated, creating unnecessary coupling between code and documentation.

## Decision Drivers

- Comments should not require updates when code changes
- Reduce maintenance burden
- Keep comments focused on "why", not "what"

## Decision

Do not write comments that enumerate specific items that may change.

**Avoid:**
- Listing specific implementations, types, or features
- Summarizing "what" code does when code is self-explanatory

**Prefer:**
- Explaining "why" something is done
- Documenting non-obvious constraints or decisions

## Rationale

Enumeration comments create a coupling between code and comments. When code changes, comments become stale or require manual updates. This adds maintenance burden and risks misleading future readers when comments become outdated.

## Consequences

**Positive:**
- Less maintenance burden
- Comments stay accurate longer
- Forces better code organization (if code needs enumeration comments to be understood, the structure may need improvement)

**Negative:**
- May require more careful code organization to be self-documenting

## Related Decisions

None.
