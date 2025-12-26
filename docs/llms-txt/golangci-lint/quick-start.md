# Quick Start

## Basic Usage

Run golangci-lint to analyze Go code in the current directory and subdirectories:

```bash
golangci-lint run
```

Specify targets explicitly:

```bash
golangci-lint run dir1 dir2/... file1.go
```

**Important:** Directories are NOT analyzed recursively by default. To analyze them recursively, append `/...` to their path.

## Default Linters

golangci-lint works without configuration and enables five linters by default:

- **errcheck** - Detects unchecked errors
- **govet** - Examines suspicious constructs (supports auto-fix)
- **ineffassign** - Identifies unused assignments (fast)
- **staticcheck** - Applies staticcheck rules (supports auto-fix)
- **unused** - Finds unused code elements

## Enabling and Disabling Linters

Control which linters run using command-line flags:

Enable specific linters with default linters disabled:
```bash
golangci-lint run --default=none -E errcheck
```

Enable additional linters:
```bash
golangci-lint run -E gosec -E gocritic
```

Disable specific linters:
```bash
golangci-lint run -D errcheck
```

Use `-E/--enable` to activate linters and `-D/--disable` to deactivate them.

## Code Formatting

Format your code with the fmt command:

```bash
golangci-lint fmt
```

Target specific directories or files:

```bash
golangci-lint fmt ./pkg/... main.go
```

## Viewing Configuration

See which configuration file is being used:

```bash
golangci-lint run -v
```

List enabled linters:

```bash
golangci-lint linters
```

View all available linters:

```bash
golangci-lint help linters
```

## Next Steps

- Configure linters in `.golangci.yml` for persistent settings
- Explore the [complete list of linters](linters.md) to enable additional checks
- Learn about [configuration options](configuration-file.md) for fine-tuning
