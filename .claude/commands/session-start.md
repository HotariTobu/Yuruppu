---
description: Start a coding session with progress review and planning
argument-hint: [spec-name]
allowed-tools: Bash(git *), Bash(make preflight), Read, Glob, TodoWrite, Task
---

# Start Session

Spec name (optional): $ARGUMENTS

## Task

Start a coding session with progress review and planning.

### Steps

1. **Get current branch name**
   ```bash
   git branch --show-current
   ```

2. **Identify target spec**
   - If spec-name argument is provided: Use `docs/specs/<spec-name>/`
   - Else: Infer from branch name (e.g., `feature/spotify-adapter` -> search for `*spotify*`)
   - If not found: Ask user which spec to work on

3. **Load spec context**
   - Read `docs/specs/<spec-name>/spec.md`
   - Read `docs/specs/<spec-name>/progress.json`
   - If `progress.json` doesn't exist, stop and ask user to run `/spec-new` first

4. **Check phase**
   - If `phase` is `"blocked"`: Display blockers and ask user how to proceed
   - If `phase` is `"designed"` or `"in-progress"`: Continue to next step
   - Otherwise: Stop and prompt user to run `/design <spec-name>` first

5. **Run preflight check**
   ```bash
   make preflight
   ```
   - If preflight fails, prioritize fixing issues before new work
   - Document any environment issues as blockers

6. **Review recent history**
   ```bash
   git log -1
   ```
   - Read the full commit message, especially the "Next" section

7. **Analyze progress**
   - Identify incomplete requirements (where `passes: false`)
   - Read notes from previous session
   - **Check for blockers**: If blockers exist, display them and ask user:
     - "Resolve blockers first?" → Help resolve, then continue
     - "Proceed anyway?" → Document decision in notes

8. **Create session plan**
   - Recommend next task based on incomplete requirements
   - Provide focused plan for this session
   - Keep scope to ONE requirement (split if too large)

9. **Update phase to in-progress** (after user approval):
   - If phase is `"designed"`, update progress.json to `"in-progress"`

10. **TDD Implementation**
   - Use `go-test-generator` agent to generate tests and verify red phase
   - Use `go-implementer` agent to implement code and verify green phase

## Output Format

```
## Session Start: <spec-name>

### Current Status
- Phase: <designed|in-progress|completed>
- Progress: <X/Y requirements passed>
- Last updated: <date>

### Previous Notes
<notes from progress.json>

### Blockers
<list of blockers or "None">

### Recommended Next Task
<specific requirement to work on>

### Session Plan
1. <step 1>
2. <step 2>
...
```

## Guidelines

- Focus on ONE requirement per session (not "when possible" - always)
- If a requirement is too large, split into sub-tasks before starting
- If no progress.json exists, stop and prompt user to run `/spec-new` first
- If design phase not complete, stop and prompt user to run `/design` first
- If all requirements pass, suggest running final verification
- If preflight check fails, fix before starting new work

## Requirement Size Guidelines

A requirement is **TOO LARGE** if:
- Implementation touches more than 3 files
- Expected to take more than 50 tool calls
- Contains "and" connecting distinct features

**Split strategy:**
1. Identify sub-tasks
2. Create temporary sub-requirements (e.g., FR-001a, FR-001b)
3. Complete each in separate session
4. Mark parent requirement as passed only when all sub-tasks complete

## Error Recovery

- **Session interrupted**: `git stash` to preserve work, run `/session-start` to re-orient
- **Requirement too large**: Split into sub-requirements (FR-001a, FR-001b)
- **Unexpected failure**: Document in blockers, update notes for next session
