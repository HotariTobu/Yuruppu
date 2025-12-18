---
description: Analyze codebase and create ADRs for a specification
argument-hint: <spec-name>
allowed-tools: Bash(git *), Read, Write, Glob, Grep, TodoWrite, Task, Skill
---

# Design Phase

Spec name: $ARGUMENTS

## Task

1. **Load the specification**:
   - Find spec directory matching the argument (e.g., `docs/specs/*<spec-name>*/`)
   - Read `spec.md` and `progress.json`
   - If spec not found, ask user for correct name

2. **Analyze the codebase** (use Explore agent):
   - Identify existing related code
   - Check current libraries in use (`go.mod`)
   - Understand project structure

3. **Identify required ADRs**:
   Based on the spec and codebase analysis, determine if ADRs are needed for:
   - New library/framework introduction
   - Architecture pattern decisions
   - Technology choices

   If no ADRs needed, skip to step 6.

4. **Create ADRs** (use tech-stack-adr skill for each):
   - Run the skill for each identified decision
   - Follow the ADR workflow to completion

5. **Generate llms.txt** (use llms-generator agent):
   - For each adopted technology with official documentation, generate llms.txt
   - Store in `docs/llms-txt/<technology-name>/`
   - Skip if documentation is already present or not applicable (e.g., standard library, CLI tools)

6. **Update progress.json**:
   ```json
   {
     "phase": "designed",
     ...
   }
   ```

7. **Commit changes**:
   ```bash
   git add docs/adr/ docs/llms-txt/ docs/specs/<spec-name>/progress.json
   git commit -m "docs(<spec-name>): complete design phase"
   ```

## Output

```
## Design Complete: <spec-name>

### Codebase Analysis
- [Summary of relevant existing code]
- [Current libraries/patterns in use]

### ADRs Created
- ADR-XXX: [Title] → [Decision]
- (or "None required")

### llms.txt Generated
- docs/llms-txt/<technology>/
- (or "None required")

### Ready for Implementation
Run `/session-start <spec-name>` to begin.
```

## Guidelines

- Focus on decisions that affect implementation approach
- Do not create ADRs for trivial choices
- Reference existing patterns in the codebase when possible
- If the only option is obvious (e.g., official SDK), still document it briefly in ADR