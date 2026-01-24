---
description: Create detailed design collaboratively with user
argument-hint: <spec-name>
requires-phase: prototyped
sets-phase: designed
allowed-tools: Bash(git:*), Read, Write, Edit, Glob, Grep, TodoWrite, AskUserQuestion
---

# Design Phase

**Workflow**: /spec-new → /tech-research → /prototype → **`/design`** → /session-start

Spec name: $ARGUMENTS

## Purpose

Create detailed design through collaboration with user. This ensures implementation aligns with user expectations.

## Rules

- **Collaborate with user** - Do NOT delegate to subagents
- **Confirm frequently** - Ask user for confirmation at each major decision
- **Document decisions** - Write to `docs/specs/<spec-name>/design.md`

## Task

0. **Identify target spec**
   - If argument provided: Find `docs/specs/*<spec-name>*/`
   - If no argument: Infer from branch name (e.g., `feature/chat-history` → `*chat-history*`)
   - If not found: Stop and show error with available specs

1. **Load context**:
   - Read `spec.md` and `progress.json`
   - Read prototype commits and any prototype code

2. **Review prototype learnings**:
   - What worked in the prototype?
   - What needs to change?
   - Ask user: "Are there any additional insights from the prototype?"

3. **Design file structure** (confirm with user):
   - List files involved in this feature
   - Describe what each file provides (target state, not changes)
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

9. **Handle prototype code** (confirm with user):
   - Ask user: "Do you want to revert the prototype code before implementation?"
     - If yes: Revert prototype commits
     - If no: Keep as reference

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

func (e *Example) Method() error
```

## Data Flow
1. Input: <description>
2. Process: <description>
3. Output: <description>

```

## Output

```
## Design Complete: <spec-name>

### Summary
<Brief summary of design decisions>

### Files
- <list of files involved>

### Ready for Implementation
Run `/session-start <spec-name>` to begin TDD implementation.
```

## Guidelines

- Never proceed without user confirmation
- Keep design focused on what's needed for implementation
- Reference ADRs and prototype learnings
- If user disagrees, revise and confirm again
- **Describe the target state, not diffs** - Write the final implementation, not "Before/After" comparisons. Design documents should work regardless of whether prototype code exists or has been reverted.

## Error Recovery

- **Design reveals spec issues**: Update spec.md, re-run spec-reviewer
- **User and Claude disagree**: Document both options, let user decide
