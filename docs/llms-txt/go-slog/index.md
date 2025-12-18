# Go log/slog Package

> Structured logging package for Go 1.21+ providing type-safe key-value logging with configurable handlers (Text, JSON), dynamic log levels, and zero-allocation value types.

This documentation covers Go's standard library structured logging package, designed for high-performance applications requiring structured log output with minimal overhead.

## Getting Started

- [Basic Usage](basic-usage.md): Quick start with slog.Info, slog.Debug, slog.Warn, slog.Error
- [Logger Creation](logger-creation.md): Creating and configuring custom loggers

## Core Concepts

- [Structured Logging](structured-logging.md): Key-value pairs and attributes
- [Handlers](handlers.md): TextHandler, JSONHandler, and custom handler implementation
- [Logger Configuration](logger-configuration.md): Setting defaults, log levels, and options
- [Attributes and Values](attributes-values.md): Creating type-safe structured data

## API Reference

- [Logger Type](api-logger.md): Logger methods and creation functions
- [Handler Interface](api-handler.md): Handler interface and built-in implementations
- [Attr and Value Types](api-attr-value.md): Type constructors and methods
- [Level and Record Types](api-level-record.md): Log levels and record structure

## Advanced Topics

- [Best Practices](best-practices.md): Performance optimization and usage patterns
- [Context Support](context-support.md): Using context.Context for tracing
- [Custom Handlers](custom-handlers.md): Implementing custom handler logic

## Optional

- [Constants and Variables](reference-constants.md): Package constants and global variables
- [Source Type](reference-source.md): Source code location tracking
