# Linter Configuration

This guide covers how to configure individual linters in golangci-lint.

## Configuration Structure

Linter-specific settings are defined under `linters.settings` in the configuration file:

```yaml
linters:
  settings:
    linter-name:
      option-key: value
```

## Common Configuration Patterns

### Enable/Disable Specific Checks

Many linters support selective enabling of checks:

```yaml
linters:
  settings:
    gocritic:
      enabled-checks:
        - appendAssign
        - badCall
        - caseOrder
      disabled-checks:
        - regexpMust

    gosec:
      # Include specific rules
      includes:
        - G101  # Look for hardcoded credentials
        - G102  # Bind to all interfaces
      # Exclude specific rules
      excludes:
        - G104  # Unhandled errors
```

**Options:**
- `enable` / `enabled-checks` - Add specific checks
- `disable` / `disabled-checks` - Remove checks
- `enable-all` - Activate all available checks
- `disable-all` - Deactivate all checks

### Complexity and Size Limits

Linters that measure code complexity accept numeric thresholds:

```yaml
linters:
  settings:
    cyclop:
      # Maximum cyclomatic complexity (default: 10)
      max-complexity: 15
      # Skip tests
      skip-tests: true

    funlen:
      # Maximum function lines
      lines: 100
      # Maximum statements
      statements: 50

    gocognit:
      # Minimum cognitive complexity to report
      min-complexity: 20

    maintidx:
      # Maintainability index threshold
      under: 20
```

### Pattern Matching

Several linters use regex patterns:

```yaml
linters:
  settings:
    forbidigo:
      # Forbidden identifier patterns
      forbid:
        - ^print.*$
        - ^fmt\.Print.*$
      # Exclude test files
      exclude-godoc-examples: true

    godox:
      # Keywords to detect (TODO, FIXME, etc.)
      keywords:
        - TODO
        - FIXME
        - BUG
        - HACK

    misspell:
      # Locale (US or UK)
      locale: US
      # Ignore words list
      ignore-words:
        - someword
```

### Package and Import Control

```yaml
linters:
  settings:
    depguard:
      # List type: allowlist or denylist
      list-type: denylist
      # Packages to block
      packages:
        - github.com/sirupsen/logrus
        - log
      # Additional deny lists
      additional-guards:
        - list-type: denylist
          packages:
            - github.com/pkg/errors

    gci:
      # Import sections
      sections:
        - standard          # Standard library
        - default          # Everything else
        - prefix(github.com/org/project)  # Project imports
      # Skip generated files
      skip-generated: true

    goimports:
      # Local package prefix
      local-prefixes: github.com/org/project
```

## Popular Linters Configuration

### errcheck

Checks for unchecked errors.

```yaml
linters:
  settings:
    errcheck:
      # Check type assertions
      check-type-assertions: true
      # Check assignment to blank identifier
      check-blank: true
      # Ignore specific functions
      exclude-functions:
        - io/ioutil.ReadFile
        - io.Copy(*bytes.Buffer)
```

### staticcheck

Applies staticcheck rules (SA*, ST*, S* categories).

```yaml
linters:
  settings:
    staticcheck:
      # Enable all checks
      checks: ["all"]
      # Or select specific categories
      # checks: ["SA*", "ST1000", "ST1003"]
```

### revive

Fast, configurable linter with 100+ rules.

```yaml
linters:
  settings:
    revive:
      # Minimum confidence (0.0-1.0)
      confidence: 0.8
      # Severity (warning or error)
      severity: warning
      # Enable all rules
      enable-all-rules: false
      # Rule configuration
      rules:
        - name: blank-imports
        - name: context-as-argument
          arguments:
            - allowTypesBefore: "*testing.T"
        - name: dot-imports
        - name: error-return
        - name: error-strings
        - name: exported
          arguments:
            - "checkPrivateReceivers"
            - "sayRepetitiveInsteadOfStutters"
        - name: if-return
        - name: var-naming
          arguments:
            - ["ID"]  # Allow ID as exception to camelCase
```

### govet

Examines Go source code for suspicious constructs.

```yaml
linters:
  settings:
    govet:
      # Enable specific analyzers
      enable:
        - appends
        - atomic
        - bools
        - buildtag
        - composites
        - copylocks
        - errorsas
        - httpresponse
        - loopclosure
        - lostcancel
        - nilfunc
        - printf
        - shift
        - stdmethods
        - structtag
        - tests
        - unmarshal
        - unreachable
        - unsafeptr
      # Disable specific analyzers
      disable:
        - shadow
```

### gocritic

Highly extensible Go linter with many checks.

```yaml
linters:
  settings:
    gocritic:
      # Enable all checks
      enabled-checks:
        - appendAssign
        - badCall
        - badCond
        - captLocal
        - caseOrder
        - defaultCaseOrder
        - dupArg
        - dupBranchBody
        - dupCase
        - dupSubExpr
        - flagDeref
        - nilValReturn
        - offBy1
        - rangeExprCopy
        - regexpMust
        - sloppyLen
        - switchTrue
        - typeSwitchVar
        - underef
        - unlambda
        - unslice
      # Settings per check
      settings:
        rangeValCopy:
          sizeThreshold: 512
```

### gosec

Inspects source code for security problems.

```yaml
linters:
  settings:
    gosec:
      # Include rules
      includes:
        - G101  # Hardcoded credentials
        - G102  # Bind to all interfaces
        - G103  # Audit unsafe block
        - G104  # Unhandled errors
        - G201  # SQL injection
        - G202  # SQL query string building
      # Exclude rules
      excludes:
        - G104  # Often too noisy
      # Severity settings
      severity: medium
      confidence: medium
```

### dupl

Reports duplicated code.

```yaml
linters:
  settings:
    dupl:
      # Minimum token sequence length to report (default: 150)
      threshold: 100
```

### lll

Reports long lines.

```yaml
linters:
  settings:
    lll:
      # Maximum line length
      line-length: 120
      # Tab width
      tab-width: 4
```

## Configuration by Category

### Error Handling

```yaml
linters:
  enable:
    - errcheck      # Unchecked errors
    - errorlint     # Error wrapping issues
    - errname       # Error naming conventions
    - nilerr        # Nil returns despite error checks
  settings:
    errcheck:
      check-type-assertions: true
    errorlint:
      errorf: true
      asserts: true
      comparison: true
```

### Code Quality

```yaml
linters:
  enable:
    - cyclop        # Cyclomatic complexity
    - gocognit      # Cognitive complexity
    - funlen        # Function length
    - nestif        # Nested if depth
    - maintidx      # Maintainability index
  settings:
    cyclop:
      max-complexity: 15
    funlen:
      lines: 100
      statements: 50
```

### Security

```yaml
linters:
  enable:
    - gosec         # Security issues
    - bodyclose     # HTTP body close
    - gosec         # Security problems
  settings:
    gosec:
      excludes:
        - G104
```

### Performance

```yaml
linters:
  enable:
    - prealloc      # Slice preallocation
    - makezero      # Slice declaration issues
  settings:
    prealloc:
      simple: true
      range-loops: true
      for-loops: true
```

## Validation

Verify your configuration:

```bash
golangci-lint config verify
```

View the configuration file path:

```bash
golangci-lint config path
```

## Reference

For complete linter-specific options, see:
- [Official configuration documentation](https://golangci-lint.run/docs/linters/configuration/)
- [Configuration schema](https://github.com/golangci/golangci-lint/blob/master/golangci.jsonschema.json)
