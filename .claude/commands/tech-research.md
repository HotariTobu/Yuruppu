---
description: Analyze codebase and create ADRs for a specification
argument-hint: <spec-name>
allowed-tools: Bash(git *), Read, Write, Glob, Grep, TodoWrite, Task, Skill, AskUserQuestion
---

# Tech Research Phase

Spec name: $ARGUMENTS

## Task

1. **Load the specification**:
   - Find spec directory matching the argument (e.g., `docs/specs/*<spec-name>*/`)
   - Read `spec.md` and `progress.json`
   - If spec not found, ask user for correct name
   - Verify phase is `"spec"`. If not, warn user about unexpected phase

2. **Enumerate design considerations** (use tech-researcher agent):
   - Pass only the spec path to the agent
   - The agent analyzes the codebase itself (no summaries)
   - The agent systematically checks EVERY requirement
   - This step MUST NOT be skipped or done manually

3. **Create ADRs** (for each ADR candidate from step 2):
   - tech-researcher marks items as "requires ADR"
   - For each ADR candidate, run `/tech-stack-adr` skill
   - If no ADR candidates, skip to step 5.

4. **Generate llms.txt** (use llms-generator agent):
   - For each adopted technology with official documentation, generate llms.txt
   - Store in `docs/llms-txt/<technology-name>/`
   - Skip if documentation is already present or not applicable (e.g., standard library, CLI tools)

5. **Update progress.json**:
   ```json
   {
     "phase": "tech-researched",
     ...
   }
   ```

6. **Commit changes**:
   ```bash
   git add docs/adr/ docs/llms-txt/ docs/specs/<spec-name>/progress.json
   git commit -m "docs(<spec-name>): complete tech-research phase"
   ```

## Output

```
## Tech Research Complete: <spec-name>

### Design Considerations
(Paste output from tech-researcher agent)

### ADRs Created
- ADR-XXX: [Title] → [Decision]
- (or "None required")

### llms.txt Generated
- docs/llms-txt/<technology>/
- (or "None required")

### Next Step
Run `/prototype <spec-name>` to validate technical decisions.
```

## Guidelines

- Focus on decisions that affect implementation approach
- Do not create ADRs for trivial choices
- Reference existing patterns in the codebase when possible
- If the only option is obvious (e.g., official SDK), still document it briefly in ADR

## Error Recovery

- **ADR creation fails**: Create ADR manually or skip if not critical
- **llms.txt generation fails**: Skip and note in progress.json
