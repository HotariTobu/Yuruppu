# CLI Commands and Flags

## Primary Commands

### run

Executes linters and formatters (does not format code by default).

```bash
golangci-lint run [flags] [paths...]
```

**Key flags:**
- `-c, --config <path>` - Path to configuration file
- `-E, --enable <linters>` - Enable specific linters
- `-D, --disable <linters>` - Disable specific linters
- `--fix` - Apply detected fixes automatically
- `--timeout <duration>` - Analysis timeout (e.g., 5m)
- `--tests` - Include test files (default: true)
- `-j, --concurrency <n>` - Number of parallel threads

**Output options:**
- `--output.text.path <file>` - Write text output to file
- `--output.json.path <file>` - Write JSON output to file
- `--output.html.path <file>` - Write HTML output to file

**Filtering:**
- `--new` - Show only new issues
- `--new-from-rev <rev>` - Show new issues from git revision
- `--new-from-merge-base <branch>` - Show new issues from merge base
- `--fast-only` - Run only fast linters
- `--whole-files` - Show issues in whole files with --new-from-xxx

### fmt

Formats Go source files.

```bash
golangci-lint fmt [flags] [paths...]
```

**Flags:**
- `-c, --config <path>` - Path to configuration file
- `-E, --enable <formatters>` - Enable specific formatters
- `-d, --diff` - Show diff instead of rewriting files
- `--diff-colored` - Show colored diff
- `--stdin` - Read from stdin

### linters

Lists current linter configuration.

```bash
golangci-lint linters [flags]
```

**Flags:**
- `-c, --config <path>` - Path to configuration file
- `-E, --enable <linters>` - Enable specific linters
- `-D, --disable <linters>` - Disable specific linters
- `--enable-only <linters>` - Enable only specified linters
- `--fast-only` - List only fast linters
- `--json` - Output in JSON format

### formatters

Lists current formatter configuration.

```bash
golangci-lint formatters [flags]
```

**Flags:**
- `-c, --config <path>` - Path to configuration file
- `-E, --enable <formatters>` - Enable specific formatters
- `--json` - Output in JSON format

## Administrative Commands

### cache

Manages the analysis cache.

```bash
golangci-lint cache clean    # Clear cache
golangci-lint cache status   # Show cache status
```

### config

Handles configuration verification.

```bash
golangci-lint config path     # Show config file path
golangci-lint config verify   # Validate config file
```

### migrate

Converts v1 to v2 configuration files.

```bash
golangci-lint migrate [config-file]
```

### custom

Builds golangci-lint with custom linters.

```bash
golangci-lint custom
```

### version

Displays version information.

```bash
golangci-lint version
```

### completion

Generates shell autocompletion scripts.

```bash
golangci-lint completion bash      # Bash completion
golangci-lint completion fish      # Fish completion
golangci-lint completion powershell # PowerShell completion
golangci-lint completion zsh       # Zsh completion
```

### help

Provides additional documentation.

```bash
golangci-lint help [command]
```

## Global Flags

These flags work with all commands:

- `--color <when>` - Colorize output (always/auto/never)
- `-h, --help` - Show help
- `-v, --verbose` - Enable verbose output

## Exit Codes

- `0` - No issues found
- `1` - Issues found or error occurred

Use the exit code in CI/CD to fail builds when issues are detected.
