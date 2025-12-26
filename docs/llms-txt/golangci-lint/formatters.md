# Formatters

golangci-lint includes code formatting tools that can analyze and fix Go code formatting issues.

## Available Formatters

All formatters support automatic fixing (autofix).

### gci

Controls package import order and formatting with custom rules.

**Configuration:**

```yaml
formatters:
  enable:
    - gci
  settings:
    gci:
      # Define import sections
      sections:
        - standard                           # Standard library
        - default                           # Everything else
        - prefix(github.com/org/project)    # Project imports
      # Custom separators
      section-separators:
        - newLine
      # Skip generated files
      skip-generated: true
```

### gofmt

Standard Go formatting tool. Checks if code is formatted according to the gofmt command.

**Configuration:**

```yaml
formatters:
  enable:
    - gofmt
  settings:
    gofmt:
      # Simplify code (gofmt -s)
      simplify: true
```

### gofumpt

Stricter version of gofmt with additional formatting rules.

**Configuration:**

```yaml
formatters:
  enable:
    - gofumpt
  settings:
    gofumpt:
      # Module path for local package detection
      module-path: github.com/org/project
      # Extra rules
      extra-rules: true
```

### goimports

Manages imports and applies gofmt formatting. Updates import lines, adding missing ones and removing unreferenced ones.

**Configuration:**

```yaml
formatters:
  enable:
    - goimports
  settings:
    goimports:
      # Local package prefixes
      local-prefixes: github.com/org/project
```

### golines

Formats code and fixes long lines by breaking them intelligently.

**Configuration:**

```yaml
formatters:
  enable:
    - golines
  settings:
    golines:
      # Maximum line length
      max-len: 120
      # Tab width
      tab-len: 4
      # Base formatter (gofmt, gofumpt, goimports)
      base-formatter: gofumpt
      # Ignore generated files
      ignore-generated: true
```

### swaggo

Formats Swagger/OpenAPI comments for swag/swaggo.

**Configuration:**

```yaml
formatters:
  enable:
    - swaggo
```

## Using Formatters

### Format Code

Format all Go files in the current directory and subdirectories:

```bash
golangci-lint fmt
```

Format specific paths:

```bash
golangci-lint fmt ./pkg/... ./cmd/... main.go
```

**Note:** Directories are NOT recursive by default. Append `/...` for recursive formatting.

### Show Diff Without Modifying

Display formatting changes without applying them:

```bash
golangci-lint fmt -d
```

Show colored diff:

```bash
golangci-lint fmt --diff-colored
```

### Enable Specific Formatters

```bash
golangci-lint fmt -E gofumpt -E goimports
```

### Format from stdin

```bash
cat file.go | golangci-lint fmt --stdin
```

## Configuration File

### Enable Formatters

```yaml
formatters:
  enable:
    - gci
    - gofumpt
    - goimports
```

### Formatter-Specific Settings

```yaml
formatters:
  settings:
    gci:
      sections:
        - standard
        - default
        - prefix(github.com/myorg)

    goimports:
      local-prefixes: github.com/myorg

    golines:
      max-len: 120
      base-formatter: gofumpt
```

### Exclusions

Exclude specific paths or file types:

```yaml
formatters:
  exclusions:
    # Exclude by path pattern
    paths:
      - "generated/.*"
      - "vendor/.*"
      - ".*\\.pb\\.go$"

    # Exclude by file type
    file-types:
      - "*.pb.go"
      - "*.gen.go"
```

## Formatters vs Linters

**Formatters:**
- Focus on code formatting and style
- Modify code when run with `fmt` command
- All support autofix by design

**Linters:**
- Focus on code quality, bugs, and best practices
- Report issues without modifying code (unless --fix is used)
- Some support autofix

## Common Configurations

### Minimal Setup

```yaml
formatters:
  enable:
    - gofmt
```

### Standard Setup

```yaml
formatters:
  enable:
    - goimports
    - gofumpt
  settings:
    goimports:
      local-prefixes: github.com/org/project
```

### Complete Setup

```yaml
formatters:
  enable:
    - gci
    - gofumpt
    - golines
  settings:
    gci:
      sections:
        - standard
        - default
        - prefix(github.com/org/project)
      section-separators:
        - newLine

    gofumpt:
      extra-rules: true
      module-path: github.com/org/project

    golines:
      max-len: 120
      tab-len: 4
      base-formatter: gofumpt

  exclusions:
    paths:
      - ".*\\.pb\\.go$"
```

## Commands

View all available formatters:

```bash
golangci-lint help formatters
```

List enabled formatters:

```bash
golangci-lint formatters
```

List in JSON format:

```bash
golangci-lint formatters --json
```

## Integration with Linters

Some formatters are also available as linters and can be used with `golangci-lint run --fix`:

- gci (both linter and formatter)
- gofmt (both linter and formatter)
- gofumpt (both linter and formatter)
- goimports (both linter and formatter)
- golines (both linter and formatter)
- swaggo (both linter and formatter)

The `fmt` command is specifically designed for formatting, while `run --fix` handles both linting and formatting.

## References

- [Formatters documentation](https://golangci-lint.run/docs/formatters/)
- [Configuration guide](configuration-file.md)
