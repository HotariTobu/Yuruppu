# Configuration File (.golangci.yml)

## File Discovery

golangci-lint searches for configuration files in this order:

1. `.golangci.yml`
2. `.golangci.yaml`
3. `.golangci.toml`
4. `.golangci.json`

The tool searches from the current working directory up to the root, then checks the home directory. Use the `-v` flag to see which config file is being used.

## Root Structure

```yaml
version: "2"
linters:
  # Linter configuration
formatters:
  # Formatter configuration
issues:
  # Issue reporting configuration
output:
  # Output configuration
run:
  # Execution configuration
severity:
  # Severity configuration
```

## Version

**Required.** Must be set to `"2"`.

```yaml
version: "2"
```

## Linters Section

Configure which linters to enable/disable and their settings.

```yaml
linters:
  # Choose preset: standard, all, none, or fast
  default: standard

  # Enable specific linters
  enable:
    - errcheck
    - gosec
    - gocritic

  # Disable specific linters
  disable:
    - typecheck

  # Per-linter configuration
  settings:
    errcheck:
      check-type-assertions: true
      check-blank: true

    gocritic:
      enabled-checks:
        - appendAssign
        - badCall
      disabled-checks:
        - regexpMust

    gosec:
      excludes:
        - G104  # Unhandled errors

  # Exclusion rules
  exclusions:
    # Exclude by text pattern
    rules:
      - text: "should have comment"
        linters:
          - golint

      # Exclude by source pattern
      - source: "^//"
        linters:
          - lll

    # Exclude by path (regex)
    paths:
      - ".*_test\\.go$"
      - "vendor/.*"

    # Check only specific paths
    path-except:
      - "pkg/.*\\.go$"

    # Use preset exclusions
    presets:
      - comments                    # Documentation-related
      - common-false-positives      # Known issues
      - legacy                      # Backwards compatibility
      - std-error-handling          # Standard library output
```

### Default Linter Presets

- `standard` - Recommended linters (default)
- `all` - All available linters
- `none` - No linters (use with enable list)
- `fast` - Only fast-running linters

## Formatters Section

Configure code formatting tools.

```yaml
formatters:
  # Enable formatters
  enable:
    - gci
    - gofmt
    - goimports

  # Formatter settings
  settings:
    gci:
      sections:
        - standard
        - default
        - prefix(github.com/org/project)

    goimports:
      local-prefixes: github.com/org/project

    golines:
      max-len: 120
      tab-len: 4

  # Exclusions
  exclusions:
    paths:
      - "generated/.*"

    file-types:
      - "*.pb.go"
```

### Available Formatters

- **gci** - Import statement formatting with custom rules (autofix)
- **gofmt** - Standard Go formatting (autofix)
- **gofumpt** - Stricter gofmt with additional rules (autofix)
- **goimports** - Import management (autofix)
- **golines** - Long line splitting (autofix)
- **swaggo** - Swagger comment formatting (autofix)

## Issues Section

Control issue reporting behavior.

```yaml
issues:
  # Maximum issues per linter (default: 50, 0 = unlimited)
  max-issues-per-linter: 0

  # Maximum identical issues (default: 3, 0 = unlimited)
  max-same-issues: 0

  # Report only new issues
  new: false

  # Apply auto-fixes
  fix: false

  # Exclude test files
  exclude-tests: false

  # Exclude by rule
  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - errcheck
        - dupl

    - path: "internal/.*"
      text: "exported.*should have comment"
```

## Output Section

Define reporting formats and paths.

```yaml
output:
  # Output formats (can specify multiple)
  formats:
    - format: text
      path: stdout

    - format: json
      path: report.json

    - format: html
      path: report.html

  # Available formats:
  # text, json, tab, html, checkstyle, code-climate,
  # junit-xml, teamcity, sarif

  # Add prefix to file paths
  path-prefix: ""

  # Sort results (linter, severity, file)
  sort-order:
    - linter
    - severity
    - file

  # Show statistics per linter
  show-stats: false
```

## Run Section

Configure analysis execution.

```yaml
run:
  # Analysis timeout (default: 1m)
  timeout: 5m

  # Include test files (default: true)
  tests: true

  # Build tags
  build-tags:
    - integration
    - e2e

  # Module download mode (readonly, vendor, mod)
  modules-download-mode: readonly

  # Specify Go version limit
  go: "1.21"

  # Concurrency (CPUs by default, 0 = number of CPUs)
  concurrency: 4

  # Skip directories
  skip-dirs:
    - vendor
    - third_party

  # Skip files
  skip-files:
    - ".*\\.pb\\.go$"
```

## Severity Section

Set issue severity levels.

```yaml
severity:
  # Default severity for unmapped issues
  default: error

  # Context-specific severity
  rules:
    - linters:
        - dupl
      severity: warning

    - path: ".*_test\\.go"
      severity: info
```

## Path Variables

- `${base-path}` - Relative to golangci-lint execution directory
- `${config-path}` - Relative to configuration file location

This supports monorepo and multi-project setups.

## Validation

Configuration can be validated against the JSON Schema: `golangci.jsonschema.json`

```bash
golangci-lint config verify
```

## Complete Example

```yaml
version: "2"

linters:
  default: standard
  enable:
    - errcheck
    - gosec
    - gocritic
    - gocyclo
    - dupl

  settings:
    errcheck:
      check-type-assertions: true
    gocyclo:
      min-complexity: 15

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - errcheck

run:
  timeout: 5m
  tests: true
  build-tags:
    - integration

output:
  formats:
    - format: text
    - format: json
      path: report.json
```
