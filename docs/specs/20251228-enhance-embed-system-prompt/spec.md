# Enhancement: Embed System Prompt from External File

## Overview

Enhance system prompt management by using Go's `embed` package to load the prompt from an external text file instead of a hardcoded string constant.

## Background & Purpose

Currently, the system prompt for the Yuruppu character is defined as a Go string constant in `internal/yuruppu/prompt.go`. This approach has limitations:

- Editing the prompt requires modifying Go code
- Long prompts reduce code readability
- Prompt content is mixed with code logic

Using `embed` allows:
- Separation of prompt content from code
- Easier prompt editing (plain text file)
- Better readability and maintainability
- Single binary deployment (no external file dependencies at runtime)

## Current Behavior

- `internal/yuruppu/prompt.go` defines `SystemPrompt` as a `const` string
- `internal/yuruppu/handler.go` uses `SystemPrompt` from the same package
- Two separate files exist for prompt definition and usage

## Proposed Changes

- [ ] CH-001: Create new directory `internal/yuruppu/prompt/` and add `system.txt` containing the system prompt text
- [ ] CH-002: Move `SystemPrompt` from `prompt.go` to `handler.go` as an embedded variable using `//go:embed` directive
- [ ] CH-003: Delete `internal/yuruppu/prompt.go` (no longer needed)
- [ ] CH-004: Verify all existing tests pass without modification

## Acceptance Criteria

### AC-001: System prompt loaded from text file [Linked to CH-001, CH-002]

- **Given**: The application is built with Go 1.16+
- **When**: The `handler.go` file is compiled
- **Then**:
  - `SystemPrompt` is declared as `var SystemPrompt string` with `//go:embed prompt/system.txt` directive
  - The prompt content is embedded at compile time (no runtime file access needed)
  - If `prompt/system.txt` is missing, build fails with Go's default embed error (pattern matches no files)

### AC-002: Prompt file contains character definition [Linked to CH-001]

- **Given**: `internal/yuruppu/prompt/system.txt` exists
- **When**: The file is read
- **Then**:
  - Content is identical to the original `SystemPrompt` constant (exact copy, preserving all formatting)
  - Contains "Yuruppu" character name
  - Contains personality traits and guidelines

### AC-003: prompt.go removed [Linked to CH-003]

- **Given**: The embed approach is implemented
- **When**: Checking the codebase
- **Then**:
  - `internal/yuruppu/prompt.go` no longer exists
  - `SystemPrompt` is defined only in `handler.go`
  - `SystemPrompt` remains exported (capital S) for test access

### AC-004: All existing tests pass [Linked to CH-004]

- **Given**: The refactoring is complete
- **When**: Running `go test ./...`
- **Then**:
  - All tests pass without any test code modifications
  - Tests access `yuruppu.SystemPrompt` as before (same package, same exported name)
  - Tests still verify `SystemPrompt` is non-empty and contains "Yuruppu"

## Implementation Notes

- Create directory: `internal/yuruppu/prompt/`
- Create file: `internal/yuruppu/prompt/system.txt` with exact content from current `SystemPrompt` constant
- In `handler.go`:
  - Add blank import `_ "embed"` (required when only using `//go:embed` directive without embed.FS)
  - Add directive and variable: `//go:embed prompt/system.txt` followed by `var SystemPrompt string`
- Delete `internal/yuruppu/prompt.go`
- Rationale: Since `SystemPrompt` is only used within the `yuruppu` package, consolidating it into `handler.go` eliminates an unnecessary separate file

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-28 | 1.0 | Initial version | - |
