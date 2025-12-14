---
description: End session with progress update and structured commit
allowed-tools: Bash(git *), Bash(make preflight), Read, Edit, Write, Glob, TodoWrite
---

# End Session

## Task

End the current coding session with progress update and structured commit.

### Steps

1. **Identify worked spec**
   - Infer from current branch name
   - Or ask user if unclear

2. **Review changes**
   ```bash
   git status
   git diff --staged
   ```

3. **Update progress.json**
   - If `progress.json` doesn't exist, stop and prompt user to run `/spec-new` first
   - Update `passes` for completed requirements
   - Update `lastUpdated` to today's date
   - Update `notes` with context for next session
   - Update `status` if all requirements pass
   - Add any new `blockers` discovered

4. **Run preflight check (quality gate)**
   ```bash
   make preflight
   ```
   - If preflight fails, fix issues before proceeding
   - Do NOT skip this step - broken commits waste future sessions

5. **Create structured commit**
   - Stage all relevant changes
   - Create commit with Conventional Commits subject (max 72 chars)
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
- Requirements passed: <X/Y>
- Status: <status>

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
