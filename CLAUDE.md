# CLAUDE.md

> **Rule**: Do not add anything to CLAUDE.md unless it is necessary.

## Project Overview

Yuruppu is a LINE bot that responds as the character "Yuruppu". Written in Go.

## Language

All documentation, code comments, commit messages, and issues must be written in **English**.

Exception: Specification content (`docs/specs/*.md`) may be written in other languages.

## Directory Structure

```
docs/
  adr/              # Architecture Decision Records
  llms-txt/         # LLM documentation for libraries (LINE SDK, etc.)
  specs/            # Feature specifications (spec-driven development)
    templates/      # Spec templates (FEATURE, ENHANCEMENT, FIX, REFACTOR)
```

## Spec-Driven Development

This repository follows **Spec-Driven Development**.

1. **Write specs before code** - Always create a specification before implementation
2. **Keep change history** - Update the history section when modifying specs
3. **Follow the spec** - Do not implement features not described in the spec
4. **Derive tests from specs** - Test cases should be based on spec requirements

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/<feature-name>
```

### 2. Create a Specification

Before writing any code, create a specification in `docs/specs/`:

```
/spec-new <feature-name>
```

### 3. Review the Specification

Ensure the specification is complete and correct:

```
Review docs/specs/<spec-name>/spec.md using spec-reviewer
```

### 4. Tech Selection (if needed)

Write an ADR (`docs/adr/`) when:
- Adding or replacing a library/framework
- Choosing between multiple implementation approaches
- Adopting a new pattern or convention

Use the `tech-stack-adr` skill to guide technology selection and create an ADR.

### 5. Implement (Session Workflow)

Each session focuses on **one requirement**. Repeat until all requirements are complete.

```
/session-start <spec-name>
# ... implement and test one requirement ...
/session-end
```

**First session** (or when introducing new libraries/tools):

Generate LLM documentation before implementation using the `llms-generator` agent:
```
Generate llms.txt from https://example.com/docs
```

**Every session** (TDD):

1. Write tests based on spec requirements
2. Run tests (should fail):
   ```bash
   make test
   ```
3. Implement to make tests pass
4. Run tests again (should pass):
   ```bash
   make test
   ```

### 6. Check Code

Run linting and formatting checks:

```bash
make check
```

### 7. Create a Pull Request

- Link to the related specification
- Ensure all tests pass
- Request a review

## Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Feature | `feature/<name>` | `feature/spotify-adapter` |
| Bug fix | `fix/<name>` | `fix/auth-error` |
| Documentation | `docs/<name>` | `docs/api-guide` |
| Refactoring | `refactor/<name>` | `refactor/error-handling` |

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/).

### Format

```
<type>: <description>

[optional body]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `refactor` | Code refactoring |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks |

### Body Format (for feat/fix)

```
feat: add rate limit handling to Spotify adapter

## What
- Implemented exponential backoff for 429 responses
- Added RetryableError class

## Why
- Spotify API has strict rate limits (180 req/min)

## Next
- Add unit tests for retry logic
- Handle 503 Service Unavailable

## Blockers
- None
```

This format ensures session continuity by documenting what was done and what comes next.

## Pull Request Guidelines

1. **Link the specification** - Every PR must reference its specification
2. **One feature per PR** - Keep PRs focused and small
3. **All tests must pass** - Ensure CI is green
4. **Request a review** - Wait for approval before merging

### Preflight Check

Run `make preflight` to verify codebase health. Used by CI, `/session-start`, and `/session-end`.

### Preventing Context Exhaustion

- Focus on **ONE requirement per session**
- If a requirement is too large, split into sub-tasks and restart session planning
- Always read existing code with Read tool before writing new code
- Do not attempt to implement multiple features at once

#### Requirement Size Guidelines

A requirement is **TOO LARGE** if:
- Implementation touches more than 3 files
- Expected to take more than 50 tool calls
- Contains "and" connecting distinct features

**Split strategy:**
1. Identify sub-tasks
2. Create temporary sub-requirements (e.g., FR-001a, FR-001b)
3. Complete each in separate session
4. Mark parent requirement as passed only when all sub-tasks complete

### Test Integrity

- **NEVER delete or modify existing tests** to make them pass
- If a test fails, fix the implementation, not the test
- Exception: Test is genuinely incorrect (document reason in commit)
- New functionality must have corresponding new tests

### Preventing Premature Completion

A requirement can only be marked as `passes: true` when:

1. `make preflight` passes
2. Manual verification completed (when applicable)
3. You have actually verified the behavior, not assumed it works

**"Probably works" = `passes: false`**

### Recovery from Failed Sessions

1. `git stash` or `git reset --soft HEAD~1` to preserve work
2. Run `/session-start` to re-orient
3. Document what went wrong in blockers
4. Split the failed requirement into smaller pieces
