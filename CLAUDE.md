# CLAUDE.md

> **Rule**: Do not add anything to CLAUDE.md unless it is necessary.

## Project Overview

Yuruppu is a LINE bot that responds as the character "Yuruppu". Written in Go.

## Language

All documentation, code comments, commit messages, and issues must be written in **English**.

Exception: Specification content (`docs/specs/*.md`) may be written in other languages.

## Directory Structure

```
docs/
  adr/              # Architecture Decision Records
  llms-txt/         # LLM documentation for libraries (LINE SDK, etc.)
  specs/            # Feature specifications (spec-driven development)
    templates/      # Spec templates (FEATURE, ENHANCEMENT, FIX, REFACTOR)
```

## Spec-Driven Development

This repository follows **Spec-Driven Development**.

1. **Write specs before code** - Always create a specification before implementation
2. **Keep change history** - Update the history section when modifying specs
3. **Follow the spec** - Do not implement features not described in the spec
4. **Derive tests from specs** - Test cases should be based on spec requirements

## Development Workflow

```
/spec-new → /dependency-research → [ /session-start → /session-end ]* → PR
```

`*` = Repeat until all requirements pass. Each session focuses on **one requirement**.

## Ambiguous Instructions

When receiving ambiguous or short instructions from the user, explain your interpretation and ask for confirmation before proceeding with significant actions.

## Web Search

When including a year in search queries, you **must** check the current date first to ensure accuracy.
