# Error Handling

## Basic Error Handling

```go
err := schema.Validate(instance)
if err != nil {
    fmt.Println("Validation failed:", err.Error())
}
```

## Introspectable Validation Errors

Validation errors provide detailed information with hierarchical error causes:

```go
err := schema.Validate(instance)
if err != nil {
    ve, ok := err.(*jsonschema.ValidationError)
    if !ok {
        // Not a validation error (compilation/schema error)
        log.Fatal(err)
    }

    // Access error details
    fmt.Println("Schema URL:", ve.SchemaURL)
    fmt.Println("Instance Location:", strings.Join(ve.InstanceLocation, "/"))
    fmt.Println("Error Kind:", ve.ErrorKind)
    fmt.Println("Message:", ve.Error())

    // Iterate through child errors (nested validation failures)
    for _, cause := range ve.Causes {
        fmt.Printf("  - %s: %s\n",
            strings.Join(cause.InstanceLocation, "/"),
            cause.Error())
    }
}
```

## Error Output Formats

The `ValidationError` type supports multiple output formats:

```go
ve := err.(*jsonschema.ValidationError)

// Basic output (simple format)
fmt.Println(ve.BasicOutput())

// Detailed output (verbose format)
fmt.Println(ve.DetailedOutput())

// Flag output (boolean flags)
fmt.Println(ve.FlagOutput())

// Localized errors (i18n support)
p := message.NewPrinter(language.English)
fmt.Println(ve.LocalizedError(p))
```

## Error Types

- `ValidationError` - Instance validation failure (introspectable)
- `SchemaValidationError` - Schema itself is invalid
- `CompilationError` - Schema compilation failure
- `AnchorNotFoundError` - Invalid anchor reference
- `InvalidRegexError` - Invalid pattern regex
- `LoadURLError` - Failed to load schema URL
- `UnsupportedDraftError` - Unsupported $schema version

## Collect All Validation Errors

```go
func collectErrors(ve *jsonschema.ValidationError, errs *[]string) {
    *errs = append(*errs, fmt.Sprintf("%s: %s",
        strings.Join(ve.InstanceLocation, "/"),
        ve.Error()))

    for _, cause := range ve.Causes {
        collectErrors(cause, errs)
    }
}

var allErrors []string
if err := schema.Validate(inst); err != nil {
    if ve, ok := err.(*jsonschema.ValidationError); ok {
        collectErrors(ve, &allErrors)
    }
}
```
