# Handling False Positives

False positives are inevitable in static analysis. golangci-lint provides several strategies to manage them.

## Inline Nolint Directives

Use `//nolint` comments to suppress warnings at specific locations.

### Basic Usage

Exclude all linters for a line:

```go
var bad_name = 123 //nolint:all
```

Exclude specific linters:

```go
var bad_name = 123 //nolint:stylecheck,revive
```

**Important:** No spaces allowed between `//`, `nolint`, `:`, or linter names.

### Block-Level Exclusion

Place at the beginning of a line to affect the entire function or declaration:

```go
//nolint:gocyclo,funlen
func complexFunction() {
    // Complex code here
}
```

### File-Level Exclusion

Place at the package declaration to exclude entire files:

```go
//nolint:all
package generated

// File content
```

### With Explanation

Add explanation after two slashes:

```go
var legacy_code = "test" //nolint:stylecheck // legacy code, will refactor later
```

## Configuration-Based Exclusions

### Exclude by Text Pattern

Exclude issues matching specific text:

```yaml
linters:
  exclusions:
    rules:
      - text: "should have comment"
        linters:
          - golint
          - revive

      - text: "G104"  # Exclude gosec G104
        linters:
          - gosec
```

### Exclude by Source Pattern

Exclude based on source code content:

```yaml
linters:
  exclusions:
    rules:
      # Ignore line length for comments
      - source: "^//"
        linters:
          - lll
```

### Exclude by Path

Exclude entire files or directories:

```yaml
linters:
  exclusions:
    # Path patterns (regex)
    paths:
      - ".*_test\\.go$"        # All test files
      - "vendor/.*"            # Vendor directory
      - "internal/generated/.*" # Generated code
      - ".*\\.pb\\.go$"        # Protocol buffer files

    # Check only specific paths
    path-except:
      - "pkg/.*\\.go$"
```

### Exclude by Rule Combination

Combine multiple criteria:

```yaml
issues:
  exclude-rules:
    # Disable specific linters in test files
    - path: ".*_test\\.go"
      linters:
        - errcheck
        - dupl
        - gosec

    # Disable in specific directories
    - path: "internal/.*"
      text: "exported.*should have comment"

    # Disable specific checks in specific files
    - path: "cmd/.*"
      linters:
        - gochecknoglobals
```

## Exclusion Presets

Use predefined exclusion sets for common scenarios:

```yaml
linters:
  exclusions:
    presets:
      - comments                    # Documentation-related false positives
      - common-false-positives      # Known issues in gosec and others
      - legacy                      # Backwards compatibility concerns
      - std-error-handling          # Standard library output operations
```

### Preset Descriptions

- **comments** - Excludes comment-related issues (missing godoc, comment formatting)
- **common-false-positives** - Excludes well-known false positives from various linters
- **legacy** - Excludes issues that maintain backwards compatibility
- **std-error-handling** - Excludes unchecked errors from standard library functions like `fmt.Printf`

## Linter-Specific Configuration

Many linters offer settings to disable specific rules.

### staticcheck

```yaml
linters:
  settings:
    staticcheck:
      # Disable specific checks
      checks: ["all", "-SA1000", "-ST1000"]
```

### gosec

```yaml
linters:
  settings:
    gosec:
      # Exclude specific rules
      excludes:
        - G104  # Unhandled errors (often too noisy)
        - G204  # Subprocess with variable
```

### revive

```yaml
linters:
  settings:
    revive:
      rules:
        - name: exported
          disabled: true
```

### errcheck

```yaml
linters:
  settings:
    errcheck:
      # Ignore specific functions
      exclude-functions:
        - io/ioutil.ReadFile
        - io.Copy(*bytes.Buffer)
        - (io.Closer).Close
```

## Advanced Patterns

### Exclude Test Files Globally

```yaml
issues:
  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - errcheck
        - dupl
        - funlen
        - gocognit
```

### Exclude Generated Files

```yaml
issues:
  exclude-rules:
    - path: ".*\\.pb\\.go$"
      linters:
        - all

    - path: ".*\\.gen\\.go$"
      linters:
        - all
```

### Exclude by Severity

```yaml
issues:
  exclude-rules:
    - severity: info
```

### Maximum Issues Limits

Limit the number of reported issues to avoid being overwhelmed:

```yaml
issues:
  # Maximum issues per linter (0 = unlimited)
  max-issues-per-linter: 50

  # Maximum identical issues (0 = unlimited)
  max-same-issues: 3
```

To see all issues, set both to 0:

```yaml
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

## Gradual Integration

For projects with many existing issues, only check new code:

### Check Only New Issues

```yaml
issues:
  new: true
```

Or via CLI:

```bash
golangci-lint run --new
```

### Check New Issues from Git Revision

```bash
# From specific revision
golangci-lint run --new-from-rev=HEAD~1

# From merge base with main
golangci-lint run --new-from-merge-base=main
```

**Note:** These commands compare git diff output against reported issues. Use `--whole-files` if issues aren't detected on modified lines.

## Complete Example

```yaml
version: "2"

linters:
  enable:
    - errcheck
    - gosec
    - gocritic
    - revive

  settings:
    gosec:
      excludes:
        - G104  # Too noisy for unhandled errors

    errcheck:
      exclude-functions:
        - (io.Closer).Close

  exclusions:
    presets:
      - common-false-positives
      - std-error-handling

    paths:
      - ".*_test\\.go$"
      - ".*\\.pb\\.go$"

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

  exclude-rules:
    # Disable in test files
    - path: ".*_test\\.go"
      linters:
        - errcheck
        - dupl

    # Disable in generated code
    - path: "internal/generated/.*"
      linters:
        - all

    # Disable exported comment requirement in cmd
    - path: "cmd/.*"
      text: "exported.*should have comment"
```

## Best Practices

1. **Prefer configuration over inline comments** - Easier to maintain and review
2. **Use presets first** - They handle common false positives
3. **Document exclusions** - Add comments explaining why issues are excluded
4. **Review exclusions regularly** - Remove obsolete exclusions
5. **Use gradual integration** - For large projects, use `--new` flag
6. **Be specific** - Exclude only what's necessary, not entire linters

## References

- [False positives documentation](https://golangci-lint.run/docs/linters/false-positives/)
- [Configuration file guide](configuration-file.md)
