---
description: Create detailed design collaboratively with user
argument-hint: <spec-name>
allowed-tools: Bash(git *), Read, Write, Edit, Glob, Grep, TodoWrite, AskUserQuestion
---

# Design Phase

Spec name: $ARGUMENTS

## Purpose

Create detailed design through collaboration with user. This ensures implementation aligns with user expectations.

## Rules

- **Collaborate with user** - Do NOT delegate to subagents
- **Confirm frequently** - Ask user for confirmation at each major decision
- **Document decisions** - Write to `docs/specs/<spec-name>/design.md`

## Task

1. **Load context**:
   - Read `docs/specs/<spec-name>/spec.md`
   - Read `docs/specs/<spec-name>/progress.json`
   - Read prototype commits and any prototype code
   - Verify phase is `"prototyped"`

2. **Review prototype learnings**:
   - What worked in the prototype?
   - What needs to change?
   - Ask user: "Are there any additional insights from the prototype?"

3. **Design file structure** (confirm with user):
   - List files to create/modify
   - Explain purpose of each file
   - Ask user: "Does this file structure look correct?"

4. **Design interfaces** (confirm with user):
   - Define types, structs, method signatures
   - Show proposed interfaces
   - Ask user: "Do these interfaces meet your expectations?"

5. **Design data flow** (confirm with user):
   - Describe how data flows through the system
   - Identify transformations
   - Ask user: "Does this data flow make sense?"

6. **Write design.md**:
   - Create `docs/specs/<spec-name>/design.md` with all decisions
   - Show final design to user for approval
   - Ask user: "Is this design ready for implementation?"

7. **Update progress.json**:
   ```json
   {
     "phase": "designed",
     ...
   }
   ```

8. **Commit changes**:
   ```bash
   git add docs/specs/<spec-name>/design.md docs/specs/<spec-name>/progress.json
   git commit -m "docs(<spec-name>): complete design phase"
   ```

## design.md Template

```markdown
# Design: <spec-name>

## Overview
<Brief description of what this design covers>

## File Structure
| File | Purpose |
|------|---------|
| path/to/file.go | Description |

## Interfaces

### <Interface/Type Name>
```go
type Example struct {
    // fields
}

func (e *Example) Method() error {
    // signature
}
```

## Data Flow
1. Input: <description>
2. Process: <description>
3. Output: <description>

## Implementation Notes
- <Important consideration 1>
- <Important consideration 2>
```

## Output

```
## Design Complete: <spec-name>

### Summary
<Brief summary of design decisions>

### Files
- <list of files to create/modify>

### Ready for Implementation
Run `/session-start <spec-name>` to begin TDD implementation.
```

## Guidelines

- Never proceed without user confirmation
- Keep design focused on what's needed for implementation
- Reference ADRs and prototype learnings
- If user disagrees, revise and confirm again
