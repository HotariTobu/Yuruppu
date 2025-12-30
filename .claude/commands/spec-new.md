---
description: Create a new specification with progress.json
argument-hint: <name>
sets-phase: spec
allowed-tools: Bash(git *), Read, Write, Glob, TodoWrite, AskUserQuestion, WebSearch, Task
---

# Create a New Specification

**Workflow**: **`/spec-new`** → /tech-research → /prototype → /design → /session-start

Feature name: $ARGUMENTS

## Task

1. Ask the user which type of specification to create:
   - **Feature**: New feature (use `docs/specs/templates/FEATURE.md`)
   - **Enhancement**: Improvement to existing feature (use `docs/specs/templates/ENHANCEMENT.md`)
   - **Fix**: Bug fix (use `docs/specs/templates/FIX.md`)
   - **Refactor**: Code refactoring (use `docs/specs/templates/REFACTOR.md`)

2. Check current branch:
   - If on `main`: proceed to step 3
   - If on another branch: ask user "Create branch from current branch (`<branch>`) or from `main`?"

3. Create a branch based on the spec type:
   - Feature/Enhancement: `feature/<name>`
   - Fix: `fix/<name>`
   - Refactor: `refactor/<name>`

4. Create spec directory and files:
   - Directory: `docs/specs/yyyymmdd-[type]-name/`
     - `yyyymmdd`: Date (e.g., `20251204`)
     - `[type]`: `feat` | `enhance` | `fix` | `refact`
     - `name`: Kebab-case name (e.g., `spotify-adapter`)
   - Create `spec.md` based on the selected template
   - Create `progress.json` with all requirements set to `passes: false`

5. Work with the user to fill in the spec.md sections

6. After spec.md is complete, generate progress.json:
   ```json
   {
     "phase": "spec",
     "lastUpdated": "<today>",
     "requirements": [
       { "id": "FR-001", "passes": false },
       { "id": "FR-002", "passes": false }
     ],
     "blockers": [],
     "notes": ""
   }
   ```
   - Extract all requirement IDs from spec.md (FR-*, NFR-*, TR-*)
   - Set all `passes` to `false`

7. **Run spec-reviewer to validate the specification**:
   - Launch spec-reviewer agent with the spec name
   - Review the output with the user
   - If "Needs revision": work with user to fix issues, then re-run spec-reviewer
   - Repeat until spec-reviewer returns "Ready for implementation"

8. Create initial commit:
   ```bash
   git add docs/specs/<spec-name>/
   git commit -m "docs(<spec-name>): add specification"
   ```

## Guidelines

- Write specifications clearly and specifically
- Avoid ambiguous expressions; detail to an implementable level
- Include Go type definitions
- Consider edge cases
- Ensure all requirements have unique IDs for progress tracking

## Error Recovery

- **Branch already exists**: Ask user to delete or use existing branch
- **Spec-reviewer keeps rejecting**: Simplify requirements, split into smaller specs
