# Available Linters

golangci-lint includes 100+ linters for Go code analysis. This document categorizes them by purpose.

## Default Linters

These linters are enabled by default:

- **errcheck** - Checks for unchecked errors in Go code
- **gosimple** (via staticcheck) - Simplification suggestions
- **govet** - Examines Go source code and reports suspicious constructs (autofix)
- **ineffassign** - Detects when assignments to existing variables are not used (fast)
- **staticcheck** - Applies a collection of static analysis checks (autofix)
- **unused** - Checks for unused constants, variables, functions and types

## Error Handling

- **errcheck** - Unchecked errors (default)
- **errchkjson** - Checks types passed to JSON encoding functions
- **errname** - Checks that sentinel errors are prefixed with Err and error types are suffixed with Error
- **errorlint** - Finds code that will cause problems with Go 1.13 error wrapping (autofix)
- **nilerr** - Finds code returning nil even when it checks that error is not nil
- **wrapcheck** - Checks that errors from external packages are wrapped

## Code Quality & Complexity

- **cyclop** - Checks function and package cyclomatic complexity
- **funlen** - Detects long functions
- **gocognit** - Computes and checks cognitive complexity
- **gocyclo** - Computes and checks cyclomatic complexity
- **maintidx** - Measures maintainability index
- **nestif** - Reports deeply nested if statements

## Best Practices

- **containedctx** - Detects struct fields with context.Context
- **contextcheck** - Checks if functions use non-inherited context
- **exhaustive** - Checks exhaustiveness of enum switch statements (autofix)
- **exhaustruct** - Checks if all struct fields are initialized
- **ireturn** - Enforces "accept interfaces, return concrete types"
- **noctx** - Finds sending HTTP requests without context.Context
- **unparam** - Reports unused function parameters

## Security

- **gosec** - Inspects source code for security problems
- **bidichk** - Checks for dangerous unicode character sequences
- **bodyclose** - Checks whether HTTP response bodies are closed successfully

## Style & Formatting

- **godot** - Checks if comments end with a period (autofix)
- **godox** - Detects FIXME, TODO and other comment keywords
- **goheader** - Checks if file headers match a pattern (autofix)
- **misspell** - Finds commonly misspelled English words (autofix)
- **nlreturn** - Requires newline before return (autofix)
- **whitespace** - Detects leading and trailing whitespace (autofix)
- **wsl** - Enforces empty lines at the right places (autofix)

## Performance

- **durationcheck** - Checks for multiplication of two durations
- **makezero** - Finds slice declarations with non-zero initial length
- **prealloc** - Finds slice declarations that could potentially be pre-allocated

## Code Patterns

- **asasalint** - Checks for pass []any as any in variadic func(...any)
- **copyloopvar** - Detects places where loop variables are copied (autofix)
- **dupl** - Reports duplicated code
- **goconst** - Finds repeated strings that could be constants
- **gocritic** - Provides diagnostics that check for bugs, performance and style issues (autofix)
- **interfacebloat** - Checks the number of methods in interfaces
- **unconvert** - Removes unnecessary type conversions

## Import Management

- **depguard** - Checks if package imports are allowed
- **gci** - Controls package import order (autofix, formatter)
- **goimports** - Manages imports and formatting (autofix, formatter)
- **importas** - Enforces consistent import aliases (autofix)

## Documentation

- **godot** - Checks comment formatting (autofix)
- **revive** - Fast, configurable, extensible linter with many rules (autofix)

## Testing

- **ginkgolinter** - Enforces standards of using ginkgo and gomega (autofix)
- **testableexamples** - Checks if examples are testable
- **testifylint** - Checks usage of github.com/stretchr/testify (autofix)
- **thelper** - Detects test helpers without t.Helper() call
- **tparallel** - Detects inappropriate usage of t.Parallel()

## Naming Conventions

- **errname** - Checks error naming conventions
- **predeclared** - Finds code that shadows predeclared identifiers
- **revive** - Includes naming convention rules (autofix)
- **stylecheck** - Replacement for golint, enforces style guide
- **varnamelen** - Checks variable name length

## SQL

- **execinquery** - Checks query string in Query function
- **rowserrcheck** - Checks whether Err of rows is checked successfully
- **sqlclosecheck** - Checks that sql.Rows and sql.Stmt are closed

## Concurrency

- **govet** - Includes loopclosure and other concurrency checks
- **intrange** - Finds places where int range over arrays/slices can be simplified (autofix)

## Deprecated Detection

- **staticcheck** - Includes deprecation checks (SA1019)
- **govet** - Can detect deprecated usage

## All Linters by Name (Alphabetical)

- asasalint
- asciicheck
- bidichk
- bodyclose
- canonicalheader (autofix)
- containedctx
- contextcheck
- copyloopvar (autofix)
- cyclop
- decorder (autofix)
- depguard
- dogsled
- dupl
- dupword (autofix)
- durationcheck
- err113 (autofix)
- errcheck (default)
- errchkjson
- errname
- errorlint (autofix)
- execinquery
- exhaustive (autofix)
- exhaustruct
- exportloopref (deprecated, use copyloopvar)
- fatcontext (autofix)
- forbidigo
- forcetypeassert
- funlen
- gci (autofix, formatter)
- ginkgolinter (autofix)
- gocheckcompilerdirectives
- gochecknoglobals
- gochecknoinits
- gochecksumtype
- gocognit
- goconst
- gocritic (autofix)
- gocyclo
- godot (autofix)
- godox
- gofmt (formatter, autofix)
- gofumpt (formatter, autofix)
- goheader (autofix)
- goimports (formatter, autofix)
- golines (formatter, autofix)
- gomoddirectives
- gomodguard
- goprintffuncname
- gosec
- gosimple (default, via staticcheck)
- gosmopolitan
- govet (default, autofix)
- grouper
- iface (autofix)
- importas (autofix)
- ineffassign (default, fast)
- inamedparam
- interfacebloat
- intrange (autofix)
- ireturn
- lll
- loggercheck
- maintidx
- makezero
- mirror (autofix)
- misspell (autofix)
- mnd
- musttag
- nakedret (autofix)
- nestif
- nilerr
- nilnil
- nlreturn (autofix)
- noctx
- nolintlint (autofix)
- nonamedreturns
- nosprintfhostport
- paralleltest
- perfsprint (autofix)
- prealloc
- predeclared
- promlinter
- protogetter (autofix)
- reassign
- revive (autofix)
- rowserrcheck
- sloglint (autofix)
- spancheck
- sqlclosecheck
- staticcheck (default, autofix)
- stylecheck
- swaggo (formatter, autofix)
- tagalign (autofix)
- tagliatelle
- tenv
- testableexamples
- testifylint (autofix)
- testpackage
- thelper
- tparallel
- unconvert
- unparam
- unused (default)
- usestdlibvars (autofix)
- usetesting (autofix)
- varnamelen
- wastedassign
- whitespace (autofix)
- wrapcheck
- wsl (autofix)
- zerologlint

## Autofix-Capable Linters

27 linters support automatic fixing:

canonicalheader, copyloopvar, dupword, err113, errorlint, exptostd, fatcontext, ginkgolinter, gocritic, godot, goheader, govet, iface, importas, intrange, mirror, misspell, nakedret, nlreturn, nolintlint, perfsprint, protogetter, revive, sloglint, staticcheck, tagalign, testifylint, usestdlibvars, usetesting, whitespace, wsl

Use `--fix` flag to apply fixes:

```bash
golangci-lint run --fix
```

## Fast Linters

These linters are optimized for speed and suitable for frequent use:

- errcheck
- gosimple
- govet
- ineffassign
- staticcheck
- typecheck
- unused

Enable only fast linters:

```bash
golangci-lint run --fast-only
```

## Commands

View all available linters:

```bash
golangci-lint help linters
```

List currently enabled linters:

```bash
golangci-lint linters
```

List in JSON format:

```bash
golangci-lint linters --json
```

## References

- [Official linters documentation](https://golangci-lint.run/docs/linters/)
- [Linter configuration guide](linter-configuration.md)
