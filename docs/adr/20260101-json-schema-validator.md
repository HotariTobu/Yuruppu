# ADR: JSON Schema Validator for Function Calling

> Date: 2026-01-01
> Status: **Adopted**

<!--
ADR records decisions only. Do NOT add:
- Configuration examples or code snippets
- Version numbers
- Setup instructions or commands
-->

## Context

The LINE bot needs to validate LLM function calling parameters. When an LLM invokes a tool/function, it returns parameters as `map[string]any`. These parameters must be validated against JSON Schema to ensure type safety and catch malformed responses before processing.

## Decision Drivers

- **Validate map[string]any directly**: Must validate Go native types without JSON re-marshaling
- **Lightweight**: Minimal dependencies preferred for a LINE bot deployment
- **Correctness**: Must accurately validate against JSON Schema specification
- **Performance**: Fast validation for real-time bot responses
- **Draft support**: Draft-07 or later for modern schema features

## Options Considered

- **Option 1:** santhosh-tekuri/jsonschema
- **Option 2:** kaptinlin/jsonschema
- **Option 3:** google/jsonschema-go
- **Option 4:** xeipuuv/gojsonschema

## Evaluation

See `evaluation-criteria.md` for criteria definitions.

| Criterion | Weight | santhosh-tekuri | kaptinlin | google | xeipuuv |
|-----------|--------|-----------------|-----------|--------|---------|
| Functional Fit | 25% | 5 (1.25) | 5 (1.25) | 4 (1.00) | 4 (1.00) |
| Go Compatibility | 20% | 5 (1.00) | 4 (0.80) | 5 (1.00) | 4 (0.80) |
| Lightweight | 15% | 5 (0.75) | 3 (0.45) | 5 (0.75) | 4 (0.60) |
| Security | 15% | 4 (0.60) | 5 (0.75) | 4 (0.60) | 3 (0.45) |
| Documentation | 15% | 3 (0.45) | 4 (0.60) | 3 (0.45) | 4 (0.60) |
| Ecosystem | 10% | 3 (0.30) | 3 (0.30) | 2 (0.20) | 2 (0.20) |
| **Total** | 100% | **4.35** | **4.15** | **4.00** | **3.65** |

## Decision

Adopt **santhosh-tekuri/jsonschema**.

## Rationale

1. **Zero dependencies**: Uses only Go stdlib, meeting the lightweight requirement
2. **Best performance**: ~15.3µs/op with 5KB memory per operation (benchmarked 2x faster than xeipuuv)
3. **High correctness**: Near-perfect JSON Schema Test Suite compliance (only 1 edge case failure in Draft-07)
4. **Direct map[string]any support**: `Validate(any)` method accepts Go native types directly
5. **Active maintenance**: Regular releases with responsive maintainer
6. **Comprehensive draft support**: Supports Draft 4, 6, 7, 2019-09, and 2020-12

## Consequences

**Positive:**
- No additional dependencies added to the project
- Fast validation suitable for real-time bot responses
- Thread-safe for concurrent validation operations
- Rich error reporting with JSON pointers to exact locations

**Negative:**
- Smaller community (1.2k stars) compared to some alternatives
- Documentation is adequate but not comprehensive

**Risks:**
- Single maintainer project. Mitigation: Code is stable and well-tested; vendoring provides fallback

## Confirmation (optional)

- Validation errors from LLM outputs are caught before processing
- No dependency bloat in go.mod

## Related Decisions

- (None yet)

## Resources

| Option | Documentation | Repository |
|--------|---------------|------------|
| santhosh-tekuri/jsonschema | [pkg.go.dev](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6) | [GitHub](https://github.com/santhosh-tekuri/jsonschema) |
| kaptinlin/jsonschema | [pkg.go.dev](https://pkg.go.dev/github.com/kaptinlin/jsonschema) | [GitHub](https://github.com/kaptinlin/jsonschema) |
| google/jsonschema-go | [pkg.go.dev](https://pkg.go.dev/github.com/google/jsonschema-go) | [GitHub](https://github.com/google/jsonschema-go) |
| xeipuuv/gojsonschema | [pkg.go.dev](https://pkg.go.dev/github.com/xeipuuv/gojsonschema) | [GitHub](https://github.com/xeipuuv/gojsonschema) |

## Sources

- [Benchmarking Go JSON Schema validators (DEV Community)](https://dev.to/vearutop/benchmarking-correctness-and-performance-of-go-json-schema-validators-3247)
- [go-json-schema-bench](https://github.com/swaggest/go-json-schema-bench)
