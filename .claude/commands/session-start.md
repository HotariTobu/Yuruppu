---
description: Start a coding session with progress review and planning
argument-hint: [spec-name]
allowed-tools: Bash(git *), Bash(make preflight), Read, Glob, TodoWrite
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

4. **Run preflight check**
   ```bash
   make preflight
   ```
   - If preflight fails, prioritize fixing issues before new work
   - Document any environment issues as blockers

5. **Review recent history**
   ```bash
   git log -1
   ```
   - Read the full commit message, especially the "Next" section

6. **Analyze progress**
   - Identify incomplete requirements (where `passes: false`)
   - Check for blockers
   - Read notes from previous session

7. **Create session plan**
   - Recommend next task based on incomplete requirements
   - Provide focused plan for this session
   - Keep scope to ONE requirement (split if too large)

## Output Format

```
## Session Start: <spec-name>

### Current Status
- Status: <pending|in_progress|completed>
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
- If no progress.json exists, stop and prompt user to run `/spec-new` first (do not proceed)
- If all requirements pass, suggest running final verification
- If preflight check fails, fix before starting new work
