# ADR: Constructor Naming Convention

> Date: 2025-12-28
> Status: **Adopted**

## Context

Go packages in this project need a consistent constructor naming convention. The standard Go idiom is to use `New()` when a package exports a single primary type, or `NewTypeName()` when multiple types exist.

This project has packages with varying numbers of exported types:
- Single type packages: `gcp` (MetadataClient), `llm` (Provider)
- Multiple type packages: `line` (Client and Server)

## Decision Drivers

- Consistency across all packages
- Idiomatic Go naming conventions
- Clear API when using qualified imports (e.g., `pkg.New()`)
- Avoid confusion when a package has multiple constructors

## Options Considered

- **Option 1:** Use `New()` for all packages, split multi-type packages into subpackages
- **Option 2:** Use `NewTypeName()` for all packages (e.g., `gcp.NewMetadataClient()`)
- **Option 3:** Mixed approach based on package size

## Decision

Adopt **Option 1**: Use `New()` consistently across all packages.

For packages that would have multiple constructors, split them into subpackages:
- `internal/line/client/` with `client.New()` → returns `*Client`
- `internal/line/server/` with `server.New()` → returns `*Server`

Shared types remain in the parent package:
- `internal/line/` contains `ConfigError` (shared error type)
- `internal/line/server/` contains `Handler` interface (server-specific)

## Rationale

- **Consistency**: Every package uses the same `New()` pattern
- **Clarity**: `client.New()` and `server.New()` are clearer than `line.NewClient()` and `line.NewServer()`
- **Separation of concerns**: Subpackages enforce clear boundaries between client and server functionality
- **Idiomatic Go**: Follows the pattern used by standard library (e.g., `bufio.NewReader()` in a multi-type package vs `ring.New()` in a single-type package)

## Consequences

**Positive:**
- Predictable API: developers always call `pkg.New()`
- Clear package boundaries for LINE client vs server
- Easier to understand each package's responsibility

**Negative:**
- Slightly deeper import paths for LINE packages
- Parent `line` package only contains types, no constructors
- Subpackages import parent for shared types (unusual but valid pattern)

## Package Structure

```
internal/
├── agent/          → agent.New()
├── gcp/            → gcp.New()
├── line/
│   ├── types.go    → ConfigError (shared error type)
│   ├── client/     → client.New()
│   └── server/     → server.New(), Handler interface
├── llm/            → llm.New()
└── yuruppu/        → yuruppu.New()
```

## Related Decisions

- [20251217-project-structure.md](./20251217-project-structure.md)
