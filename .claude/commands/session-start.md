---
description: Start a coding session with progress review and planning
argument-hint: [spec-name]
requires-phase: designed, in-progress
sets-phase: in-progress
allowed-tools: Bash(git *), Bash(make preflight), Read, Glob, TodoWrite, Task
---

# Start Session

**Workflow**: /spec-new → /tech-research → /prototype → /design → [ **`/session-start`** → /session-end ]*

Spec name (optional): $ARGUMENTS

## Task

Start a coding session with progress review and planning.

### Steps

0. **Identify target spec**
   - If argument provided: Find `docs/specs/*<spec-name>*/`
   - If no argument: Infer from branch name (e.g., `feature/chat-history` → `*chat-history*`)
   - If not found: Stop and show error with available specs

1. **Load the specification**
   - Read `spec.md`, `progress.json`, and `design.md`

2. **Handle blocked phase** (if applicable)
   - If `phase` is `"blocked"`: Display blockers and ask user how to proceed

3. **Run preflight check**
   ```bash
   make preflight
   ```
   - If preflight fails, prioritize fixing issues before new work
   - Document any environment issues as blockers

4. **Review recent history**
   ```bash
   git log -1
   ```
   - Read the full commit message, especially the "Next" section

5. **Analyze progress**
   - Identify incomplete requirements (where `passes: false`)
   - Read notes from previous session

6. **Create session plan**
   - Recommend next task based on incomplete requirements
   - Provide focused plan for this session
   - Keep scope to ONE requirement (split if too large)

7. **Update phase to in-progress**:
   - If phase is `"designed"`, update progress.json to `"in-progress"`

8. **TDD Implementation**
   - Use `go-test-generator` agent to generate tests and verify red phase
   - Use `go-implementer` agent to implement code and verify green phase

## Output Format

```
## Session Start: <spec-name>

### Current Status
- Phase: <phase>
- Progress: <X/Y requirements passed>

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
