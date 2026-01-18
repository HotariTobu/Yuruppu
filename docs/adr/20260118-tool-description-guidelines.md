# ADR: Tool Description Guidelines

> Date: 2026-01-18
> Status: **Adopted**

## Context

When defining tools for LLM function calling, we need guidelines for writing tool descriptions, parameter descriptions, and response descriptions. Currently, there is ambiguity about whether these descriptions should mention other tools or explain multi-tool workflows.

## Decision Drivers

- Separation of concerns
- Maintainability of tool definitions
- Flexibility in orchestration changes
- Clear responsibility boundaries

## Options Considered

- **Option 1:** Allow cross-references - Tool, parameter, and response descriptions may mention other tools and orchestration patterns
- **Option 2:** Isolate descriptions - Each tool describes only itself; orchestration belongs in system prompt

## Evaluation

| Criterion | Cross-references | Isolate |
|-----------|-----------------|---------|
| Separation of concerns | Low - mixes responsibilities | High - clear boundaries |
| Maintainability | Low - changes cascade | High - isolated changes |
| Flexibility | Low - embedded patterns | High - orchestration is configurable |
| Reusability | Low - coupled definitions | High - tools are self-contained |

## Decision

Adopt **Isolate descriptions**.

## Rationale

- Each tool description must describe only that tool's functionality
- Parameter descriptions must describe only that parameter's purpose and constraints
- Response descriptions must describe only the response structure and meaning
- Tool, parameter, and response descriptions must not reference other tools
- Multi-tool orchestration instructions belong in the system prompt
- This separation allows changing orchestration patterns without modifying tool definitions

## Consequences

**Positive:**
- Tool definitions are self-contained and reusable
- Orchestration can be changed by modifying only the system prompt
- Clear responsibility: tool = what it does, system prompt = how tools work together

**Negative:**
- Orchestration context is separated from tool definitions
- Must remember to update system prompt when adding tools that need coordination
