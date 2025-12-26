# Frequently Asked Questions

Common questions and troubleshooting for golangci-lint.

## Supported Go Versions

**Q: Which Go versions does golangci-lint support?**

A: golangci-lint supports the same versions as the Go team: the 2 latest minor versions. The tool only supports Go versions equal to or lower than the version used to compile golangci-lint, as newer Go versions may require linter adaptations.

## CI/CD Integration

**Q: How do I use golangci-lint in CI?**

A: Run golangci-lint and check the exit code. A non-zero exit code indicates issues were found and should fail the build:

```bash
golangci-lint run
if [ $? -ne 0 ]; then
    echo "Linting failed"
    exit 1
fi
```

Or simply:

```bash
golangci-lint run || exit 1
```

See the [CI installation documentation](https://golangci-lint.run/docs/welcome/install/ci/) for platform-specific setup.

## Typecheck Errors

**Q: I'm getting typecheck errors. What are they?**

A: Typecheck errors aren't from a dedicated linter but represent compilation errors. Your code must compile successfully before golangci-lint can analyze it. These errors block most linters from functioning.

**Q: Can I skip typecheck errors?**

A: No. Typecheck errors prevent most linters from working because they need type information. You must fix compilation errors first.

**Common solutions:**

1. Verify Go version compatibility:
   ```bash
   go version
   ```

2. Update dependencies:
   ```bash
   go mod tidy
   ```

3. Test compilation:
   ```bash
   go build ./...
   ```

4. Check for:
   - CGO dependencies
   - Build tags
   - Git configuration for private repositories
   - Missing or outdated dependencies

## Performance

**Q: Why is the first run slow even with --fast-only?**

A: The first run builds a type information cache. Subsequent runs reuse this cache and are much faster.

**Q: How can I speed up golangci-lint?**

A: Several strategies:

1. Use `--fast-only` for quick checks:
   ```bash
   golangci-lint run --fast-only
   ```

2. Reduce concurrency if memory-constrained:
   ```yaml
   run:
     concurrency: 2
   ```

3. Enable specific linters instead of all:
   ```yaml
   linters:
     default: none
     enable:
       - errcheck
       - govet
       - staticcheck
   ```

4. Exclude vendor and generated files:
   ```yaml
   run:
     skip-dirs:
       - vendor
       - generated
   ```

## Large Projects

**Q: My project has thousands of existing issues. How do I integrate golangci-lint gradually?**

A: Only check new code using one of these options:

**Check only new issues:**
```bash
golangci-lint run --new
```

**Check new issues from merge base:**
```bash
golangci-lint run --new-from-merge-base=main
```

**Check new issues from specific revision:**
```bash
golangci-lint run --new-from-rev=HEAD~1
```

**Note:** These commands compare git diff output against reported issues. Use `--whole-files` if issues aren't detected on modified lines:

```bash
golangci-lint run --new-from-merge-base=main --whole-files
```

## Configuration

**Q: Which configuration file is being used?**

A: Use verbose mode to see the config file path:

```bash
golangci-lint run -v
```

Or check directly:

```bash
golangci-lint config path
```

**Q: How do I validate my configuration?**

A: Use the verify command:

```bash
golangci-lint config verify
```

**Q: Why aren't my configuration changes taking effect?**

A: Common issues:

1. Wrong file location - golangci-lint searches from current directory upward
2. Syntax errors - validate with `golangci-lint config verify`
3. Cache issues - clear cache with `golangci-lint cache clean`
4. File name - must be `.golangci.yml`, `.golangci.yaml`, `.golangci.toml`, or `.golangci.json`

## Linter Issues

**Q: A linter is reporting false positives. How do I disable it?**

A: Several options:

1. Disable the linter entirely:
   ```yaml
   linters:
     disable:
       - lintername
   ```

2. Exclude specific issues:
   ```yaml
   issues:
     exclude-rules:
       - text: "specific error text"
         linters:
           - lintername
   ```

3. Use inline comments:
   ```go
   var x = "value" //nolint:lintername
   ```

See [false-positives.md](false-positives.md) for detailed guidance.

**Q: How do I enable all linters?**

A: Use the `all` preset:

```yaml
linters:
  default: all
```

Or via CLI:

```bash
golangci-lint run --default=all
```

**Q: Can I create custom linters?**

A: Yes. Use the custom command:

```bash
golangci-lint custom
```

See the [contributing documentation](https://golangci-lint.run/docs/contributing/new-linters/) for details.

## Output Formatting

**Q: How do I change the output format?**

A: Use the output configuration:

```yaml
output:
  formats:
    - format: json
      path: report.json
    - format: html
      path: report.html
```

Or via CLI:

```bash
golangci-lint run --output.json.path=report.json
```

Available formats: text, json, tab, html, checkstyle, code-climate, junit-xml, teamcity, sarif

## Cache

**Q: How do I clear the cache?**

A: Use the cache clean command:

```bash
golangci-lint cache clean
```

**Q: Where is the cache stored?**

A: Check cache status to see the location:

```bash
golangci-lint cache status
```

## Auto-fix

**Q: Can golangci-lint automatically fix issues?**

A: Yes. 27 linters support auto-fix:

```bash
golangci-lint run --fix
```

For formatting only:

```bash
golangci-lint fmt
```

**Q: Which linters support auto-fix?**

A: See the [linters documentation](linters.md) for the complete list. Notable ones include:

- govet
- staticcheck
- errorlint
- godot
- gofmt
- goimports
- misspell

## Exclude Files

**Q: How do I exclude specific files or directories?**

A: Several options:

1. In configuration:
   ```yaml
   run:
     skip-dirs:
       - vendor
       - generated

     skip-files:
       - ".*\\.pb\\.go$"
   ```

2. Using exclusion patterns:
   ```yaml
   issues:
     exclude-rules:
       - path: ".*_test\\.go"
         linters:
           - errcheck
   ```

## Error Messages

**Q: What does "no go files to analyze" mean?**

A: golangci-lint couldn't find Go files in the specified path. Check that:

1. You're in the correct directory
2. Go files exist in the path
3. Files aren't excluded by skip-dirs or skip-files

**Q: What does "context deadline exceeded" mean?**

A: Analysis timed out. Increase the timeout:

```yaml
run:
  timeout: 10m
```

Or via CLI:

```bash
golangci-lint run --timeout=10m
```

## IDE Integration

**Q: How do I integrate with my IDE?**

A: golangci-lint integrates with:

- VS Code (via Go extension)
- GoLand/IntelliJ
- Vim (via ale, syntastic, or coc.nvim)
- Emacs (via flycheck)
- Sublime Text

See the [integrations documentation](https://golangci-lint.run/docs/welcome/integrations/) for setup instructions.

## Updating

**Q: How do I update golangci-lint?**

A: Depends on installation method:

**Binary/script:**
```bash
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2
```

**Homebrew:**
```bash
brew upgrade golangci-lint
```

**Go install (not recommended):**
```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

## Help

**Q: Where can I get more help?**

A: Resources:

- Official documentation: https://golangci-lint.run/
- GitHub issues: https://github.com/golangci/golangci-lint/issues
- Built-in help: `golangci-lint help`
- Command-specific help: `golangci-lint help <command>`

## References

- [FAQ documentation](https://golangci-lint.run/docs/welcome/faq/)
- [Installation guide](installation.md)
- [Configuration guide](configuration-file.md)
