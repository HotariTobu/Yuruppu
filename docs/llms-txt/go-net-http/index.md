# Go net/http Package

> The net/http package provides HTTP client and server implementations for Go. This documentation focuses on building HTTP servers, handling requests/responses, and implementing middleware patterns.

This documentation is optimized for LLM consumption and covers the essential components for building HTTP servers in Go. The package supports HTTP/1.x and HTTP/2 with transparent HTTPS support.

## Getting Started

- [Quick Start](getting-started.md): Basic server setup and handler registration

## Core Concepts

- [HTTP Server](server.md): Server configuration, lifecycle, and graceful shutdown
- [Handlers](handlers.md): Handler interface, HandlerFunc, and implementation patterns
- [Request Handling](request.md): Request type, form parsing, cookies, and context
- [Response Writing](response.md): ResponseWriter interface and response construction

## Patterns and Best Practices

- [Middleware](middleware.md): Middleware patterns and handler wrapping
- [Routing](routing.md): ServeMux and URL pattern matching
- [Common Patterns](common-patterns.md): File serving, redirects, error handling

## Reference

- [Constants](constants.md): HTTP methods, status codes, and headers
- [Helper Functions](helpers.md): Built-in utilities for common tasks

## Optional

- [Advanced Features](advanced.md): HTTP/2, connection hijacking, server push, and streaming
- [Client Usage](client.md): HTTP client functionality (out of scope for server focus)
