# Feature: [Feature Name]

> Template for new features.
> Filename: `yyyymmdd-feat-name.md`

## Overview

<!-- What does this feature do? (1-2 sentences) -->

## Background & Purpose

<!-- Why is this feature needed? -->

## Requirements

<!-- Each requirement must have a unique ID for progress tracking -->

### Functional Requirements

- [ ] FR-001: Requirement 1
- [ ] FR-002: Requirement 2
- [ ] FR-003: Requirement 3

### Non-Functional Requirements

- [ ] NFR-001: Requirement 1

## API Design

### Functions/Methods

```go
// ExampleFunction does something.
// param1 is the parameter description.
// Returns the result and any error encountered.
func ExampleFunction(param1 Type) (ReturnType, error)
```

### Type Definitions

```go
// ExampleStruct represents an example structure.
type ExampleStruct struct {
    Property1 Type
    Property2 Type
}
```

## Usage Examples

```go
import "github.com/user/project/pkg"

result, err := pkg.ExampleFunction(param)
if err != nil {
    // handle error
}
```

## Error Handling

| Error Type | Condition | Message |
|------------|-----------|---------|
| TypeError | When invalid argument is passed | "Invalid argument: ..." |

## Acceptance Criteria

<!--
Define acceptance criteria using Given-When-Then (GWT) format.
Each criterion must have a unique ID (AC-XXX) linked to a requirement (FR-XXX).
-->

### AC-001: [Linked to FR-001]

- **Given**: [Initial context/state]
- **When**: [Action performed]
- **Then**:
  - [Expected outcome 1]
  - [Expected outcome 2]

### AC-002: [Linked to FR-001, Error Case]

- **Given**: [Initial context/state]
- **When**: [Action that causes error]
- **Then**:
  - [Expected error type] is thrown
  - Error message contains "[expected message]"

<!--
Example:

### AC-001: Search returns results [FR-001]

- **Given**: Valid API key is configured
- **When**: User searches for "bohemian rhapsody"
- **Then**:
  - Results array is returned
  - Each result contains id, title, artist
  - Results are sorted by relevance

### AC-002: Invalid API key error [FR-001, Error]

- **Given**: Invalid API key is configured
- **When**: User performs any search
- **Then**:
  - AuthenticationError is thrown
  - Error message contains "Invalid API key"
-->

## Implementation Notes

<!-- Notes for implementation, references, etc. -->

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| YYYY-MM-DD | 1.0 | Initial version | - |
