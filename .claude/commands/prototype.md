---
description: Create a prototype to validate technical decisions
argument-hint: <spec-name>
requires-phase: tech-researched
sets-phase: prototyped
allowed-tools: Bash(git:*), Read, Write, Edit, Glob, Grep, TodoWrite, Task
---

# Prototype Phase

**Workflow**: /spec-new → /tech-research → **`/prototype`** → /design → /session-start

Spec name: $ARGUMENTS

## Purpose

Validate technical decisions through rapid prototyping:
- Technical feasibility (PoC)
- UI/UX validation
- Architecture validation
- Design verification

## Rules

- **NO TESTS** - Do NOT run, write, or modify tests
- **Commit separately** - All prototype changes must be in isolated commits for easy revert
- **Speed over perfection** - Prototype code is disposable

## Task

0. **Identify target spec**
   - If argument provided: Find `docs/specs/*<spec-name>*/`
   - If no argument: Infer from branch name (e.g., `feature/chat-history` → `*chat-history*`)
   - If not found: Stop and show error with available specs

1. **Load the specification**:
   - Read `spec.md` and `progress.json`

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

### Ready for Design
Run `/design <spec-name>` to create detailed design.

To revert prototype if not adopted:
git revert <prototype-commit-hash>
```

## Guidelines

- Keep prototype scope minimal - validate only what's uncertain
- If prototype reveals issues, update ADRs before proceeding
- Prototype code may be kept or discarded in implementation phase

## Error Recovery

- **Prototype proves approach infeasible**: Revert commits, update ADRs, re-run `/tech-research`
