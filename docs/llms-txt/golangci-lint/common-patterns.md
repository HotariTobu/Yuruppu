# Common Configuration Patterns

This guide provides ready-to-use configuration patterns for common scenarios.

## Minimal Configuration

A basic setup for new projects:

```yaml
version: "2"

linters:
  default: standard

run:
  timeout: 5m
```

## Standard Configuration

Recommended setup for most projects:

```yaml
version: "2"

linters:
  default: standard
  enable:
    - errcheck
    - gosec
    - gocritic
    - gocyclo
    - gofmt
    - goimports
    - revive
    - misspell

  settings:
    errcheck:
      check-type-assertions: true

    gocyclo:
      min-complexity: 15

    goimports:
      local-prefixes: github.com/yourorg/yourproject

issues:
  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - errcheck
        - dupl

run:
  timeout: 5m
  tests: true

output:
  formats:
    - format: text
```

## Strict Configuration

Maximum code quality enforcement:

```yaml
version: "2"

linters:
  default: all
  disable:
    - exhaustruct      # Too strict
    - varnamelen      # Too opinionated
    - wrapcheck       # Too noisy

  settings:
    errcheck:
      check-type-assertions: true
      check-blank: true

    gocyclo:
      min-complexity: 10

    funlen:
      lines: 80
      statements: 40

    gocognit:
      min-complexity: 15

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

  exclude-rules:
    - path: ".*\\.pb\\.go$"
      linters:
        - all

run:
  timeout: 10m
  tests: true
```

## Security-Focused Configuration

Emphasize security checks:

```yaml
version: "2"

linters:
  enable:
    - gosec
    - bodyclose
    - errcheck
    - errorlint
    - exportloopref
    - gocritic
    - noctx
    - rowserrcheck
    - sqlclosecheck

  settings:
    gosec:
      excludes:
        # Keep G104 enabled (unhandled errors)
      severity: medium
      confidence: medium

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

run:
  timeout: 5m
```

## Performance-Focused Configuration

Optimize for fast analysis:

```yaml
version: "2"

linters:
  default: fast
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused

run:
  timeout: 2m
  concurrency: 4

issues:
  max-issues-per-linter: 50
  max-same-issues: 3
```

## Large Project Configuration

For projects with many existing issues:

```yaml
version: "2"

linters:
  default: standard
  enable:
    - errcheck
    - govet
    - staticcheck

  exclusions:
    presets:
      - common-false-positives
      - std-error-handling

issues:
  # Only check new code
  new: true

  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - errcheck

run:
  timeout: 10m
  skip-dirs:
    - vendor
    - third_party
    - generated
```

## Monorepo Configuration

For monorepos with multiple services:

```yaml
version: "2"

linters:
  default: standard
  enable:
    - depguard
    - gci
    - goimports

  settings:
    depguard:
      list-type: denylist
      packages:
        - github.com/sirupsen/logrus  # Use structured logging
        - log                          # Use structured logging

    gci:
      sections:
        - standard
        - default
        - prefix(github.com/yourorg)
        - prefix(github.com/yourorg/yourproject)

    goimports:
      local-prefixes: github.com/yourorg/yourproject

issues:
  exclude-rules:
    - path: "service1/.*"
      linters:
        - gochecknoglobals

run:
  timeout: 10m
  build-tags:
    - integration
```

## Test-Focused Configuration

Special handling for test files:

```yaml
version: "2"

linters:
  enable:
    - testifylint
    - thelper
    - tparallel
    - paralleltest
    - errcheck

issues:
  exclude-rules:
    # Relax rules for test files
    - path: ".*_test\\.go"
      linters:
        - funlen
        - gocognit
        - gocyclo
        - dupl

    # Allow globals in test setup
    - path: ".*_test\\.go"
      linters:
        - gochecknoglobals

run:
  tests: true
```

## Generated Code Handling

Exclude generated files:

```yaml
version: "2"

linters:
  default: standard

issues:
  exclude-rules:
    # Exclude all generated files
    - path: ".*\\.pb\\.go$"
      linters:
        - all

    - path: ".*\\.gen\\.go$"
      linters:
        - all

    - path: "generated/.*"
      linters:
        - all

    # Exclude mock files
    - path: ".*_mock\\.go$"
      linters:
        - all

run:
  skip-dirs:
    - vendor
    - generated
    - mocks
```

## CI/CD Configuration

Optimized for continuous integration:

```yaml
version: "2"

linters:
  default: standard
  enable:
    - errcheck
    - gosec
    - gocritic

  exclusions:
    presets:
      - common-false-positives

issues:
  # Show all issues in CI
  max-issues-per-linter: 0
  max-same-issues: 0

  # Only check new code in PR builds
  new-from-merge-base: true

run:
  timeout: 10m
  tests: true

output:
  formats:
    - format: text
      path: stdout
    - format: junit-xml
      path: report.xml
    - format: github-actions  # For GitHub Actions annotation
```

## Docker/Container Configuration

For containerized builds:

```yaml
version: "2"

linters:
  default: standard

run:
  timeout: 10m
  modules-download-mode: readonly

  skip-dirs:
    - vendor

issues:
  max-issues-per-linter: 0
```

## Migration from v1 to v2

Use the migrate command:

```bash
golangci-lint migrate .golangci.yml
```

Or manually update:

```yaml
# v1 (old)
linters:
  enable-all: true
  disable:
    - maligned

# v2 (new)
version: "2"
linters:
  default: all
  disable:
    - maligned
```

## Import Organization

Organize imports by section:

```yaml
version: "2"

formatters:
  enable:
    - gci
    - goimports

  settings:
    gci:
      sections:
        - standard                              # Standard library
        - default                              # Third party
        - prefix(github.com/yourorg)           # Organization
        - prefix(github.com/yourorg/project)   # Current project
      section-separators:
        - newLine

    goimports:
      local-prefixes: github.com/yourorg/project
```

## Custom Error Handling

Strict error handling:

```yaml
version: "2"

linters:
  enable:
    - errcheck
    - errorlint
    - errname
    - nilerr
    - wrapcheck

  settings:
    errcheck:
      check-type-assertions: true
      check-blank: true
      exclude-functions:
        - (io.Closer).Close  # Common pattern

    errorlint:
      errorf: true
      asserts: true
      comparison: true

    wrapcheck:
      ignoreSigs:
        - .Errorf(
        - errors.New(
        - errors.Unwrap(
```

## Code Quality Enforcement

Enforce code quality metrics:

```yaml
version: "2"

linters:
  enable:
    - cyclop
    - funlen
    - gocognit
    - gocyclo
    - nestif
    - maintidx

  settings:
    cyclop:
      max-complexity: 10
      skip-tests: true

    funlen:
      lines: 60
      statements: 40

    gocognit:
      min-complexity: 15

    nestif:
      min-complexity: 4

    maintidx:
      under: 20
```

## Style Consistency

Enforce consistent code style:

```yaml
version: "2"

linters:
  enable:
    - gofmt
    - gofumpt
    - goimports
    - godot
    - goheader
    - misspell

formatters:
  enable:
    - gofumpt
    - goimports

  settings:
    gofumpt:
      extra-rules: true

issues:
  exclude-rules:
    - path: ".*_test\\.go"
      linters:
        - godot
```

## Complete Production-Ready Configuration

A comprehensive configuration for production projects:

```yaml
version: "2"

linters:
  default: standard
  enable:
    # Error handling
    - errcheck
    - errorlint
    - errname

    # Security
    - gosec
    - bodyclose

    # Code quality
    - gocyclo
    - gocognit
    - funlen
    - nestif

    # Style
    - gofmt
    - goimports
    - misspell
    - revive

    # Best practices
    - gocritic
    - unconvert
    - unparam

  settings:
    errcheck:
      check-type-assertions: true

    gocyclo:
      min-complexity: 15

    funlen:
      lines: 100
      statements: 50

    goimports:
      local-prefixes: github.com/yourorg/project

    gosec:
      excludes:
        - G104  # Managed via errcheck

formatters:
  enable:
    - gofumpt
    - goimports

  settings:
    gofumpt:
      extra-rules: true

issues:
  max-issues-per-linter: 0
  max-same-issues: 0

  exclude-rules:
    # Test files
    - path: ".*_test\\.go"
      linters:
        - funlen
        - gocognit
        - errcheck

    # Generated files
    - path: ".*\\.pb\\.go$"
      linters:
        - all

  exclusions:
    presets:
      - common-false-positives

run:
  timeout: 5m
  tests: true
  skip-dirs:
    - vendor
    - generated

output:
  formats:
    - format: text
      path: stdout
    - format: json
      path: golangci-lint-report.json
  show-stats: true
```

## References

- [Configuration file reference](configuration-file.md)
- [Linter configuration guide](linter-configuration.md)
- [Available linters](linters.md)
