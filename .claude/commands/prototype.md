---
description: Create a prototype to validate technical decisions
argument-hint: <spec-name>
allowed-tools: Bash(git *), Read, Write, Edit, Glob, Grep, TodoWrite, Task
---

# Prototype Phase

Spec name: $ARGUMENTS

## Purpose

Validate technical decisions through rapid prototyping:
- Technical feasibility (PoC)
- UI/UX validation
- Architecture validation
- Design verification

## Rules

- **NO TESTS** - Focus on validation, not quality
- **Commit separately** - All prototype changes must be in isolated commits for easy revert
- **Speed over perfection** - Prototype code is disposable

## Task

1. **Load the specification**:
   - Find spec directory matching the argument (e.g., `docs/specs/*<spec-name>*/`)
   - Read `spec.md` and `progress.json`
   - Verify phase is `"tech-researched"`

2. **Review ADRs**:
   - Read all ADRs created in tech-research phase
   - Identify technologies/approaches to validate

3. **Design with multiple agents** (scale based on complexity):
   - Launch multiple `prototype-designer` agents in parallel
   - Small scope: 2 agents with different perspectives
   - Medium scope: 3 agents
   - Large scope: 4+ agents
   - Give each agent a different focus or perspective
   - Review and synthesize the best ideas from each proposal
   - Resolve conflicts and select the approach before proceeding

4. **Create prototype**:
   - Implement minimal code to validate technical decisions
   - Focus on proving feasibility, not production quality
   - Skip error handling, logging, and edge cases

5. **Commit prototype** (IMPORTANT: for revert capability):
   ```bash
   git add -A
   git commit -m "prototype(<spec-name>): validate <what was validated>

   ## Findings
   - <what worked>
   - <what needs adjustment>
   - <blockers if any>"
   ```

6. **Update progress.json**:
   ```json
   {
     "phase": "prototyped",
     ...
   }
   ```

7. **Commit progress update**:
   ```bash
   git add docs/specs/<spec-name>/progress.json
   git commit -m "docs(<spec-name>): complete prototype phase"
   ```

## Output

```
## Prototype Complete: <spec-name>

### Validated
- <what was validated and result>

### Findings
- <key findings>

### Adjustments Needed
- <any ADR or design adjustments>

### Prototype Commits
- <commit hash>: <description>

### Ready for Implementation
Run `/session-start <spec-name>` to begin TDD implementation.

To revert prototype if not adopted:
git revert <prototype-commit-hash>
```

## Guidelines

- Keep prototype scope minimal - validate only what's uncertain
- If prototype reveals issues, update ADRs before proceeding
- Prototype code may be kept or discarded in implementation phase

## Error Recovery

- **Prototype proves approach infeasible**: Revert commits, update ADRs, re-run `/tech-research`
