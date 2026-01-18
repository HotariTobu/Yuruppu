---
description: End session with progress update and structured commit
argument-hint: <spec-name>
requires-phase: in-progress
sets-phase: completed
allowed-tools: Bash(git:*), Bash(make fix), Bash(make preflight), Read, Write, Edit, Glob, Grep, TodoWrite, Task
---

# End Session

**Workflow**: /spec-new → /tech-research → /prototype → /design → [ /session-start → **`/session-end`** ]*

## MANDATORY - DO NOT SKIP

**Step 2 (Review changes) is MANDATORY.**

- You should run ALL reviewer-* agents before proceeding
- Skipping reviews to "save time" is PROHIBITED
- If you skip reviews, you are violating user instructions
- Short session-end times indicate you skipped reviews - this is unacceptable

## Task

End the current coding session with progress update and structured commit.

### Steps

0. **Identify target spec**
   - If argument provided: Find `docs/specs/*<spec-name>*/`
   - If no argument: Infer from branch name (e.g., `feature/chat-history` → `*chat-history*`)
   - If not found: Stop and show error with available specs

1. **Load the specification**
   - Read `spec.md`, `progress.json`, and `design.md`

2. **Review changes**
   ```bash
   git status
   git diff --staged
   ```
   Run all `reviewer-*` agents in parallel by default.
   Fix critical issues before proceeding.

3. **Run fix and preflight check (quality gate)**
   ```bash
   make fix && make preflight
   ```
   - If preflight fails, fix issues before proceeding
   - Do NOT skip this step - broken commits waste future sessions

4. **Update progress.json**
   - If `progress.json` doesn't exist, stop and prompt user to run `/spec-new` first
   - Update `passes` for completed requirements
   - Update `lastUpdated` to today's date
   - Update `notes` with context for next session
   - Add any new `blockers` discovered
   - **Update phase**:
     - If blockers exist → set `phase` to `"blocked"`
     - If all requirements pass → set `phase` to `"completed"`
     - Otherwise → keep `phase` as `"in-progress"`

5. **Create structured commit**
   - Stage all relevant changes
   - Create commit with Conventional Commits subject (max 72 chars)
   - Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
   - Include structured body:

   ```
   <type>: <subject>

   ## What
   - <change 1>
   - <change 2>

   ## Why
   - <reason>

   ## Next
   - <next task 1>
   - <next task 2>

   ## Blockers
   - <blocker or "None">
   ```

6. **Verify clean state**
   ```bash
   git status
   ```

## Output Format

```
## Session End: <spec-name>

### Changes Made
- <summary of changes>

### Progress Updated
- Phase: <phase>
- Requirements passed: <X/Y>

### Commit Created
<commit hash and message>

### Next Session
- <what to do next>
```

## Guidelines

- Only mark requirements as `passes: true` if verified
- Keep commit subject under 72 characters
- Notes should be actionable for next session
- If tests are failing, do NOT mark requirements as passed
- Always include "Next session should start with..." in notes
- Leave the codebase in a state where the next session can start immediately

## Preventing Premature Completion

A requirement can only be marked as `passes: true` when:

1. `make preflight` passes
2. Manual verification completed (when applicable)
3. You have actually verified the behavior, not assumed it works

**"Probably works" = `passes: false`**

## Error Recovery

- **Preflight fails**: Fix issues before committing, do not skip
- **Reviewer finds critical issues**: Fix before proceeding, add to blockers if unresolvable
- **Partial progress**: Commit what works, update notes for next session
