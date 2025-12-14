---
description: Create a new specification with progress.json
argument-hint: <name>
allowed-tools: Bash(git *), Read, Write, Glob, TodoWrite, AskUserQuestion, WebSearch
---

# Create a New Specification

Feature name: $ARGUMENTS

## Task

1. Ask the user which type of specification to create:
   - **Feature**: New feature (use `docs/specs/templates/FEATURE.md`)
   - **Enhancement**: Improvement to existing feature (use `docs/specs/templates/ENHANCEMENT.md`)
   - **Fix**: Bug fix (use `docs/specs/templates/FIX.md`)
   - **Refactor**: Code refactoring (use `docs/specs/templates/REFACTOR.md`)

2. Create spec directory and files:
   - Directory: `docs/specs/yyyymmdd-[type]-name/`
     - `yyyymmdd`: Date (e.g., `20251204`)
     - `[type]`: `feat` | `enhance` | `fix` | `refact`
     - `name`: Kebab-case name (e.g., `spotify-adapter`)
   - Create `spec.md` based on the selected template
   - Create `progress.json` with all requirements set to `passes: false`

3. Work with the user to fill in the spec.md sections

4. After spec.md is complete, generate progress.json:
   ```json
   {
     "status": "pending",
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

5. Create initial commit:
   ```bash
   git add docs/specs/<spec-name>/
   git commit -m "docs(<spec-name>): add specification"
   ```

## Guidelines

- Write specifications clearly and specifically
- Avoid ambiguous expressions; detail to an implementable level
- Include TypeScript type definitions
- Consider edge cases
- Ensure all requirements have unique IDs for progress tracking
