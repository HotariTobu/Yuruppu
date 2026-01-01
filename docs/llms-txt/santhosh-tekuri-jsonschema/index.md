# santhosh-tekuri/jsonschema

> Go library for validating JSON data against JSON Schema specifications. Supports Draft 4, 6, 7, 2019-09, and 2020-12. Passes JSON-Schema-Test-Suite with high compliance rates.

**Version:** v6.0.2
**Import:** `github.com/santhosh-tekuri/jsonschema/v6`

This library provides robust JSON Schema validation with detailed error reporting, custom format validators, and support for content assertions. Key features include cycle detection, vocabulary-based validation, and introspectable validation errors.

## Getting Started

- [Quick Start](quick-start.md): Installation and basic usage examples
- [Schema Compilation](schema-compilation.md): How to compile schemas from different sources

## Core Concepts

- [Validation](validation.md): Validating data with compiled schemas
- [Error Handling](error-handling.md): Understanding and working with validation errors
- [Draft Versions](draft-versions.md): Working with different JSON Schema draft versions
- [Thread Safety](thread-safety.md): Using schemas and compilers safely in concurrent code

## Advanced Features

- [Format Assertions](format-assertions.md): Built-in and custom format validators
- [Content Assertions](content-assertions.md): Validating content encoding and media types
- [Custom Vocabularies](custom-vocabularies.md): Extending the library with custom keywords
- [Custom Loaders](custom-loaders.md): Custom URL loaders and regex engines

## API Reference

- [Main Types](main-types.md): Schema, Compiler, and ValidationError types
- [CLI Tool](cli-tool.md): Command-line validation tool usage

## Optional

- [Common Patterns](common-patterns.md): Example patterns for common use cases
- [Important Notes](important-notes.md): Critical information and best practices
